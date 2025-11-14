package v1_13_0

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetServiceStatus returns the current status of sing-box service
func (h *Handler) GetServiceStatus(ctx context.Context, c *app.RequestContext) {
	running, err := h.serviceController.Status()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	status := "stopped"
	if running {
		status = "running"
	}

	c.JSON(consts.StatusOK, utils.H{
		"status":  status,
		"running": running,
	})
}

// StartService starts the sing-box service
func (h *Handler) StartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Start(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "service started successfully",
	})
}

// StopService stops the sing-box service
func (h *Handler) StopService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Stop(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "service stopped successfully",
	})
}

// RestartService restarts the sing-box service
func (h *Handler) RestartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Restart(); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "service restarted successfully",
	})
}
