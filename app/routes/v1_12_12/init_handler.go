package v1_13_0

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetInitStatus returns initialization status
func (h *Handler) GetInitStatus(ctx context.Context, c *app.RequestContext) {
	state := h.initStateManager.GetState()

	respOK(ctx, c, map[string]any{
		"initialized": state.Initialized,
		"steps": map[string]any{
			"sing_box_installed":  state.SingBoxInstalled,
			"config_generated":    state.ConfigGenerated,
			"dashboard_installed": state.DashboardInstalled,
		},
		"sing_box_version": state.SingBoxVersion,
		"init_time":        state.InitTime,
	})
}

// CompleteInit marks initialization as complete
func (h *Handler) CompleteInit(ctx context.Context, c *app.RequestContext) {
	// Mark initialization as complete (also sets config_generated)
	if err := h.initStateManager.CompleteInitialization(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "initialization completed successfully"})
}

// ResetInit resets initialization state
func (h *Handler) ResetInit(ctx context.Context, c *app.RequestContext) {
	if err := h.initStateManager.Reset(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "initialization state reset successfully"})
}
