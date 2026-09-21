// Package diagnostics owns runtime-assisted DNS and routing explanations.
package diagnostics

import (
	"encoding/json"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox"
)

type Service struct {
	configManager     *config.Manager
	serviceController *service.Controller
}

func New(cm *config.Manager, controller *service.Controller) *Service {
	return &Service{cm, controller}
}
func (s *Service) readClashAPISettings() (config.ClashAPISettings, error) {
	return s.configManager.GetClashAPISettings()
}
// DNSConfig exposes the probe's two projections of the dns section: the typed
// servers (for live queries) and the raw section (for the rule walk).
func (s *Service) DNSConfig() (*config.SingBoxConfig, json.RawMessage, error) {
	return s.loadDNSProbeConfig()
}
