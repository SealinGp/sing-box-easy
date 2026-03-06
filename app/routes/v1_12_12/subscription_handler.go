package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
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

	if err := h.subscriptionManager.Update(id, sub); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "subscription updated successfully",
		"id":      id,
	})
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

// UpdateSubscriptionContent fetches and updates nodes from a subscription
func (h *Handler) UpdateSubscriptionContent(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	// Get subscription
	sub, err := h.subscriptionManager.Get(id)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	// Parse nodes using the existing sublink package
	subNodes, err := h.sublink.ListNodes([]string{sub.URL})
	if err != nil {
		respErr(ctx, c, CodeInternalError, "failed to parse subscription nodes: "+err.Error())
		return
	}

	// Update last_update timestamp
	if err := h.subscriptionManager.UpdateLastUpdate(id); err != nil {
		respErr(ctx, c, CodeInternalError, "failed to update subscription timestamp: "+err.Error())
		return
	}

	// Convert SubNodes to option.Outbound for proper serialization
	outbounds := make([]option.Outbound, 0, len(subNodes))
	for _, subNode := range subNodes {
		outbound := option.Outbound{
			Tag:     subNode.Tag,
			Type:    subNode.Type,
			Options: subNode.Options,
		}
		outbounds = append(outbounds, outbound)
	}

	addedTags, skippedTags, err := h.configManager.UpdateOutbounds(outbounds)
	if err != nil {
		respErr(ctx, c, CodeInternalError, "failed to update outbounds: "+err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message":      "subscription updated successfully",
		"id":           id,
		"node_count":   len(addedTags),
		"added_tags":   addedTags,
		"skipped_tags": skippedTags,
	})
}
