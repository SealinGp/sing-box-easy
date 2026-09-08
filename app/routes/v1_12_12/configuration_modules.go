package v1_13_0

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/configuration"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"
)

func (h *Handler) configuration() *configuration.Service { return h.configurationModule }
func (h *Handler) outbounds() *outbounds.Service         { return h.outboundsModule }
func (h *Handler) diagnostics() *diagnostics.Service     { return h.diagnosticsModule }
func (h *Handler) settingsService() *settings.Service    { return h.settingsServiceModule }
func (h *Handler) trafficService() *trafficflow.Service  { return h.trafficServiceModule }
func (h *Handler) installation() *installer.Service      { return h.installationModule }
