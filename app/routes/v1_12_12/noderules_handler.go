package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetNodeRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetNodeRules(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetFilters(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetFilters(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) CreateFilter(ctx context.Context, c *app.RequestContext) {
	var req outbounds.FilterRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.outbounds().CreateFilter(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateFilter(ctx context.Context, c *app.RequestContext) {
	var req outbounds.FilterRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.outbounds().UpdateFilter(ctx, c.Param("id"), req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteFilter(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().DeleteFilter(ctx, c.Param("id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetGroups(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetGroups(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) CreateGroup(ctx context.Context, c *app.RequestContext) {
	var req outbounds.GroupRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.outbounds().CreateGroup(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateGroup(ctx context.Context, c *app.RequestContext) {
	var req outbounds.GroupRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.outbounds().UpdateGroup(ctx, c.Param("id"), req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DeleteGroup(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().DeleteGroup(ctx, c.Param("id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetNodeRuleKeywords(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetNodeRuleKeywords(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetNodeRuleTemplates(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().GetNodeRuleTemplates(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) ApplyNodeRuleTemplate(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().ApplyNodeRuleTemplate(ctx, c.Param("id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) PreviewNodeRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().PreviewNodeRules(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) ApplyNodeRules(ctx context.Context, c *app.RequestContext) {
	result, err := h.outbounds().ApplyNodeRules(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
