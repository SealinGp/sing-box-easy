package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetSettings returns all application settings as a key→value map, plus the
// typed config-version retention for convenience.
func (h *Handler) GetSettings(ctx context.Context, c *app.RequestContext) {
	all, err := h.settingsManager.All()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	// Secrets never leave the server. The GitHub token is managed entirely by
	// the sign-in endpoints under /github/auth; it is never readable here.
	safe := make(map[string]string, len(all))
	for k, v := range all {
		if settings.SecretKeys[k] {
			continue
		}
		safe[k] = v
	}

	respOK(ctx, c, map[string]any{
		"settings":             safe,
		"config_versions_keep": h.settingsManager.GetConfigVersionsKeep(),
	})
}

// UpdateSettings updates application settings. Currently supports
// config_versions_keep; the new retention is applied to the config manager
// immediately so subsequent saves prune to the new count.
func (h *Handler) UpdateSettings(ctx context.Context, c *app.RequestContext) {
	// The GitHub credential is deliberately NOT writable here — it is only
	// ever issued by the device-flow sign-in under /github/auth, so there is
	// no path that accepts a raw token from a client.
	type Request struct {
		ConfigVersionsKeep *int `json:"config_versions_keep"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.ConfigVersionsKeep != nil {
		if err := h.settingsManager.SetConfigVersionsKeep(*req.ConfigVersionsKeep); err != nil {
			respErr(ctx, c, CodeValidationError, err.Error())
			return
		}
		h.configManager.SetKeepVersions(*req.ConfigVersionsKeep)
	}

	respOK(ctx, c, map[string]any{
		"config_versions_keep": h.settingsManager.GetConfigVersionsKeep(),
		"limits": map[string]int{
			"min": settings.MinConfigVersionsKeep,
			"max": settings.MaxConfigVersionsKeep,
		},
	})
}
