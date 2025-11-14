package v1_13_0

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetSubscriptions returns all subscriptions
func (h *Handler) GetSubscriptions(ctx context.Context, c *app.RequestContext) {
	subscriptions, err := h.subscriptionManager.List()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"subscriptions": subscriptions,
	})
}

// GetSubscriptionByID returns a subscription by ID
func (h *Handler) GetSubscriptionByID(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	sub, err := h.subscriptionManager.Get(id)
	if err != nil {
		c.JSON(consts.StatusNotFound, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, sub)
}

// AddSubscription adds a new subscription
func (h *Handler) AddSubscription(ctx context.Context, c *app.RequestContext) {
	var sub subscription.Subscription
	if err := c.Bind(&sub); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if sub.Name == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "name is required",
		})
		return
	}

	if sub.URL == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "url is required",
		})
		return
	}

	if err := h.subscriptionManager.Add(sub); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"message": "subscription added successfully",
		"id":      sub.ID,
	})
}

// UpdateSubscription updates an existing subscription
func (h *Handler) UpdateSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	var sub subscription.Subscription
	if err := c.Bind(&sub); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.subscriptionManager.Update(id, sub); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "subscription updated successfully",
		"id":      id,
	})
}

// DeleteSubscription deletes a subscription
func (h *Handler) DeleteSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	if err := h.subscriptionManager.Delete(id); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
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
		c.JSON(consts.StatusNotFound, utils.H{
			"error": err.Error(),
		})
		return
	}

	// Fetch subscription content
	resp, err := http.Get(sub.URL)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to fetch subscription: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "subscription server returned error: " + resp.Status,
		})
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to read subscription response: " + err.Error(),
		})
		return
	}

	// Parse nodes using the existing sublink package
	lines := strings.Split(string(body), "\n")
	nodes, err := h.sublink.ListNodes(lines)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to parse subscription nodes: " + err.Error(),
		})
		return
	}

	// Update last_update timestamp
	if err := h.subscriptionManager.UpdateLastUpdate(id); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to update subscription timestamp: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message":    "subscription updated successfully",
		"id":         id,
		"node_count": len(nodes),
		"nodes":      nodes,
	})
}
