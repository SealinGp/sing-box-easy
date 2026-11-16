package apiv1

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) RefreshRuleSet(ctx context.Context, c *app.RequestContext) {
	var request struct {
		Tag  string   `json:"tag"`
		Tags []string `json:"tags"`
	}
	if err := c.BindJSON(&request); err != nil || (strings.TrimSpace(request.Tag) == "" && len(request.Tags) == 0) || (request.Tag != "" && len(request.Tags) > 0) {
		respErr(ctx, c, CodeBadRequest, "rule-set tag is required")
		return
	}
	if len(request.Tags) > 0 {
		results, err := h.serviceController.RefreshRuleSets(ctx, request.Tags)
		if err != nil {
			respondOperationError(ctx, c, err)
			return
		}
		respOK(ctx, c, map[string]any{"results": results})
		return
	}
	result, err := h.serviceController.RefreshRuleSet(ctx, request.Tag)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
