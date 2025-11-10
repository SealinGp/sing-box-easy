package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetLog returns the log configuration
func (h *Handler) GetLog(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.Log == nil {
		c.JSON(consts.StatusOK, &config.LogConfig{})
		return
	}

	c.JSON(consts.StatusOK, cfg.Log)
}

// UpdateLog updates the log configuration
func (h *Handler) UpdateLog(ctx context.Context, c *app.RequestContext) {
	var logConfig config.LogConfig
	if err := c.Bind(&logConfig); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		cfg.Log = &logConfig
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "log configuration updated successfully",
	})
}

// GetClashAPI returns the Clash API configuration
func (h *Handler) GetClashAPI(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.Experimental == nil || cfg.Experimental.ClashAPI == nil {
		c.JSON(consts.StatusOK, &config.ClashAPIConfig{})
		return
	}

	c.JSON(consts.StatusOK, cfg.Experimental.ClashAPI)
}

// UpdateClashAPI updates the Clash API configuration
func (h *Handler) UpdateClashAPI(ctx context.Context, c *app.RequestContext) {
	var clashAPIConfig config.ClashAPIConfig
	if err := c.Bind(&clashAPIConfig); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "Clash API configuration updated successfully",
	})
}

// GetCacheFile returns the cache file configuration
func (h *Handler) GetCacheFile(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.Experimental == nil || cfg.Experimental.CacheFile == nil {
		c.JSON(consts.StatusOK, &config.CacheFileConfig{})
		return
	}

	c.JSON(consts.StatusOK, cfg.Experimental.CacheFile)
}

// UpdateCacheFile updates the cache file configuration
func (h *Handler) UpdateCacheFile(ctx context.Context, c *app.RequestContext) {
	var cacheFileConfig config.CacheFileConfig
	if err := c.Bind(&cacheFileConfig); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "cache file configuration updated successfully",
	})
}
