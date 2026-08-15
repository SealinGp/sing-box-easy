package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
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
		// Surfaced explicitly rather than left for the caller to dig out of
		// `settings`, because "" is meaningful here: it means the deployment
		// is falling back to app.yml / GITHUB_OAUTH_CLIENT_ID, which the UI
		// needs to distinguish from "not configured at all".
		"github_oauth_client_id":  h.settingsManager.GetGitHubOAuthClientID(),
		"github_oauth_configured": h.githubAuth.Configured(),
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
		// The OAuth *client ID* is writable here — unlike the token. The device
		// flow has no client secret, so this value is public by construction
		// and carries no privilege on its own. Empty clears the override.
		GitHubOAuthClientID *string `json:"github_oauth_client_id"`
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

	if req.GitHubOAuthClientID != nil {
		if err := h.settingsManager.SetGitHubOAuthClientID(*req.GitHubOAuthClientID); err != nil {
			respErr(ctx, c, CodeValidationError, err.Error())
			return
		}
	}

	respOK(ctx, c, map[string]any{
		"config_versions_keep":    h.settingsManager.GetConfigVersionsKeep(),
		"github_oauth_client_id":  h.settingsManager.GetGitHubOAuthClientID(),
		"github_oauth_configured": h.githubAuth.Configured(),
		"limits": map[string]int{
			"min": settings.MinConfigVersionsKeep,
			"max": settings.MaxConfigVersionsKeep,
		},
	})
}

// GetSubscriptionInfoKeywords returns the labels that mark a feed entry as
// account metadata ("剩余流量：4.7 TB") instead of a proxy node: the effective
// list actually used for matching, the built-in defaults (so the UI can offer a
// reset), and whether the deployment is still on those defaults.
func (h *Handler) GetSubscriptionInfoKeywords(ctx context.Context, c *app.RequestContext) {
	configured := subscription.NormalizeInfoKeywords(h.settingsManager.GetSubscriptionInfoKeywords())

	respOK(ctx, c, map[string]any{
		"keywords":       subscription.EffectiveInfoKeywords(configured),
		"defaults":       subscription.DefaultInfoLabelKeywords,
		"using_defaults": len(configured) == 0,
		"limits": map[string]int{
			"max_keywords": settings.MaxInfoKeywords,
			"max_length":   settings.MaxInfoKeywordLen,
		},
	})
}

// UpdateSubscriptionInfoKeywords replaces the override list. Sending an empty
// array clears it, restoring the built-in defaults. The list is normalized
// (trimmed, lowercased, de-duplicated) before it is stored, so matching stays
// case-insensitive and the UI always reads back what it will actually match on.
func (h *Handler) UpdateSubscriptionInfoKeywords(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Keywords []string `json:"keywords"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	normalized := subscription.NormalizeInfoKeywords(req.Keywords)
	if err := h.settingsManager.SetSubscriptionInfoKeywords(normalized); err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"keywords":       subscription.EffectiveInfoKeywords(normalized),
		"defaults":       subscription.DefaultInfoLabelKeywords,
		"using_defaults": len(normalized) == 0,
	})
}
