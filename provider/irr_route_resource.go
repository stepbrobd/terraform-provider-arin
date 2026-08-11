package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type irrRouteResource struct {
	client *arin.Client
}

func newIRRRouteResource() resource.Resource { return &irrRouteResource{} }

type irrRouteModel struct {
	Prefix              types.String `tfsdk:"prefix"`
	OriginAS            types.Int64  `tfsdk:"origin_as"`
	Descriptions        types.List   `tfsdk:"descriptions"`
	AdminPOCs           types.Set    `tfsdk:"admin_pocs"`
	TechPOCs            types.Set    `tfsdk:"tech_pocs"`
	RoutingPOCs         types.Set    `tfsdk:"routing_pocs"`
	Remarks             types.List   `tfsdk:"remarks"`
	MemberOf            types.Set    `tfsdk:"member_of"`
	IPVersion           types.Int64  `tfsdk:"ip_version"`
	NetHandle           types.String `tfsdk:"net_handle"`
	AutoLinkedROAHandle types.String `tfsdk:"auto_linked_roa_handle"`
	OrgHandle           types.String `tfsdk:"org_handle"`
	Created             types.String `tfsdk:"created"`
	LastModified        types.String `tfsdk:"last_modified"`
}

func (r *irrRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_route"
}

func (r *irrRouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *irrRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An IRR route or route6 object. Routes maintained by ARIN for auto_link ROAs are refused; manage those through the ROA instead.",
		Attributes: map[string]schema.Attribute{
			"prefix": schema.StringAttribute{
				Required:      true,
				Description:   "CIDR prefix, either address family.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"origin_as": schema.Int64Attribute{
				Required:      true,
				Description:   "Origin AS number. ARIN forbids changing it in place.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"descriptions": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				Description: "descr lines.",
			},
			"admin_pocs": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "Admin POC handles.",
			},
			"tech_pocs": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "Tech POC handles.",
			},
			"routing_pocs": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Routing POC handles.",
			},
			"remarks": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Remarks lines.",
			},
			"member_of": schema.SetAttribute{
				Computed:      true,
				ElementType:   types.StringType,
				Description:   "Route-set names referencing this route. Server derived.",
				PlanModifiers: []planmodifier.Set{unknownOnUpdate{}},
			},
			"ip_version":             schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{unknownOnUpdate{}}},
			"net_handle":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"auto_linked_roa_handle": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"org_handle":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"created":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"last_modified":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
		},
	}
}

// object converts the configured model into the wire shape
func (m *irrRouteModel) object(ctx context.Context, org string, diags *diag.Diagnostics) arin.IRRRoute {
	admin := fromSet(ctx, m.AdminPOCs, diags)
	tech := fromSet(ctx, m.TechPOCs, diags)
	routing := fromSet(ctx, m.RoutingPOCs, diags)
	return arin.IRRRoute{
		Prefix:      m.Prefix.ValueString(),
		OriginAS:    asString(m.OriginAS.ValueInt64()),
		Description: arin.MakeLines(fromList(ctx, m.Descriptions, diags)),
		Remarks:     arin.MakeLines(fromList(ctx, m.Remarks, diags)),
		PocLinks:    pocRefs(admin, tech, routing),
		OrgHandle:   org,
		Source:      "ARIN",
	}
}

// refresh overwrites the model from the wire shape, preserving the
// configured textual prefix when semantically unchanged
func (m *irrRouteModel) refresh(r *arin.IRRRoute, diags *diag.Diagnostics) {
	if !samePrefix(m.Prefix.ValueString(), r.Prefix) {
		m.Prefix = types.StringValue(canonPrefix(r.Prefix))
	}
	m.OriginAS = types.Int64Value(asNumber(r.OriginAS, diags))
	m.Descriptions = toList(r.Description.Strings())
	m.Remarks = toList(r.Remarks.Strings())
	admin, tech, routing := pocSplit(r.PocLinks)
	m.AdminPOCs = toSet(admin)
	m.TechPOCs = toSet(tech)
	m.RoutingPOCs = toSet(routing)
	m.MemberOf = toSet(names(r.MemberOf))
	if m.MemberOf.IsNull() {
		m.MemberOf = toSet([]string{})
	}
	m.IPVersion = types.Int64Value(r.Version)
	m.NetHandle = types.StringValue(r.NetHandle)
	m.AutoLinkedROAHandle = types.StringValue(r.AutoLinkedRoaHandle)
	m.OrgHandle = types.StringValue(r.OrgHandle)
	m.Created = types.StringValue(r.Created)
	m.LastModified = types.StringValue(r.Modified)
}

// guard rejects objects arin maintains for auto_link roas
func routeGuard(r *arin.IRRRoute) error {
	if r != nil && r.AutoLinkedRoaHandle != "" {
		return fmt.Errorf("route %s origin %s is auto-linked to roa %s and managed by arin; manage it through the roa's auto_link instead", r.Prefix, r.OriginAS, r.AutoLinkedRoaHandle)
	}
	return nil
}

func (r *irrRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan irrRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	prefix := plan.Prefix.ValueString()
	origin := plan.OriginAS.ValueInt64()
	// pre-create get so auto-linked objects fail with a clear error
	// instead of a duplicate-create error
	existing, err := r.client.Route(ctx, prefix, origin)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("checking for existing route", err.Error())
		return
	}
	if err == nil {
		if gerr := routeGuard(existing); gerr != nil {
			resp.Diagnostics.AddError("route is auto-linked", gerr.Error())
			return
		}
		resp.Diagnostics.AddError("route already exists", "import it instead of creating")
		return
	}
	created, err := r.client.RouteCreate(ctx, prefix, origin, plan.object(ctx, r.client.Org(), &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("creating route", err.Error())
		return
	}
	plan.refresh(created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *irrRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prefix types.String
	var origin types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("prefix"), &prefix)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("origin_as"), &origin)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.Route(ctx, prefix.ValueString(), origin.ValueInt64())
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("reading route", err.Error())
		return
	}
	if gerr := routeGuard(got); gerr != nil {
		resp.Diagnostics.AddError("route is auto-linked", gerr.Error())
		return
	}
	state := irrRouteModel{Prefix: prefix, OriginAS: origin}
	var descr types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("descriptions"), &descr)...)
	if !descr.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	state.refresh(got, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *irrRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan irrRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.RouteUpdate(ctx, plan.Prefix.ValueString(), plan.OriginAS.ValueInt64(), plan.object(ctx, r.client.Org(), &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("updating route", err.Error())
		return
	}
	plan.refresh(updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *irrRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state irrRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.RouteDelete(ctx, state.Prefix.ValueString(), state.OriginAS.ValueInt64())
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("deleting route", err.Error())
	}
}

func (r *irrRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idx := strings.LastIndex(req.ID, "/")
	if idx < 0 {
		resp.Diagnostics.AddError("invalid import id", "expected PREFIX/ORIGINAS, e.g. 192.0.2.0/24/64496")
		return
	}
	origin, err := strconv.ParseInt(req.ID[idx+1:], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", fmt.Sprintf("origin as: %v", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("prefix"), req.ID[:idx])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("origin_as"), origin)...)
}
