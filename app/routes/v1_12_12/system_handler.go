package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetSystemInfo(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.system.Info())
}
