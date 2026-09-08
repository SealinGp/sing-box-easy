package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

type RetentionPolicy interface{ SetKeepVersions(int) }
type Service struct {
	settingsManager *ManagerXORM
	configManager   RetentionPolicy
	configured      func() bool
}

func NewService(store *ManagerXORM, retention RetentionPolicy, configured func() bool) *Service {
	return &Service{store, retention, configured}
}
func (h *Service) GetSettings(ctx context.Context) (any, error) {
	all, err := h.settingsManager.All()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	// Secrets never leave the server. The GitHub token is managed entirely by
	// the sign-in endpoints under /github/auth; it is never readable here.
	safe := make(map[string]string, len(all))
	for k, v := range all {
		if SecretKeys[k] {
			continue
		}
		safe[k] = v
	}

	return map[string]any{
		"settings":             safe,
		"config_versions_keep": h.settingsManager.GetConfigVersionsKeep(),
		// Surfaced explicitly rather than left for the caller to dig out of
		// `settings`, because "" is meaningful here: it means the deployment
		// is falling back to app.yml / GITHUB_OAUTH_CLIENT_ID, which the UI
		// needs to distinguish from "not configured at all".
		"github_oauth_client_id":  h.settingsManager.GetGitHubOAuthClientID(),
		"github_oauth_configured": h.configured(),
	}, nil
}

type UpdateSettingsCommand struct {
	ConfigVersionsKeep *int `json:"config_versions_keep"`
	// The OAuth *client ID* is writable here — unlike the token. The device
	// flow has no client secret, so this value is public by construction
	// and carries no privilege on its own. Empty clears the override.
	GitHubOAuthClientID *string `json:"github_oauth_client_id"`
}

func (h *Service) UpdateSettings(ctx context.Context, req UpdateSettingsCommand) (any, error) {
	// The GitHub credential is deliberately NOT writable here — it is only
	// ever issued by the device-flow sign-in under /github/auth, so there is
	// no path that accepts a raw token from a client.

	values := make(map[string]string)
	if req.ConfigVersionsKeep != nil {
		n := *req.ConfigVersionsKeep
		if n < MinConfigVersionsKeep || n > MaxConfigVersionsKeep {
			return nil, fault.New(fault.Invalid, fmt.Sprintf("config_versions_keep must be between %d and %d", MinConfigVersionsKeep, MaxConfigVersionsKeep))
		}
		values[KeyConfigVersionsKeep] = strconv.Itoa(n)
	}
	if req.GitHubOAuthClientID != nil {
		id := strings.TrimSpace(*req.GitHubOAuthClientID)
		if id != "" {
			if err := validateOAuthClientID(id); err != nil {
				return nil, fault.New(fault.Invalid, err.Error())
			}
		}
		values[KeyGitHubOAuthClientID] = id
	}
	if err := h.settingsManager.SetMany(values); err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	if req.ConfigVersionsKeep != nil {
		h.configManager.SetKeepVersions(*req.ConfigVersionsKeep)
	}

	return map[string]any{
		"config_versions_keep":    h.settingsManager.GetConfigVersionsKeep(),
		"github_oauth_client_id":  h.settingsManager.GetGitHubOAuthClientID(),
		"github_oauth_configured": h.configured(),
		"limits": map[string]int{
			"min": MinConfigVersionsKeep,
			"max": MaxConfigVersionsKeep,
		},
	}, nil
}
