package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
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

	if !normalizeProbeURL(ctx, c, &sub) {
		return
	}

	// A client that did not mention probing gets it ON. Go decodes an absent
	// bool as false, so without this every subscription added by an older UI
	// build — or by curl — would be silently excluded from the quality chart
	// and nothing on screen would say why.
	if explicit := probeEnabledFromBody(c); explicit != nil {
		sub.ProbeEnabled = *explicit
	} else {
		sub.ProbeEnabled = true
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

	if !normalizeProbeURL(ctx, c, &sub) {
		return
	}

	// Update writes an explicit column list, so an absent probe_enabled would
	// be persisted as false. A partial update from any other API client must
	// not turn off a feature it never mentioned — keep what is stored.
	if explicit := probeEnabledFromBody(c); explicit != nil {
		sub.ProbeEnabled = *explicit
	} else if existing, err := h.subscriptionManager.Get(id); err == nil {
		sub.ProbeEnabled = existing.ProbeEnabled
	} else {
		// The read that would have told us the current value failed, so the
		// zero value (false) is about to be written for a field the client
		// never mentioned. Say so — silently turning probing off is exactly
		// the failure this whole read-back exists to prevent.
		logger.Warn("could not read existing probe_enabled; leaving it disabled for this update",
			zap.String("id", id), zap.Error(err))
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

// normalizeProbeURL validates the latency-test target, rejecting anything that
// is not https.
//
// Refused at the boundary rather than fixed up, because sing-box does the
// fixing up silently: it discards an http:// delay URL and substitutes its own
// default (see subscription.NormalizeProbeURL). Accepting one would leave the
// operator reading a configured target on screen while every measurement
// described a different endpoint.
func normalizeProbeURL(ctx context.Context, c *app.RequestContext, sub *subscription.Subscription) bool {
	normalized, err := subscription.NormalizeProbeURL(sub.ProbeURL)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return false
	}
	sub.ProbeURL = normalized
	return true
}

// DeleteSubscription deletes a subscription
func (h *Handler) DeleteSubscription(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")

	if err := h.subscriptionManager.Delete(id); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	// Drop the quality history with it. Subscription ids are minted from a
	// unix timestamp, so re-adding a feed almost always gets a new id — but
	// "almost always" is not a guarantee worth inheriting a stranger's
	// availability chart over, and the rows are dead weight regardless.
	if h.probeStore != nil {
		if err := h.probeStore.DeleteSubscription(id); err != nil {
			logger.Warn("failed to delete probe history for subscription",
				zap.String("id", id), zap.Error(err))
		}
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
