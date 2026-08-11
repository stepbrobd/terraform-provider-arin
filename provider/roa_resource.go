package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type roaResource struct {
	client *arin.Client
}

func newROAResource() resource.Resource { return &roaResource{} }

func (r *roaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roa"
}

func (r *roaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *roaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Route Origin Authorization under the provider org. Updates run as one atomic delete+add ARIN transaction and reissue the ROA under a new handle.",
		Attributes: map[string]schema.Attribute{
			"roa_handle": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned handle. Changes on every update.",
				PlanModifiers: []planmodifier.String{unknownOnUpdate{}},
			},
			"as_number": schema.Int64Attribute{
				Required:    true,
				Description: "Origin AS number.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Descriptive name for the ROA.",
			},
			"auto_link": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Create and maintain matching IRR route/route6 objects. ARIN does not report this back, so imported resources show the default.",
			},
			"auto_renewed": schema.BoolAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{unknownOnUpdate{}},
			},
			"not_valid_before": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{unknownOnUpdate{}},
			},
			"not_valid_after": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{unknownOnUpdate{}},
			},
			"resources": schema.ListNestedAttribute{
				Required:    true,
				Description: "Prefixes the AS may originate.",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start_address": schema.StringAttribute{
							Required:    true,
							Description: "First address of the prefix.",
						},
						"cidr_length": schema.Int64Attribute{
							Required:    true,
							Description: "Prefix length.",
						},
						"max_length": schema.Int64Attribute{
							Optional:    true,
							Description: "Longest prefix the AS may advertise.",
						},
						"end_address": schema.StringAttribute{
							Computed:      true,
							PlanModifiers: []planmodifier.String{unknownOnUpdate{}},
						},
						"ip_version": schema.Int64Attribute{
							Computed:      true,
							PlanModifiers: []planmodifier.Int64{unknownOnUpdate{}},
						},
						"auto_linked": schema.BoolAttribute{
							Computed:      true,
							PlanModifiers: []planmodifier.Bool{unknownOnUpdate{}},
						},
					},
				},
			},
		},
	}
}

// handles snapshots the org's current roa handles
func (r *roaResource) handles(ctx context.Context) (map[string]bool, error) {
	list, err := r.client.ROAs(ctx)
	if err != nil && !arin.IsNotFound(err) {
		return nil, err
	}
	out := make(map[string]bool, len(list))
	for i := range list {
		out[list[i].Handle] = true
	}
	return out, nil
}

// settle re-lists after a transaction and fills m from the new roa
// the listing can transiently fail or lag right behind the mutation,
// so identification retries briefly before giving up
func (r *roaResource) settle(ctx context.Context, m *roaModel, before map[string]bool) error {
	const attempts = 3
	var last error
	for i := range attempts {
		if i > 0 {
			select {
			case <-ctx.Done():
				return last
			case <-time.After(time.Second):
			}
		}
		after, err := r.client.ROAs(ctx)
		if err != nil {
			last = err
			continue
		}
		created, err := findNew(before, after, m)
		if err == nil {
			m.refresh(created)
			return nil
		}
		last = err
		// only a lagging listing can resolve by waiting
		if !errors.Is(err, errSettleNoMatch) {
			return err
		}
	}
	return last
}

func (r *roaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	before, err := r.handles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("listing roas before create", err.Error())
		return
	}
	tx := arin.Transaction{ROAAdds: &arin.ROAAdds{Specs: []arin.ROASpecRequest{plan.spec()}}}
	if err := r.client.Transact(ctx, tx); err != nil {
		resp.Diagnostics.AddError("creating roa", err.Error())
		return
	}
	if err := r.settle(ctx, &plan, before); err != nil {
		resp.Diagnostics.AddError(
			"identifying created roa",
			fmt.Sprintf("%s\n\nthe transaction succeeded, so the roa exists in arin without terraform tracking. find its handle with the arin_roas data source and adopt it with terraform import.", err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var handle types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("roa_handle"), &handle)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.ROAs(ctx)
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("listing roas", err.Error())
		return
	}
	var found *arin.ROASpec
	for i := range list {
		if list[i].Handle == handle.ValueString() {
			found = &list[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// imported state has every attribute except the handle null and a
	// null resources list cannot reflect into the model slice, so the
	// model is rebuilt from server state alone in that case
	var state roaModel
	var raw types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("resources"), &raw)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// imports leave auto_link null, align it with the schema default
	if state.AutoLink.IsNull() {
		state.AutoLink = types.BoolValue(false)
	}
	state.refresh(found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	before, err := r.handles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("listing roas before update", err.Error())
		return
	}
	old := state.ROAHandle.ValueString()
	delete(before, old)
	tx := arin.Transaction{
		ROADeletes: &arin.ROADeletes{Handles: []arin.ROAHandleRef{{
			AutoLink: state.AutoLink.ValueBool(),
			Handle:   old,
		}}},
		ROAAdds: &arin.ROAAdds{Specs: []arin.ROASpecRequest{plan.spec()}},
	}
	if err := r.client.Transact(ctx, tx); err != nil {
		resp.Diagnostics.AddError("updating roa", err.Error())
		return
	}
	if err := r.settle(ctx, &plan, before); err != nil {
		resp.Diagnostics.AddError(
			"identifying updated roa",
			fmt.Sprintf("%s\n\nthe transaction succeeded, roa %s was deleted and its replacement exists in arin without terraform tracking. refresh to drop the stale state entry, then find the new handle with the arin_roas data source and adopt it with terraform import.", err, old),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tx := arin.Transaction{ROADeletes: &arin.ROADeletes{Handles: []arin.ROAHandleRef{{
		AutoLink: state.AutoLink.ValueBool(),
		Handle:   state.ROAHandle.ValueString(),
	}}}}
	err := r.client.Transact(ctx, tx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("deleting roa", err.Error())
	}
}

func (r *roaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("roa_handle"), req, resp)
}
