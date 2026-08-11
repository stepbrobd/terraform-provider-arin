package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// unknownOnUpdate marks a computed attribute unknown whenever the
// planned resource differs from prior state
// arin reissues a roa on every change, so every server-assigned value
// is recomputed on update and keeping the stale known value would make
// terraform reject the apply result
type unknownOnUpdate struct{}

func (unknownOnUpdate) Description(context.Context) string {
	return "recomputed when any attribute changes"
}

func (m unknownOnUpdate) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (unknownOnUpdate) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if updating(req.Plan.Raw, req.State.Raw) {
		resp.PlanValue = types.StringUnknown()
	}
}

func (unknownOnUpdate) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if updating(req.Plan.Raw, req.State.Raw) {
		resp.PlanValue = types.Int64Unknown()
	}
}

func (unknownOnUpdate) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if updating(req.Plan.Raw, req.State.Raw) {
		resp.PlanValue = types.BoolUnknown()
	}
}

// updating is true only for updates in place
// create has null state and destroy has a null plan
func updating(plan, state tftypes.Value) bool {
	return !state.IsNull() && !plan.IsNull() && !plan.Equal(state)
}
