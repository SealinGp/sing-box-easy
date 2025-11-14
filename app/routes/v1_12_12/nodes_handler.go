package v1_13_0

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ParseNodes parses base64 encoded nodes/subscription
func (h *Handler) ParseNodes(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Subscription string `json:"subscription"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if req.Subscription == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "subscription is required",
		})
		return
	}

	// Split by newlines to support multiple nodes
	lines := strings.Split(req.Subscription, "\n")

	// Parse nodes using sublink package
	nodes, err := h.sublink.ListNodes(lines)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to parse nodes: " + err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message":    "nodes parsed successfully",
		"node_count": len(nodes),
		"nodes":      nodes,
	})
}
