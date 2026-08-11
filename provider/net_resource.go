package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type netResource struct {
	client *arin.Client
}

func newNetResource() resource.Resource { return &netResource{} }

// netBlockType mirrors the nested net block attributes for value
// building, a framework list carries plan-time unknowns which a plain
// slice cannot
var netBlockType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"start_address": types.StringType,
	"end_address":   types.StringType,
	"cidr_length":   types.Int64Type,
	"description":   types.StringType,
	"type":          types.StringType,
}}

type netModel struct {
	Handle           types.String `tfsdk:"handle"`
	NetName          types.String `tfsdk:"net_name"`
	Comment          types.List   `tfsdk:"comment"`
	OrgHandle        types.String `tfsdk:"org_handle"`
	ParentNetHandle  types.String `tfsdk:"parent_net_handle"`
	Version          types.Int64  `tfsdk:"version"`
	RegistrationDate types.String `tfsdk:"registration_date"`
	OriginASes       types.Set    `tfsdk:"origin_ases"`
	NetBlocks        types.List   `tfsdk:"net_blocks"`
}

func (r *netResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_net"
}

func (r *netResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func netBlockAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"start_address": schema.StringAttribute{Computed: true},
		"end_address":   schema.StringAttribute{Computed: true},
		"cidr_length":   schema.Int64Attribute{Computed: true},
		"description":   schema.StringAttribute{Computed: true},
		"type":          schema.StringAttribute{Computed: true},
	}
}

func (r *netResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An existing registry NET adopted by import. Only net_name and comment are updatable; allocation, reassignment, and removal stay in ARIN Online.",
		Attributes: map[string]schema.Attribute{
			"handle": schema.StringAttribute{
				Required:      true,
				Description:   "NET handle, the import identity.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"net_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Network name.",
			},
			"comment": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Public comment lines. Per RFC 9092 the geofeed URL lives here as a line of the form: Geofeed https://example.com/geofeed.csv",
			},
			"org_handle":        schema.StringAttribute{Computed: true},
			"parent_net_handle": schema.StringAttribute{Computed: true},
			"version":           schema.Int64Attribute{Computed: true},
			"registration_date": schema.StringAttribute{Computed: true},
			"origin_ases": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"net_blocks": schema.ListNestedAttribute{
				Computed:     true,
				NestedObject: schema.NestedAttributeObject{Attributes: netBlockAttrs()},
			},
		},
	}
}

func (m *netModel) refresh(n *arin.Net) {
	m.Handle = types.StringValue(n.Handle)
	m.NetName = types.StringValue(n.NetName)
	m.Comment = mergeList(m.Comment, n.Comment.Strings())
	m.OrgHandle = types.StringValue(n.OrgHandle)
	m.ParentNetHandle = types.StringValue(n.ParentNetHandle)
	m.Version = types.Int64Value(n.Version)
	m.RegistrationDate = types.StringValue(n.RegistrationDate)
	var ases []string
	if n.OriginASes != nil {
		ases = n.OriginASes.ASes
	}
	m.OriginASes = toSet(ases)
	if m.OriginASes.IsNull() {
		m.OriginASes = toSet([]string{})
	}
	blocks := []attr.Value{}
	if n.NetBlocks != nil {
		for _, b := range n.NetBlocks.Blocks {
			blocks = append(blocks, types.ObjectValueMust(netBlockType.AttrTypes, map[string]attr.Value{
				"start_address": types.StringValue(canon(b.StartAddress)),
				"end_address":   types.StringValue(canon(b.EndAddress)),
				"cidr_length":   types.Int64Value(b.CIDRLength),
				"description":   types.StringValue(b.Description),
				"type":          types.StringValue(b.Type),
			}))
		}
	}
	m.NetBlocks = types.ListValueMust(netBlockType, blocks)
}

func (r *netResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"nets cannot be created",
		"arin allocates nets through ticketed requests, adopt an existing net with terraform import instead",
	)
}

func (r *netResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var handle types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("handle"), &handle)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.Net(ctx, handle.ValueString())
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("reading net", err.Error())
		return
	}
	// imported state has only the handle, the model is rebuilt from
	// server state alone in that case
	state := netModel{Handle: handle}
	var name types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("net_name"), &name)...)
	if !name.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	state.refresh(got)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *netResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan netModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// fetch the current object and overlay only the safe fields so
	// server controlled fields round-trip untouched
	cur, err := r.client.Net(ctx, plan.Handle.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("reading net before update", err.Error())
		return
	}
	// unknown plan values appear when the prior state predates a read,
	// for example an apply with -refresh=false right after import, and
	// they mean keep the server's value
	payload := *cur
	if !plan.NetName.IsNull() && !plan.NetName.IsUnknown() {
		payload.NetName = plan.NetName.ValueString()
	}
	if !plan.Comment.IsUnknown() {
		payload.Comment = arin.MakeLines(fromList(ctx, plan.Comment, &resp.Diagnostics))
	}
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.NetUpdate(ctx, plan.Handle.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("updating net", err.Error())
		return
	}
	plan.refresh(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *netResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// tier 2 adoption only, removing from state leaves the net alone
	resp.Diagnostics.AddWarning(
		"net remains registered",
		"arin_net only adopts existing nets, removing it from state does not return the allocation to arin",
	)
}

func (r *netResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("handle"), req, resp)
}
