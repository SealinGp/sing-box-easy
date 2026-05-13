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

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

const (
	DefaultConfigPath    = "/etc/sing-box/config.json"
	DefaultBackupPath    = "/etc/sing-box/config.old.json"
	DefaultNewConfigPath = "/etc/sing-box/config_new.json"
)

// Manager handles sing-box configuration management
type Manager struct {
	configPath    string
	backupPath    string
	newConfigPath string
	singBoxPath   string // Path to sing-box binary
	templatePath  string // Path to template config file
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
	return &Manager{
		configPath:    configPath,
		backupPath:    filepath.Join(dir, "config.old.json"),
		newConfigPath: filepath.Join(dir, "config_new.json"),
		singBoxPath:   singBoxPath,
		templatePath:  templatePath,
	}
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

// GetBackupConfig reads and returns the backup configuration
func (m *Manager) GetBackupConfig() (*SingBoxConfig, error) {
	data, err := os.ReadFile(m.backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup config file: %w", err)
	}

	var config SingBoxConfig
	jsonCtx := CreateContext(context.Background())
	if err := json.UnmarshalContext(jsonCtx, data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse backup config file: %w", err)
	}

	return &config, nil
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

	// Backup current config if it exists
	if _, err := os.Stat(m.configPath); err == nil {
		if err := m.copyFile(m.configPath, m.backupPath); err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
	}

	// Move validated config to main config
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Rollback restores the backup configuration.
//
// Deliberately skips `sing-box check` on the backup. Rationale:
//
//  1. The backup was already validated when it was first written — SaveConfig
//     never promotes a config to config.old.json unless it passed validation
//     (or baseline-was-broken tolerance, see SaveConfig comments).
//  2. Rollback is a recovery escape hatch. The user only reaches for it when
//     the live config is broken; forcing the backup to revalidate against the
//     *current* sing-box binary can strand them — e.g. when the binary was
//     upgraded and deprecated a field, the backup that worked yesterday will
//     "fail validation" today, and the user has no way out.
//
// We still parse the backup with the typed registry so a totally corrupt /
// truncated file fails fast rather than producing a broken live config; this
// is structural sanity, not semantic validation.
func (m *Manager) Rollback() error {
	if _, err := os.Stat(m.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup config found")
	}

	// Structural sanity check: parse the backup. A parse failure means the
	// backup file is unusable — refuse to overwrite the live config with it.
	if _, err := m.GetBackupConfig(); err != nil {
		return fmt.Errorf("failed to read backup config: %w", err)
	}

	// Restore by copying the backup over the live config. We do NOT touch the
	// backup file itself, so rollback is idempotent: re-running it produces
	// the same result.
	if err := m.copyFile(m.backupPath, m.configPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
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

// copyFile copies a file from src to dst.
// Uses 0600 — proxy configs may contain credentials.
func (m *Manager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
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
