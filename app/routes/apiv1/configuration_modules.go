package apiv1

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/sections"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/install"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"
)

func (h *Handler) configuration() *sections.Service   { return h.configurationModule }
func (h *Handler) configDocument() *config.Service    { return h.configServiceModule }
func (h *Handler) outbounds() *outbounds.Service      { return h.outboundsModule }
func (h *Handler) diagnostics() *diagnostics.Service  { return h.diagnosticsModule }
func (h *Handler) settingsService() *settings.Service { return h.settingsServiceModule }
func (h *Handler) trafficService() *traffic.Service   { return h.trafficServiceModule }
func (h *Handler) installation() *install.Service     { return h.installationModule }
