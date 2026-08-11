package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type delegationResource struct {
	client *arin.Client
}

func newDelegationResource() resource.Resource { return &delegationResource{} }

type delegationKeyModel struct {
	Algorithm  types.Int64  `tfsdk:"algorithm"`
	Digest     types.String `tfsdk:"digest"`
	DigestType types.Int64  `tfsdk:"digest_type"`
	KeyTag     types.Int64  `tfsdk:"key_tag"`
	TTL        types.Int64  `tfsdk:"ttl"`
}

// delegationKeyType mirrors delegationKeyModel for value building
// a framework set carries the unknown a plan puts on unconfigured
// computed attributes, which a plain slice cannot
var delegationKeyType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"algorithm":   types.Int64Type,
	"digest":      types.StringType,
	"digest_type": types.Int64Type,
	"key_tag":     types.Int64Type,
	"ttl":         types.Int64Type,
}}

type delegationModel struct {
	Name        types.String `tfsdk:"name"`
	Nameservers types.Set    `tfsdk:"nameservers"`
	Keys        types.Set    `tfsdk:"delegation_keys"`
}

func (r *delegationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_delegation"
}

func (r *delegationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func delegationKeyAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"algorithm":   schema.Int64Attribute{Required: true, Description: "DNSSEC algorithm number."},
		"digest":      schema.StringAttribute{Required: true, Description: "DS digest, uppercase hex as ARIN stores it."},
		"digest_type": schema.Int64Attribute{Required: true, Description: "DS digest type number."},
		"key_tag":     schema.Int64Attribute{Required: true},
		"ttl":         schema.Int64Attribute{Required: true},
	}
}

func (r *delegationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A reverse DNS delegation, modified in place. Delegations exist per net and cannot be created or destroyed; nameserver and DS changes take up to 24 hours to reach the distributed DNS.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Zone name, without the trailing dot.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"nameservers": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "Nameserver host names, matched exactly as ARIN stores them.",
			},
			"delegation_keys": schema.SetNestedAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "DS records. Omit to leave the server set untouched, configure (or set empty) to manage it.",
				NestedObject:  schema.NestedAttributeObject{Attributes: delegationKeyAttrs()},
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// keysValue converts wire delegation keys into a set value
func keysValue(d *arin.Delegation, diags *diag.Diagnostics) types.Set {
	vals := []attr.Value{}
	if d.Keys != nil {
		for _, k := range d.Keys.Keys {
			vals = append(vals, types.ObjectValueMust(delegationKeyType.AttrTypes, map[string]attr.Value{
				"algorithm":   types.Int64Value(numValue(k.Algorithm.Value, "algorithm", diags)),
				"digest":      types.StringValue(k.Digest),
				"digest_type": types.Int64Value(numValue(k.DigestType.Value, "digestType", diags)),
				"key_tag":     types.Int64Value(k.KeyTag),
				"ttl":         types.Int64Value(k.TTL),
			}))
		}
	}
	return types.SetValueMust(delegationKeyType, vals)
}

// numValue parses arin's numeric chardata
func numValue(s, field string, diags *diag.Diagnostics) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		diags.AddError("unparseable "+field, fmt.Sprintf("%q: %v", s, err))
	}
	return n
}

// object converts the model into the wire shape
// the name gains the trailing dot arin uses in payloads
// a nil Keys means the keys were not configured and the caller must
// carry the server's current set forward
func (m *delegationModel) object(ctx context.Context, diags *diag.Diagnostics) arin.Delegation {
	name := m.Name.ValueString()
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	var keys *arin.DelegationKeys
	if !m.Keys.IsNull() && !m.Keys.IsUnknown() {
		var kms []delegationKeyModel
		diags.Append(m.Keys.ElementsAs(ctx, &kms, false)...)
		keys = &arin.DelegationKeys{}
		for _, k := range kms {
			keys.Keys = append(keys.Keys, arin.DelegationKey{
				Algorithm:  arin.NamedValue{Value: strconv.FormatInt(k.Algorithm.ValueInt64(), 10)},
				Digest:     k.Digest.ValueString(),
				DigestType: arin.NamedValue{Value: strconv.FormatInt(k.DigestType.ValueInt64(), 10)},
				KeyTag:     k.KeyTag.ValueInt64(),
				TTL:        k.TTL.ValueInt64(),
			})
		}
	}
	return arin.Delegation{
		Name:        name,
		Keys:        keys,
		Nameservers: fromSet(ctx, m.Nameservers, diags),
	}
}

func (m *delegationModel) refresh(d *arin.Delegation, diags *diag.Diagnostics) {
	m.Name = types.StringValue(strings.TrimSuffix(d.Name, "."))
	m.Nameservers = toSet(d.Nameservers)
	m.Keys = keysValue(d, diags)
}

func (r *delegationResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"delegations cannot be created",
		"arin derives delegations from net allocations, adopt the existing zone with terraform import instead",
	)
}

func (r *delegationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.Delegation(ctx, name.ValueString())
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("reading delegation", err.Error())
		return
	}
	state := delegationModel{Name: name}
	state.refresh(got, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *delegationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan delegationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload := plan.object(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if payload.Keys == nil {
		// keys unconfigured, carry the server's current set forward
		cur, err := r.client.Delegation(ctx, plan.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("reading delegation before update", err.Error())
			return
		}
		payload.Keys = cur.Keys
	}
	updated, err := r.client.DelegationUpdate(ctx, plan.Name.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("updating delegation", err.Error())
		return
	}
	plan.refresh(updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *delegationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// modify-only singleton, removing from state leaves the zone alone
	resp.Diagnostics.AddWarning(
		"delegation remains registered",
		"arin_delegation only adopts existing zones, removing it from state does not change the delegation at arin",
	)
}

func (r *delegationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
