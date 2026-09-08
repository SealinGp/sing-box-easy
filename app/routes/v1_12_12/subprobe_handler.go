package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetProbeStatus(ctx context.Context, c *app.RequestContext) {
	result, err := h.subscriptions.GetProbeStatus(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetProbeNodes(ctx context.Context, c *app.RequestContext) {
	result, err := h.subscriptions.GetProbeNodes(ctx, c.Param("id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) RunProbe(ctx context.Context, c *app.RequestContext) {
	result, err := h.subscriptions.RunProbe(ctx, c.Param("id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetProbeHistory(ctx context.Context, c *app.RequestContext) {
	result, err := h.subscriptions.GetProbeHistory(ctx, c.Param("id"), string(c.Query("range")))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateProbeSettings(ctx context.Context, c *app.RequestContext) {
	var req subscription.ProbeSettingsCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.subscriptions.UpdateProbeSettings(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
