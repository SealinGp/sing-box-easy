package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetInitStatus(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().GetInitStatus(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) CompleteInit(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().CompleteInit(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) ResetInit(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().ResetInit(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
