package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetInitStatus returns initialization status.
//
// In addition to the stored init state, this performs live detection of two
// signals so that operators who deployed sing-box-easy *over* an existing
// sing-box setup don't get stuck on the wizard:
//
//   - config_generated: true when /etc/sing-box/config.json already has
//     meaningful (non-default) content — see hasMeaningfulSingBoxConfig.
//
// The result is OR-merged with the stored state, so a manual "Reset Init"
// won't override the live signal — that matches the pattern used for
// sing_box_installed (installer.Init at startup writes the stored flag).
func (h *Handler) GetInitStatus(ctx context.Context, c *app.RequestContext) {
	state := h.initStateManager.GetState()

	configGenerated := state.ConfigGenerated
	if !configGenerated {
		configGenerated = hasMeaningfulSingBoxConfig(h.configManager)
	}

	respOK(ctx, c, map[string]any{
		"initialized": state.Initialized,
		"steps": map[string]any{
			"sing_box_installed":  state.SingBoxInstalled,
			"config_generated":    configGenerated,
			"dashboard_installed": state.DashboardInstalled,
		},
		"sing_box_version": state.SingBoxVersion,
		"init_time":        state.InitTime,
	})
}

// hasMeaningfulSingBoxConfig returns true when config.json contains real
// user-authored content rather than just sing-box's bootstrap defaults.
// We avoid claiming "configured" on a totally empty/missing file; conversely
// any of: more than two outbounds (anything beyond direct+block), at least
// one inbound, or at least one route rule counts as user-supplied content.
//
// Read failures (missing file, parse errors) intentionally return false —
// they mean we cannot prove the user has a real config, so let the wizard
// guide them.
func hasMeaningfulSingBoxConfig(mgr *config.Manager) bool {
	if mgr == nil {
		return false
	}
	cfg, err := mgr.GetConfig()
	if err != nil || cfg == nil {
		return false
	}
	if len(cfg.Outbounds) > 2 {
		return true
	}
	if len(cfg.Inbounds) > 0 {
		return true
	}
	if cfg.Route.Rules != nil && len(cfg.Route.Rules) > 0 {
		return true
	}
	return false
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
