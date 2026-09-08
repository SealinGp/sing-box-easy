// Package diagnostics owns runtime-assisted DNS and routing explanations.
package diagnostics

import (
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
func (s *Service) DNSConfig() (*config.SingBoxConfig, string, error) { return s.loadDNSProbeConfig() }
