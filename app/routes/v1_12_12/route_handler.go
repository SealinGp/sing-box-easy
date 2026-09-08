package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetRouteRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetRouteRules(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddRouteRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().AddRouteRule(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) ReorderRouteRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().ReorderRouteRules(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateRouteRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateRouteRule(ctx, c.Request.Body(), c.Param("index"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteRouteRule(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().DeleteRouteRule(ctx, c.Param("index"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetRuleSets(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetRuleSets(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetRuleSetByTag(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetRuleSetByTag(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) AddRuleSet(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().AddRuleSet(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateRuleSet(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateRuleSet(ctx, c.Request.Body(), c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetRuleSetReferences(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetRuleSetReferences(ctx, c.Param("tag"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteRuleSet(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().DeleteRuleSet(ctx, c.Param("tag"), string(c.Query("cascade")))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetRouteFinal(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().GetRouteFinal(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateRouteFinal(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().UpdateRouteFinal(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
