package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetConfig returns the current configuration
func (h *Handler) GetConfig(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, cfg)
}

// ValidateConfig validates the provided configuration
func (h *Handler) ValidateConfig(ctx context.Context, c *app.RequestContext) {
	var cfg config.SingBoxConfig
	if err := c.Bind(&cfg); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.configManager.ValidateConfig(&cfg); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"valid":   true,
		"message": "configuration is valid",
	})
}

// GetBackupConfig returns the backup configuration
func (h *Handler) GetBackupConfig(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetBackupConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, cfg)
}

// RollbackConfig restores the backup configuration
func (h *Handler) RollbackConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configManager.Rollback(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "configuration rolled back successfully",
	})
}
