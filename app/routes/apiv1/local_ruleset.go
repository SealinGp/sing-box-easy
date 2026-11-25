package apiv1

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) ReadLocalRuleFile(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().ReadLocalRuleFile(string(c.Query("tag")))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) WriteLocalRuleFile(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().WriteLocalRuleFile(ctx, c.Request.Body())
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
