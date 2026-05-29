package v1_13_0

import (
	"context"
	"errors"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetConfig returns the current configuration
func (h *Handler) GetConfig(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, cfg)
}

// UpdateConfig saves the provided configuration
func (h *Handler) UpdateConfig(ctx context.Context, c *app.RequestContext) {
	var cfg config.SingBoxConfig
	if err := c.Bind(&cfg); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.configManager.SaveConfig(&cfg); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "configuration saved successfully",
	})
}

// ValidateConfig validates the provided configuration
func (h *Handler) ValidateConfig(ctx context.Context, c *app.RequestContext) {
	var cfg config.SingBoxConfig
	if err := c.Bind(&cfg); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.configManager.ValidateConfig(&cfg); err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"valid":   true,
		"message": "configuration is valid",
	})
}

// GetBackupConfig returns the backup configuration
func (h *Handler) GetBackupConfig(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetBackupConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, cfg)
}

// RollbackConfig restores the most recent historical configuration.
func (h *Handler) RollbackConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configManager.Rollback(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "configuration rolled back successfully",
	})
}

// ListConfigVersions returns historical config metadata (newest first).
func (h *Handler) ListConfigVersions(ctx context.Context, c *app.RequestContext) {
	versions, err := h.configManager.ListVersions()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"versions": versions})
}

// GetConfigVersion returns the full config of a single historical version.
func (h *Handler) GetConfigVersion(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respErr(ctx, c, CodeBadRequest, "invalid version id")
		return
	}
	cfg, err := h.configManager.GetVersion(id)
	if err != nil {
		if errors.Is(err, config.ErrVersionNotFound) {
			respErr(ctx, c, CodeNotFound, err.Error())
		} else {
			respErr(ctx, c, CodeInternalError, err.Error())
		}
		return
	}
	respOK(ctx, c, cfg)
}

// RollbackToConfigVersion restores a specific historical version to live config.
func (h *Handler) RollbackToConfigVersion(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respErr(ctx, c, CodeBadRequest, "invalid version id")
		return
	}
	if err := h.configManager.RollbackToVersion(id); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{
		"message": "configuration rolled back to selected version",
	})
}
