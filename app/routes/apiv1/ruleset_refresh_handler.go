package apiv1

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) RefreshRuleSet(ctx context.Context, c *app.RequestContext) {
	var request struct {
		Tag string `json:"tag"`
	}
	if err := c.BindJSON(&request); err != nil || strings.TrimSpace(request.Tag) == "" {
		respErr(ctx, c, CodeBadRequest, "rule-set tag is required")
		return
	}
	result, err := h.serviceController.RefreshRuleSet(ctx, request.Tag)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
