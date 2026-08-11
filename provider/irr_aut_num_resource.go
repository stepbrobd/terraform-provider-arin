package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type irrAutNumResource struct {
	client *arin.Client
}

func newIRRAutNumResource() resource.Resource { return &irrAutNumResource{} }

type irrAutNumModel struct {
	ASNumber     types.Int64  `tfsdk:"as_number"`
	ASName       types.String `tfsdk:"as_name"`
	Descriptions types.List   `tfsdk:"descriptions"`
	AdminPOCs    types.Set    `tfsdk:"admin_pocs"`
	TechPOCs     types.Set    `tfsdk:"tech_pocs"`
	RoutingPOCs  types.Set    `tfsdk:"routing_pocs"`
	Remarks      types.List   `tfsdk:"remarks"`
	Imports      types.List   `tfsdk:"imports"`
	Exports      types.List   `tfsdk:"exports"`
	Defaults     types.List   `tfsdk:"defaults"`
	MPImports    types.List   `tfsdk:"mp_imports"`
	MPExports    types.List   `tfsdk:"mp_exports"`
	MPDefaults   types.List   `tfsdk:"mp_defaults"`
	MemberOf     types.Set    `tfsdk:"member_of"`
	OrgHandle    types.String `tfsdk:"org_handle"`
	Created      types.String `tfsdk:"created"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *irrAutNumResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_irr_aut_num"
}

func (r *irrAutNumResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *irrAutNumResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	strList := func(desc string) schema.ListAttribute {
		return schema.ListAttribute{Optional: true, ElementType: types.StringType, Description: desc}
	}
	resp.Schema = schema.Schema{
		Description: "An IRR aut-num object carrying the AS description and routing policy.",
		Attributes: map[string]schema.Attribute{
			"as_number": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"as_name": schema.StringAttribute{Required: true},
			"descriptions": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
			},
			"admin_pocs": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"tech_pocs": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"routing_pocs": schema.SetAttribute{Optional: true, ElementType: types.StringType},
			"remarks":      strList("Remarks lines."),
			"imports":      strList("import policy lines."),
			"exports":      strList("export policy lines."),
			"defaults":     strList("default policy lines."),
			"mp_imports":   strList("mp-import policy lines."),
			"mp_exports":   strList("mp-export policy lines."),
			"mp_defaults":  strList("mp-default policy lines."),
			"member_of": schema.SetAttribute{
				Computed:      true,
				ElementType:   types.StringType,
				Description:   "Parent as-set names. Server derived.",
				PlanModifiers: []planmodifier.Set{unknownOnUpdate{}},
			},
			"org_handle":    schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"created":       schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
			"last_modified": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{unknownOnUpdate{}}},
		},
	}
}

func (m *irrAutNumModel) object(ctx context.Context, org string, diags *diag.Diagnostics) arin.IRRAutNum {
	admin := fromSet(ctx, m.AdminPOCs, diags)
	tech := fromSet(ctx, m.TechPOCs, diags)
	routing := fromSet(ctx, m.RoutingPOCs, diags)
	return arin.IRRAutNum{
		ASNumber:    asString(m.ASNumber.ValueInt64()),
		ASName:      m.ASName.ValueString(),
		Description: arin.MakeLines(fromList(ctx, m.Descriptions, diags)),
		Remarks:     arin.MakeLines(fromList(ctx, m.Remarks, diags)),
		PocLinks:    pocRefs(admin, tech, routing),
		Imports:     arin.MakeLines(fromList(ctx, m.Imports, diags)),
		Exports:     arin.MakeLines(fromList(ctx, m.Exports, diags)),
		Defaults:    arin.MakeLines(fromList(ctx, m.Defaults, diags)),
		MPImports:   arin.MakeLines(fromList(ctx, m.MPImports, diags)),
		MPExports:   arin.MakeLines(fromList(ctx, m.MPExports, diags)),
		MPDefaults:  arin.MakeLines(fromList(ctx, m.MPDefaults, diags)),
		OrgHandle:   org,
		Source:      "ARIN",
	}
}

func (m *irrAutNumModel) refresh(a *arin.IRRAutNum, diags *diag.Diagnostics) {
	m.ASNumber = types.Int64Value(asNumber(a.ASNumber, diags))
	m.ASName = types.StringValue(a.ASName)
	m.Descriptions = toList(a.Description.Strings())
	m.Remarks = toList(a.Remarks.Strings())
	admin, tech, routing := pocSplit(a.PocLinks)
	m.AdminPOCs = toSet(admin)
	m.TechPOCs = toSet(tech)
	m.RoutingPOCs = toSet(routing)
	m.Imports = toList(a.Imports.Strings())
	m.Exports = toList(a.Exports.Strings())
	m.Defaults = toList(a.Defaults.Strings())
	m.MPImports = toList(a.MPImports.Strings())
	m.MPExports = toList(a.MPExports.Strings())
	m.MPDefaults = toList(a.MPDefaults.Strings())
	m.MemberOf = toSet(names(a.MemberOf))
	if m.MemberOf.IsNull() {
		m.MemberOf = toSet([]string{})
	}
	m.OrgHandle = types.StringValue(a.OrgHandle)
	m.Created = types.StringValue(a.Created)
	m.LastModified = types.StringValue(a.Modified)
}

func (r *irrAutNumResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan irrAutNumModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.AutNumCreate(ctx, plan.ASNumber.ValueInt64(), plan.object(ctx, r.client.Org(), &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("creating aut-num", err.Error())
		return
	}
	plan.refresh(created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *irrAutNumResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var as types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("as_number"), &as)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.AutNum(ctx, as.ValueInt64())
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("reading aut-num", err.Error())
		return
	}
	state := irrAutNumModel{ASNumber: as}
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

func (r *irrAutNumResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan irrAutNumModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.AutNumUpdate(ctx, plan.ASNumber.ValueInt64(), plan.object(ctx, r.client.Org(), &resp.Diagnostics))
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("updating aut-num", err.Error())
		return
	}
	plan.refresh(updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *irrAutNumResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state irrAutNumModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.AutNumDelete(ctx, state.ASNumber.ValueInt64())
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("deleting aut-num", err.Error())
	}
}

func (r *irrAutNumResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	as, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", fmt.Sprintf("as number expected, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("as_number"), as)...)
}
