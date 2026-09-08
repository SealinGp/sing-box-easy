package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetDNS(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetDNS(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateDNS(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateDNS(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDNSServers(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetDNSServers(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDNSServerByTag(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetDNSServerByTag(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddDNSServer(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().AddDNSServer(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateDNSServer(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateDNSServer(ctx, c.Request.Body(), c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteDNSServer(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().DeleteDNSServer(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDNSHosts(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetDNSHosts(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateDNSHosts(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateDNSHosts(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDNSRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetDNSRules(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddDNSRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().AddDNSRule(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) ReorderDNSRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().ReorderDNSRules(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateDNSRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateDNSRule(ctx, c.Request.Body(), c.Param("index"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteDNSRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().DeleteDNSRule(ctx, c.Param("index"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
