package v1_13_0

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) GetLog(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().ReadSection(ctx, "log")
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateLog(ctx context.Context, c *app.RequestContext) {
	if err := h.configuration().ReplaceSection(ctx, "log", c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "log configuration updated successfully"})
}
func (h *Handler) GetClashAPI(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().Experimental(ctx, "clash_api")
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateClashAPI(ctx context.Context, c *app.RequestContext) {
	if err := h.configuration().ReplaceExperimental(ctx, "clash_api", c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "Clash API configuration updated successfully"})
}
func (h *Handler) GetCacheFile(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().Experimental(ctx, "cache_file")
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateCacheFile(ctx context.Context, c *app.RequestContext) {
	if err := h.configuration().ReplaceExperimental(ctx, "cache_file", c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "cache file configuration updated successfully"})
}
func (h *Handler) GetV2RayAPI(ctx context.Context, c *app.RequestContext) {
	result, err := h.configuration().Experimental(ctx, "v2ray_api")
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateV2RayAPI(ctx context.Context, c *app.RequestContext) {
	if err := h.configuration().ReplaceExperimental(ctx, "v2ray_api", c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "V2Ray API configuration updated successfully"})
}
