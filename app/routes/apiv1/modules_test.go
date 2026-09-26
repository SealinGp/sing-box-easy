package apiv1

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/sections"
)

func testHandler(h *Handler) *Handler {
	h.configurationModule = sections.New(h.configManager)
	h.outboundsModule = outbounds.New(h.configManager, nil)
	h.diagnosticsModule = diagnostics.New(h.configManager, h.serviceController)
	return h
}
