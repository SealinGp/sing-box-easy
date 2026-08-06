package settings

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings/repo"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// ErrSettingNotFound is returned (wrapped) when a key has no row. Callers use
// errors.Is to distinguish a missing key from a real database error.
var ErrSettingNotFound = errors.New("setting not found")

// Setting keys.
const (
	KeyConfigVersionsKeep = "config_versions_keep"

	// KeyGitHubToken holds the GitHub access token issued by the device-flow
	// sign-in. Anonymous GitHub API calls are capped at 60/hour per IP, which
	// a shared NAT egress burns through quickly; a user token raises that to
	// 5000/hour. Written only by the githubauth manager.
	KeyGitHubToken = "github_token"

	// KeyGitHubLogin is the signed-in account name, kept purely so the UI can
	// show who is connected.
	KeyGitHubLogin = "github_login"
)

// EnvGitHubToken is consulted when no token is stored in the database, so
// headless deployments can inject a credential without going through sign-in.
const EnvGitHubToken = "GITHUB_TOKEN"

// SecretKeys are settings whose values must never be returned by the API.
var SecretKeys = map[string]bool{
	KeyGitHubToken: true,
}

// config_versions_keep bounds + default (mirrors config package clamps).
const (
	DefaultConfigVersionsKeep = 10
	MinConfigVersionsKeep     = 1
	MaxConfigVersionsKeep     = 100
)

// ManagerXORM manages application settings using XORM.
type ManagerXORM struct {
	e *xorm.Engine
}

// NewManagerXORM creates a new XORM-backed settings manager.
func NewManagerXORM() *ManagerXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}
	return &ManagerXORM{e: e}
}

// Init ensures the settings table exists and seeds defaults.
func (m *ManagerXORM) Init() error {
	if err := m.e.Sync2(new(repo.Setting)); err != nil {
		logger.Error("Failed to sync settings table", zap.Error(err))
		return err
	}
	// Seed the default only when the key is genuinely absent — a real DB error
	// must surface, not be masked by an attempted re-seed.
	_, err := m.Get(KeyConfigVersionsKeep)
	switch {
	case errors.Is(err, ErrSettingNotFound):
		if err := m.Set(KeyConfigVersionsKeep, strconv.Itoa(DefaultConfigVersionsKeep)); err != nil {
			return fmt.Errorf("failed to seed default settings: %w", err)
		}
	case err != nil:
		return fmt.Errorf("failed to read settings during init: %w", err)
	}
	logger.Info("Settings manager initialized with XORM")
	return nil
}

// Get returns the raw string value for a key, or an error if absent.
func (m *ManagerXORM) Get(key string) (string, error) {
	var s repo.Setting
	has, err := m.e.ID(key).Get(&s)
	if err != nil {
		return "", fmt.Errorf("failed to get setting %q: %w", key, err)
	}
	if !has {
		return "", fmt.Errorf("setting %q: %w", key, ErrSettingNotFound)
	}
	return s.Value, nil
}

// All returns every setting as a key→value map.
func (m *ManagerXORM) All() (map[string]string, error) {
	var rows []repo.Setting
	if err := m.e.Find(&rows); err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

// Set upserts a key-value pair atomically.
//
// The UPDATE-then-INSERT pair runs inside a transaction so two concurrent
// writers cannot both see affected==0 and race on the INSERT (which would fail
// the second writer on the primary key). Cols("value") forces the value column
// to be written even when value is the empty string (xorm skips zero-value
// columns by default).
func (m *ManagerXORM) Set(key, value string) error {
	session := m.e.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return fmt.Errorf("failed to begin tx for setting %q: %w", key, err)
	}

	affected, err := session.ID(key).Cols("value").Update(&repo.Setting{Value: value})
	if err != nil {
		_ = session.Rollback()
		return fmt.Errorf("failed to update setting %q: %w", key, err)
	}
	if affected == 0 {
		if _, err := session.Insert(&repo.Setting{Key: key, Value: value}); err != nil {
			_ = session.Rollback()
			return fmt.Errorf("failed to insert setting %q: %w", key, err)
		}
	}
	return session.Commit()
}

// GetConfigVersionsKeep returns the retention count, clamped, with the default
// applied when unset or unparseable.
func (m *ManagerXORM) GetConfigVersionsKeep() int {
	raw, err := m.Get(KeyConfigVersionsKeep)
	if err != nil {
		if !errors.Is(err, ErrSettingNotFound) {
			logger.Warn("failed to read config_versions_keep, using default", zap.Error(err))
		}
		return DefaultConfigVersionsKeep
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		logger.Warn("config_versions_keep is not an integer, using default",
			zap.String("value", raw), zap.Error(err))
		return DefaultConfigVersionsKeep
	}
	return clampKeep(n)
}

// SetConfigVersionsKeep validates and stores the retention count.
func (m *ManagerXORM) SetConfigVersionsKeep(n int) error {
	if n < MinConfigVersionsKeep || n > MaxConfigVersionsKeep {
		return fmt.Errorf("config_versions_keep must be between %d and %d", MinConfigVersionsKeep, MaxConfigVersionsKeep)
	}
	return m.Set(KeyConfigVersionsKeep, strconv.Itoa(n))
}

// GetGitHubToken returns the stored token, falling back to the GITHUB_TOKEN
// environment variable. An empty result means "call GitHub anonymously".
//
// This is read on every GitHub request (it is a TokenFunc), so a token saved
// from the UI takes effect immediately.
func (m *ManagerXORM) GetGitHubToken() string {
	raw, err := m.Get(KeyGitHubToken)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		logger.Warn("failed to read github_token", zap.Error(err))
	}
	if token := strings.TrimSpace(raw); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv(EnvGitHubToken))
}

// SetGitHubToken validates and stores the token issued by device-flow sign-in.
// An empty value clears it, falling the lookup back to GITHUB_TOKEN (or
// anonymous access).
func (m *ManagerXORM) SetGitHubToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return m.Set(KeyGitHubToken, "")
	}
	if err := validateGitHubToken(token); err != nil {
		return err
	}
	return m.Set(KeyGitHubToken, token)
}

// HasGitHubToken reports whether a token is configured, without leaking it.
func (m *ManagerXORM) HasGitHubToken() bool {
	return m.GetGitHubToken() != ""
}

// GetGitHubLogin returns the signed-in GitHub account name, or "" when the
// connection predates login tracking or was made via GITHUB_TOKEN.
func (m *ManagerXORM) GetGitHubLogin() string {
	raw, err := m.Get(KeyGitHubLogin)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		logger.Warn("failed to read github_login", zap.Error(err))
	}
	return strings.TrimSpace(raw)
}

// SetGitHubLogin stores the signed-in account name. It is display-only, so an
// empty value simply clears it.
func (m *ManagerXORM) SetGitHubLogin(login string) error {
	return m.Set(KeyGitHubLogin, strings.TrimSpace(login))
}

// maxTokenLen bounds a pasted credential; real GitHub tokens are well under it.
const maxTokenLen = 512

// validateGitHubToken rejects values that cannot be a usable credential.
// It deliberately does NOT hard-require a ghp_/github_pat_ prefix: GitHub has
// changed token formats before, and Actions-minted tokens differ again.
func validateGitHubToken(token string) error {
	if len(token) < 8 {
		return fmt.Errorf("github token looks too short to be valid")
	}
	if len(token) > maxTokenLen {
		return fmt.Errorf("github token exceeds %d characters", maxTokenLen)
	}
	for _, r := range token {
		// A header value must not carry whitespace or control characters —
		// catching it here beats a confusing http.Header panic later.
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("github token contains whitespace or control characters")
		}
	}
	return nil
}

// MaskSecret renders a credential for display: only the last 4 characters
// survive, so the UI can show "configured" without echoing the secret.
func MaskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "••••"
	}
	return "••••••••" + s[len(s)-4:]
}

func clampKeep(n int) int {
	if n < MinConfigVersionsKeep {
		return MinConfigVersionsKeep
	}
	if n > MaxConfigVersionsKeep {
		return MaxConfigVersionsKeep
	}
	return n
}
