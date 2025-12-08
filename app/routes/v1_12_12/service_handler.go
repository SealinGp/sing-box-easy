package v1_13_0

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetServiceStatus returns the current status of sing-box service
func (h *Handler) GetServiceStatus(ctx context.Context, c *app.RequestContext) {
	running, err := h.serviceController.Status()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	status := "stopped"
	if running {
		status = "running"
	}

	respOK(ctx, c, map[string]any{
		"status":  status,
		"running": running,
	})
}

// StartService starts the sing-box service
func (h *Handler) StartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Start(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service started successfully"})
}

// StopService stops the sing-box service
func (h *Handler) StopService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Stop(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service stopped successfully"})
}

// RestartService restarts the sing-box service
func (h *Handler) RestartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Restart(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service restarted successfully"})
}
