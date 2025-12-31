package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetLog returns the log configuration
func (h *Handler) GetLog(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Log == nil {
		respOK(ctx, c, &config.LogConfig{})
		return
	}

	respOK(ctx, c, cfg.Log)
}

// UpdateLog updates the log configuration
func (h *Handler) UpdateLog(ctx context.Context, c *app.RequestContext) {
	var logConfig config.LogConfig
	if err := c.Bind(&logConfig); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		cfg.Log = &logConfig
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "log configuration updated successfully"})
}

// GetClashAPI returns the Clash API configuration
func (h *Handler) GetClashAPI(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Experimental == nil || cfg.Experimental.ClashAPI == nil {
		respOK(ctx, c, &config.ClashAPIConfig{})
		return
	}

	respOK(ctx, c, cfg.Experimental.ClashAPI)
}

// UpdateClashAPI updates the Clash API configuration
func (h *Handler) UpdateClashAPI(ctx context.Context, c *app.RequestContext) {
	var clashAPIConfig config.ClashAPIConfig
	if err := c.Bind(&clashAPIConfig); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Experimental == nil {
			cfg.Experimental = &config.ExperimentalConfig{}
		}
		cfg.Experimental.ClashAPI = &clashAPIConfig
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "Clash API configuration updated successfully"})
}

// GetCacheFile returns the cache file configuration
func (h *Handler) GetCacheFile(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Experimental == nil || cfg.Experimental.CacheFile == nil {
		respOK(ctx, c, &config.CacheFileConfig{})
		return
	}

	respOK(ctx, c, cfg.Experimental.CacheFile)
}

// UpdateCacheFile updates the cache file configuration
func (h *Handler) UpdateCacheFile(ctx context.Context, c *app.RequestContext) {
	var cacheFileConfig config.CacheFileConfig
	if err := c.Bind(&cacheFileConfig); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Experimental == nil {
			cfg.Experimental = &config.ExperimentalConfig{}
		}
		cfg.Experimental.CacheFile = &cacheFileConfig
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "cache file configuration updated successfully"})
}

// GetV2RayAPI returns the V2Ray API configuration
func (h *Handler) GetV2RayAPI(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Experimental == nil || cfg.Experimental.V2RayAPI == nil {
		respOK(ctx, c, &config.V2RayAPIOptions{})
		return
	}

	respOK(ctx, c, cfg.Experimental.V2RayAPI)
}

// UpdateV2RayAPI updates the V2Ray API configuration
func (h *Handler) UpdateV2RayAPI(ctx context.Context, c *app.RequestContext) {
	var v2rayAPIConfig config.V2RayAPIOptions
	if err := c.Bind(&v2rayAPIConfig); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Experimental == nil {
			cfg.Experimental = &config.ExperimentalConfig{}
		}
		cfg.Experimental.V2RayAPI = &v2rayAPIConfig
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "V2Ray API configuration updated successfully"})
}
