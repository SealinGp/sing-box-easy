package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetOutbounds(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetOutbounds(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetOutboundByTag(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetOutboundByTag(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddOutbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().AddOutbound(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddOutboundsBatch(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().AddOutboundsBatch(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateOutbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().UpdateOutbound(ctx, c.Request.Body(), c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteOutbound(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().DeleteOutbound(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteOutboundsBatch(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().DeleteOutboundsBatch(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetOutboundGroups(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetOutboundGroups(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateOutboundMembers(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().UpdateOutboundMembers(ctx, c.Request.Body(), c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
