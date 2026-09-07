package v1_13_0

import (
	"context"
	"encoding/json"

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
	document, err := mgr.GetConfigDocument()
	if err != nil {
		return false
	}
	var cfg struct {
		Outbounds []json.RawMessage `json:"outbounds"`
		Inbounds  []json.RawMessage `json:"inbounds"`
		Route     struct {
			Rules []json.RawMessage `json:"rules"`
		} `json:"route"`
		Experimental struct {
			ClashAPI struct {
				ExternalController string `json:"external_controller"`
			} `json:"clash_api"`
		} `json:"experimental"`
	}
	if err := json.Unmarshal(document.Raw, &cfg); err != nil {
		return false
	}
	if len(cfg.Outbounds) > 2 {
		return true
	}
	if len(cfg.Inbounds) > 0 {
		return true
	}
	if len(cfg.Route.Rules) > 0 {
		return true
	}
	// A configured Clash API external_controller is a reliable signal that
	// the operator has already set up the app (possibly through the wizard).
	// Treat it as "meaningful" so the wizard does not re-run on upgrades.
	if cfg.Experimental.ClashAPI.ExternalController != "" {
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
