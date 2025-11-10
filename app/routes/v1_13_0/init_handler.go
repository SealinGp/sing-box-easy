package v1_13_0

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetInitStatus returns initialization status
func (h *Handler) GetInitStatus(ctx context.Context, c *app.RequestContext) {
	state := h.initStateManager.GetState()

	c.JSON(consts.StatusOK, utils.H{
		"initialized": state.Initialized,
		"steps": utils.H{
			"sing_box_installed":   state.SingBoxInstalled,
			"config_generated":     state.ConfigGenerated,
			"dashboard_installed":  state.DashboardInstalled,
		},
		"sing_box_version": state.SingBoxVersion,
		"init_time":        state.InitTime,
	})
}

// CompleteInit marks initialization as complete
func (h *Handler) CompleteInit(ctx context.Context, c *app.RequestContext) {
	if err := h.initStateManager.CompleteInitialization(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "initialization completed successfully",
	})
}

// ResetInit resets initialization state
func (h *Handler) ResetInit(ctx context.Context, c *app.RequestContext) {
	if err := h.initStateManager.Reset(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "initialization state reset successfully",
	})
}
