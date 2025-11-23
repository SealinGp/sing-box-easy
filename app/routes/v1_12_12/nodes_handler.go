package v1_13_0

import (
	"context"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
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
	subNodes, err := h.sublink.ListNodes(lines)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to parse nodes: " + err.Error(),
		})
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

	resp := utils.H{
		"message":    "nodes parsed successfully",
		"node_count": len(outbounds),
		"nodes":      outbounds,
	}
	jsonCtx := config.CreateContext(ctx)
	responseJSON, err := json.MarshalContext(jsonCtx, resp)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to serialize DNS servers: " + err.Error(),
		})
		return
	}
	c.Data(consts.StatusOK, "application/json; charset=utf-8", responseJSON)
}
