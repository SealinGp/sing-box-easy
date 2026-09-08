package installer

import (
	"context"
	"encoding/json"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

func (h *Service) GetInitStatus(ctx context.Context) (any, error) {
	state := h.initStateManager.GetState()

	configGenerated := state.ConfigGenerated
	if !configGenerated {
		configGenerated = hasMeaningfulSingBoxConfig(h.configManager)
	}

	return map[string]any{
		"initialized": state.Initialized,
		"steps": map[string]any{
			"sing_box_installed":  state.SingBoxInstalled,
			"config_generated":    configGenerated,
			"dashboard_installed": state.DashboardInstalled,
		},
		"sing_box_version": state.SingBoxVersion,
		"init_time":        state.InitTime,
	}, nil
}
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
func (h *Service) CompleteInit(ctx context.Context) (any, error) {
	// Mark initialization as complete (also sets config_generated)
	if err := h.initStateManager.CompleteInitialization(); err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"message": "initialization completed successfully"}, nil
}
func (h *Service) ResetInit(ctx context.Context) (any, error) {
	if err := h.initStateManager.Reset(); err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"message": "initialization state reset successfully"}, nil
}
