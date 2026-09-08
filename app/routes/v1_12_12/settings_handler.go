package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetSettings(ctx context.Context, c *app.RequestContext) {
	result, err := h.settingsService().GetSettings(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateSettings(ctx context.Context, c *app.RequestContext) {
	var req settings.UpdateSettingsCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.settingsService().UpdateSettings(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetSubscriptionInfoKeywords(ctx context.Context, c *app.RequestContext) {
	result, err := h.subscriptions.GetSubscriptionInfoKeywords(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateSubscriptionInfoKeywords(ctx context.Context, c *app.RequestContext) {
	var req subscription.UpdateSubscriptionInfoKeywordsCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.subscriptions.UpdateSubscriptionInfoKeywords(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
