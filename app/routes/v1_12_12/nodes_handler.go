package v1_13_0

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
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

	// Split by newlines to support multiple nodes
	lines := strings.Split(req.Subscription, "\n")

	// Parse nodes using sublink package
	subNodes, err := h.sublink.ListNodes(lines)
	if err != nil {
		respErr(ctx, c, CodeInternalError, "failed to parse nodes: "+err.Error())
		return
	}

	// Convert SubNodes to option.Outbound
	outbounds := make([]option.Outbound, 0, len(subNodes))
	for _, subNode := range subNodes {
		outbound := option.Outbound{
			Tag:     subNode.Tag,
			Type:    subNode.Type,
			Options: subNode.Options,
		}
		outbounds = append(outbounds, outbound)
	}

	respOK(ctx, c, map[string]any{
		"message":    "nodes parsed successfully",
		"node_count": len(outbounds),
		"nodes":      outbounds,
	})
}
