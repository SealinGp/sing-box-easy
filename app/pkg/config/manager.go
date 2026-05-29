package config

import (
	"bytes"
	"context"
	js "encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

const (
	DefaultConfigPath    = "/etc/sing-box/config.json"
	DefaultNewConfigPath = "/etc/sing-box/config_new.json"
)

// Version retention bounds. The active count is configurable via settings;
// these clamp it to a sane range. Kept here (not imported from the settings
// package) so config stays free of the database dependency — they intentionally
// mirror settings.{Default,Min,Max}ConfigVersionsKeep and must stay in sync.
const (
	defaultKeepVersions = 10
	minKeepVersions     = 1
	maxKeepVersions     = 100
)

// Manager handles sing-box configuration management.
//
// Historical configs are kept in a database via the VersionStore, so the
// sing-box config directory only ever holds the live config.json (plus the
// transient config_new.json used during the atomic validate→rename save).
type Manager struct {
	configPath    string
	newConfigPath string
	singBoxPath   string // Path to sing-box binary
	templatePath  string // Path to template config file

	store VersionStore // historical config snapshots (nil => disabled), set once at startup

	// keepVersions (how many historical versions to retain) is read by
	// snapshotCurrent on config-save goroutines and written by SetKeepVersions
	// on the settings-update goroutine, so it is accessed atomically.
	keepVersions atomic.Int64
}

// NewManager creates a new configuration manager
func NewManager(configPath, singBoxPath, templatePath string) *Manager {
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	if singBoxPath == "" {
		singBoxPath = "sing-box" // Use PATH
	}
	if templatePath == "" {
		templatePath = "doc/config.1.12.12.json" // Default template
	}

	dir := filepath.Dir(configPath)
	m := &Manager{
		configPath:    configPath,
		newConfigPath: filepath.Join(dir, "config_new.json"),
		singBoxPath:   singBoxPath,
		templatePath:  templatePath,
	}
	m.keepVersions.Store(defaultKeepVersions)
	return m
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetConfig reads and returns the current configuration
func (m *Manager) GetConfig() (*SingBoxConfig, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SingBoxConfig
	jsonCtx := CreateContext(context.Background())
	if err := json.UnmarshalContext(jsonCtx, data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetBackupConfig returns the most recent historical config (the "backup").
// Backed by the version store; kept for API back-compat with /config/backup.
func (m *Manager) GetBackupConfig() (*SingBoxConfig, error) {
	if m.store == nil {
		return nil, fmt.Errorf("no backup available")
	}
	versions, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no backup config found")
	}
	return m.GetVersion(versions[0].ID)
}

func (m *Manager) createNewConfig(config *SingBoxConfig) error {
	// Save to temporary file with pretty printing
	jsonCtx := CreateContext(context.Background())
	data, err := json.MarshalContext(jsonCtx, config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Pretty print the JSON
	var prettyBuf bytes.Buffer
	if err := js.Indent(&prettyBuf, data, "", "  "); err != nil {
		return fmt.Errorf("failed to indent config: %w", err)
	}

	if err := os.WriteFile(m.newConfigPath, prettyBuf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}
	return nil
}

// runSingBoxCheck shells out to `sing-box check -c <path>` and returns a
// validation error if sing-box reports one. This is a pure read of the given
// file — it does not modify or remove it.
func (m *Manager) runSingBoxCheck(path string) error {
	cmdParams := []string{"check", "-c", path}
	cmd := exec.Command(m.singBoxPath, cmdParams...)
	cmd.Env = append(os.Environ(), "ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	logger.Infof("%s %s output:%s", m.singBoxPath, strings.Join(cmdParams, " "), output)
	if bytes.Contains(output, []byte("ERROR")) || bytes.Contains(output, []byte("FATAL")) {
		return fmt.Errorf("config validation failed: %s", string(output))
	}
	return nil
}

// ValidateConfig validates the configuration using sing-box binary.
// On success, the validated config remains on disk at newConfigPath so callers
// can reuse it (e.g. SaveConfig's rename). On failure, the temp file is removed
// so it doesn't linger and confuse the next save.
func (m *Manager) ValidateConfig(config *SingBoxConfig) error {
	if err := m.createNewConfig(config); err != nil {
		return err
	}
	if err := m.runSingBoxCheck(m.newConfigPath); err != nil {
		os.Remove(m.newConfigPath)
		return err
	}
	return nil
}

// SaveConfig saves the configuration after validation.
//
// Tolerance for pre-existing baseline errors:
// Configs in this app are often authored for a Linux target but edited from a
// macOS dev machine. Features like TUN's auto_redirect / auto_route /
// strict_route only initialize on Linux, so `sing-box check` rejects them on
// Darwin. If we strictly required a clean check on every save, the user could
// never edit their config from the dev machine — every change, including
// subscription refreshes, would be blocked by an inbound the user didn't even
// touch.
//
// Rule: if the on-disk baseline ALREADY fails validation, the user is editing
// a config that was broken before this call. Our change isn't to blame — log a
// warning and let the save proceed. If the baseline was clean and our change
// introduces a failure, keep blocking (the common production case).
func (m *Manager) SaveConfig(config *SingBoxConfig) error {
	// Capture the baseline state once, before we write the proposed config.
	// A baseline error is informational only — it controls how we react to a
	// post-change error below.
	baselineErr := m.runSingBoxCheck(m.configPath)

	if err := m.ValidateConfig(config); err != nil {
		if baselineErr != nil {
			// Baseline was already broken. Recreate the temp file (ValidateConfig
			// removed it on error) and proceed with the save.
			logger.Warn("config save tolerated despite validation failure: baseline config also fails validation, so this change is not the cause",
				zap.String("baseline_error", baselineErr.Error()),
				zap.String("post_change_error", err.Error()))
			if cerr := m.createNewConfig(config); cerr != nil {
				return cerr
			}
		} else {
			return err
		}
	}

	// Snapshot the current (about-to-be-replaced) config into history before
	// promoting the new one. Best-effort: never block a save on bookkeeping.
	m.snapshotCurrent()

	// Move validated config to main config
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Rollback restores the most recent historical config (the newest version).
// It is a thin wrapper over RollbackToVersion; see that method for the rationale
// behind skipping `sing-box check` on restore.
func (m *Manager) Rollback() error {
	if m.store == nil {
		return fmt.Errorf("no backup available")
	}
	versions, err := m.store.List()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}
	if len(versions) == 0 {
		return fmt.Errorf("no backup config found")
	}
	return m.RollbackToVersion(versions[0].ID)
}

// UpdateConfig updates the configuration with a function
// This is useful for partial updates
func (m *Manager) UpdateConfig(updateFn func(*SingBoxConfig) error) error {
	config, err := m.GetConfig()
	if err != nil {
		return err
	}

	if err := updateFn(config); err != nil {
		logger.Error("failed to update config", zap.Error(err))
		return err
	}

	return m.SaveConfig(config)
}

// UpdateOutbounds adds multiple outbounds, skipping duplicates based on existing tags.
//
// If every input outbound is already present (or the input slice is empty),
// the function returns without touching disk — there is no point running the
// write-then-validate cycle on a config that has not changed, and doing so
// would surface unrelated pre-existing validation failures.
func (m *Manager) UpdateOutbounds(outbounds []Outbound) (addedTags []string, skippedTags []string, err error) {
	if len(outbounds) == 0 {
		return nil, nil, nil
	}

	// Pre-flight: compute the diff against the current on-disk config so we
	// can decide whether a save is necessary before we touch anything.
	current, err := m.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	existingTags := make(map[string]bool, len(current.Outbounds))
	for _, existing := range current.Outbounds {
		existingTags[existing.Tag] = true
	}

	toAdd := make([]Outbound, 0, len(outbounds))
	for _, outbound := range outbounds {
		originalTag := outbound.Tag
		outbound.Tag = GenerateUniqueTag(originalTag, outbound)

		if existingTags[outbound.Tag] {
			skippedTags = append(skippedTags, outbound.Tag)
			continue
		}
		toAdd = append(toAdd, outbound)
		addedTags = append(addedTags, outbound.Tag)
	}

	if len(toAdd) == 0 {
		logger.Warn("all outbounds already exist; skipping save",
			zap.Int("input", len(outbounds)),
			zap.Int("skipped", len(skippedTags)))
		return addedTags, skippedTags, nil
	}

	err = m.UpdateConfig(func(cfg *SingBoxConfig) error {
		cfg.Outbounds = append(cfg.Outbounds, toAdd...)
		return nil
	})
	return
}

// InitializeConfig initializes the configuration from template
// This should be called after sing-box installation
func (m *Manager) InitializeConfig() error {
	// Check if config already exists
	if _, err := os.Stat(m.configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", m.configPath)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read template config
	templateData, err := os.ReadFile(m.templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template config: %w", err)
	}

	// Validate template is valid JSON
	var config SingBoxConfig
	if err := json.Unmarshal(templateData, &config); err != nil {
		return fmt.Errorf("invalid template config: %w", err)
	}

	// Write to config path with restrictive perms (proxy creds inside)
	if err := os.WriteFile(m.configPath, templateData, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
