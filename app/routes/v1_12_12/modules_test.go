package v1_13_0

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/configuration"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds"
)

func testHandler(h *Handler) *Handler {
	h.configurationModule = configuration.New(h.configManager)
	h.outboundsModule = outbounds.New(h.configManager, nil)
	h.diagnosticsModule = diagnostics.New(h.configManager, h.serviceController)
	return h
}
