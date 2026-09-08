package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/cloudwego/hertz/pkg/app"
)

type RouteProbeRequest = diagnostics.RouteProbeRequest

func (h *Handler) ProbeRoute(ctx context.Context, c *app.RequestContext) {
	var req RouteProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.diagnostics().ProbeRoute(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
