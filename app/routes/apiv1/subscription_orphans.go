package apiv1

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetOrphanSubscriptions(ctx context.Context, c *app.RequestContext) {
	plan, err := h.subscriptions.DetectOrphanSubscriptions()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, plan)
}
func (h *Handler) CleanupOrphanSubscriptions(ctx context.Context, c *app.RequestContext) {
	var request struct {
		Revision string `json:"revision"`
	}
	if err := c.Bind(&request); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.subscriptions.CleanupOrphanSubscriptions(ctx, request.Revision)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
