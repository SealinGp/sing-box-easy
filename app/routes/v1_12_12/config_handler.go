package v1_13_0

import (
	"context"

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

// RollbackConfig restores the backup configuration
func (h *Handler) RollbackConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configManager.Rollback(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "configuration rolled back successfully",
	})
}
