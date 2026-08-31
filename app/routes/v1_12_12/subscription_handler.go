package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetSubscriptions returns all subscriptions
func (h *Handler) GetSubscriptions(ctx context.Context, c *app.RequestContext) {
	subscriptions, err := h.subscriptionManager.List()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"subscriptions": subscriptions})
}

// GetSubscriptionByID returns a subscription by ID
func (h *Handler) GetSubscriptionByID(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	sub, err := h.subscriptionManager.Get(id)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, sub)
}

// AddSubscription adds a new subscription
func (h *Handler) AddSubscription(ctx context.Context, c *app.RequestContext) {
	var sub subscription.Subscription
	if err := c.Bind(&sub); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if sub.Name == "" {
		respErr(ctx, c, CodeBadRequest, "name is required")
		return
	}

	if sub.URL == "" {
		respErr(ctx, c, CodeBadRequest, "url is required")
		return
	}

	if !normalizeOfficialURL(ctx, c, &sub) {
		return
	}

	if err := h.subscriptionManager.Add(sub); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "subscription added successfully",
		"id":      sub.ID,
	})
}

// UpdateSubscription updates an existing subscription
func (h *Handler) UpdateSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	var sub subscription.Subscription
	if err := c.Bind(&sub); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if !normalizeOfficialURL(ctx, c, &sub) {
		return
	}

	if err := h.subscriptionManager.Update(id, sub); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "subscription updated successfully",
		"id":      id,
	})
}

// normalizeOfficialURL validates and canonicalizes an operator-supplied site
// link in place, reporting the error itself. Returns false when the request has
// already been answered and the handler must stop.
//
// The field ends up in an href in the UI, so a scheme that can execute is
// refused at the boundary rather than filtered at render time — the value is
// also served to any other client of this API.
func normalizeOfficialURL(ctx context.Context, c *app.RequestContext, sub *subscription.Subscription) bool {
	if sub.OfficialURL == "" {
		return true
	}
	normalized := subscription.NormalizeOfficialURL(sub.OfficialURL)
	if normalized == "" {
		respErr(ctx, c, CodeBadRequest, "official_url must be an http(s) link")
		return false
	}
	sub.OfficialURL = normalized
	return true
}

// DeleteSubscription deletes a subscription
func (h *Handler) DeleteSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	if err := h.subscriptionManager.Delete(id); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "subscription deleted successfully",
		"id":      id,
	})
}

// UpdateSubscriptionContent fetches and updates nodes from a subscription.
// All business logic (fetch → diff → apply → stats) lives in the subscription
// package; this handler only translates the HTTP request/response envelope.
func (h *Handler) UpdateSubscriptionContent(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	result, err := h.autoUpdater.RefreshByID(id)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	added, updated, deleted := result.Counts()
	respOK(ctx, c, map[string]any{
		"message":      "subscription updated successfully",
		"id":           id,
		"added_tags":   result.AddedTags,
		"updated_tags": result.UpdatedTags,
		"deleted_keys": result.DeletedKeys,
		"added":        added,
		"updated":      updated,
		"deleted":      deleted,
	})
}
