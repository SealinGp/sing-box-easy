package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetInbounds(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetInbounds(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetInboundByTag(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetInboundByTag(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddInbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().AddInbound(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateInbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateInbound(ctx, c.Request.Body(), c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteInbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().DeleteInbound(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
