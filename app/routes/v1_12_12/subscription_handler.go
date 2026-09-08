package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetSubscriptions returns all subscriptions
func (h *Handler) GetSubscriptions(ctx context.Context, c *app.RequestContext) {
	subscriptions, err := h.subscriptions.List()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"subscriptions": subscriptions})
}

// GetSubscriptionByID returns a subscription by ID
func (h *Handler) GetSubscriptionByID(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	sub, err := h.subscriptions.Get(id)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, sub)
}

func (h *Handler) AddSubscription(ctx context.Context, c *app.RequestContext) {
	var cmd subscription.EditCommand
	if err := c.Bind(&cmd); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	sub, err := h.subscriptions.Create(cmd)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "subscription added successfully", "id": sub.ID})
}
func (h *Handler) UpdateSubscription(ctx context.Context, c *app.RequestContext) {
	var cmd subscription.EditCommand
	if err := c.Bind(&cmd); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	sub, err := h.subscriptions.Edit(c.Param("id"), cmd)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "subscription updated successfully", "id": sub.ID})
}

// DeleteSubscription delegates node cleanup and service restart to the subscription manager.
func (h *Handler) DeleteSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	if err := h.subscriptions.Delete(id); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "subscription deleted successfully",
		"id":      id,
	})
}

// UpdateSubscriptionContent fetches and updates nodes from a subscription.
// All business logic (fetch → diff → save → apply) lives in the subscription
// package; this handler only translates the HTTP request/response envelope.
func (h *Handler) UpdateSubscriptionContent(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	result, err := h.subscriptions.Refresh(ctx, id)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	added, updated, deleted := result.Counts()
	respOK(ctx, c, map[string]any{
		"message":      "subscription updated and sing-box restarted successfully",
		"id":           id,
		"added_tags":   result.AddedTags,
		"updated_tags": result.UpdatedTags,
		"deleted_keys": result.DeletedKeys,
		"added":        added,
		"updated":      updated,
		"deleted":      deleted,
		"restarted":    result.Restarted,
	})
}
