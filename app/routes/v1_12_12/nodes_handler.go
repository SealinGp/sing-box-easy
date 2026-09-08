package v1_13_0

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// ParseNodes parses base64 encoded nodes/subscription
func (h *Handler) ParseNodes(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Subscription string `json:"subscription"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Subscription == "" {
		respErr(ctx, c, CodeBadRequest, "subscription is required")
		return
	}

	outbounds, err := h.subscriptions.ImportPreview(ctx, req.Subscription)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, map[string]any{
		"message":    "nodes parsed successfully",
		"node_count": len(outbounds),
		"nodes":      outbounds,
	})
}
