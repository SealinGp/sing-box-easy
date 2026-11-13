package apiv1

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/trafficpolicy"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetTrafficPolicies(ctx context.Context, c *app.RequestContext) {
	out, err := h.trafficPolicies.Snapshot(ctx)
	if err != nil {
		respErr(ctx, c, CodeConflict, err.Error())
		return
	}
	respOK(ctx, c, out)
}
func (h *Handler) PlanTrafficPolicies(ctx context.Context, c *app.RequestContext) {
	var cmd trafficpolicy.Command
	if err := c.Bind(&cmd); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	out, err := h.trafficPolicies.Plan(ctx, cmd)
	if err != nil {
		respErr(ctx, c, CodeConflict, err.Error())
		return
	}
	respOK(ctx, c, out)
}
func (h *Handler) CommitTrafficPolicies(ctx context.Context, c *app.RequestContext) {
	var cmd trafficpolicy.Command
	if err := c.Bind(&cmd); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	if err := h.trafficPolicies.Commit(ctx, cmd); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"saved": true, "restart_required": true})
}
func (h *Handler) TrafficPolicyImpact(ctx context.Context, c *app.RequestContext) {
	out, err := h.trafficPolicies.ImpactPlan(ctx, c.Request.Body())
	if err != nil {
		respErr(ctx, c, CodeConflict, err.Error())
		return
	}
	respOK(ctx, c, out)
}
