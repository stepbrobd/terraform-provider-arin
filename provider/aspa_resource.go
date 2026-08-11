package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/stepbrobd/terraform-provider-arin/arin"
)

type aspaResource struct {
	client *arin.Client
}

func newASPAResource() resource.Resource { return &aspaResource{} }

type aspaModel struct {
	CustomerAS    types.Int64 `tfsdk:"customer_as"`
	ProviderASIDs []int64     `tfsdk:"provider_as_ids"`
}

// aspa converts the model into the arin request shape
func (m *aspaModel) aspa() arin.ASPA {
	ids := append([]int64(nil), m.ProviderASIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return arin.ASPA{CustomerASID: m.CustomerAS.ValueInt64(), ProviderASIDs: ids}
}

func (r *aspaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aspa"
}

func (r *aspaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *aspaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An Autonomous System Provider Authorization under the provider org.",
		Attributes: map[string]schema.Attribute{
			"customer_as": schema.Int64Attribute{
				Required:      true,
				Description:   "Customer AS number. This is the ASPA identity, changing it replaces the object.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"provider_as_ids": schema.SetAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
				Description: "Provider AS numbers authorized to propagate the customer's routes.",
			},
		},
	}
}

func (r *aspaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aspaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tx := arin.Transaction{ASPAAdds: &arin.ASPAAdds{ASPAs: []arin.ASPA{plan.aspa()}}}
	if err := r.client.Transact(ctx, tx); err != nil {
		resp.Diagnostics.AddError("creating aspa", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aspaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var customer types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("customer_as"), &customer)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.ASPAs(ctx)
	if arin.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("listing aspas", err.Error())
		return
	}
	for i := range list {
		if list[i].CustomerASID == customer.ValueInt64() {
			state := aspaModel{
				CustomerAS:    customer,
				ProviderASIDs: list[i].ProviderASIDs,
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *aspaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aspaModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// customer_as requires replace, so an update only changes providers
	// delete and add run in one atomic transaction
	tx := arin.Transaction{
		ASPADeletes: &arin.ASPADeletes{CustomerASIDs: []int64{plan.CustomerAS.ValueInt64()}},
		ASPAAdds:    &arin.ASPAAdds{ASPAs: []arin.ASPA{plan.aspa()}},
	}
	if err := r.client.Transact(ctx, tx); err != nil {
		resp.Diagnostics.AddError("updating aspa", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aspaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aspaModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tx := arin.Transaction{ASPADeletes: &arin.ASPADeletes{CustomerASIDs: []int64{state.CustomerAS.ValueInt64()}}}
	err := r.client.Transact(ctx, tx)
	if err != nil && !arin.IsNotFound(err) {
		resp.Diagnostics.AddError("deleting aspa", err.Error())
	}
}

func (r *aspaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	as, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("invalid import id", fmt.Sprintf("customer as number expected, got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("customer_as"), as)...)
}
