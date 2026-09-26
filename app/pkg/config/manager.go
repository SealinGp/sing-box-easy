package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

const DefaultConfigPath = "/etc/sing-box/config.json"

// stagingFileName is the temp file used during the atomic validate→rename save.
// It lives in the same directory as config.json (so os.Rename stays on one
// filesystem) but is deliberately NOT named "*.json": a sing-box instance run
// in config-directory mode (`sing-box -C <dir>`) merges every "*.json" file in
// the directory, so a ".json" staging file — especially one left behind by a
// validate-without-save — would collide with config.json and abort sing-box
// with a "duplicate tag" error. A non-".json" suffix is ignored by that scan.
const stagingFileName = ".config_new.tmp"

// legacyStagingFileName is the old staging filename (pre-".config_new.tmp").
// Removed on startup so upgraded hosts don't keep a stale "*.json" sibling that
// directory-mode sing-box would still merge.
const legacyStagingFileName = "config_new.json"

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
// transient .config_new.tmp staging file used during the atomic
// validate→rename save — see stagingFileName for why it isn't a *.json file).
type Manager struct {
	configPath    string
	newConfigPath string
	singBoxPath   string // Path to sing-box binary
	templatePath  string // Path to template config file
	core          CoreAdapter

	store VersionStore // historical config snapshots (nil => disabled), set once at startup
	// mutationMu protects the single staging path and serializes atomic config
	// replacement. Validation also uses that path, so it participates in the
	// same lock even though it does not modify the active file.
	mutationMu sync.Mutex

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
		newConfigPath: filepath.Join(dir, stagingFileName),
		singBoxPath:   singBoxPath,
		templatePath:  templatePath,
		core:          NewBinaryCoreAdapter(singBoxPath),
	}
	// Best-effort cleanup of a stale legacy staging file from older versions.
	// It is never the authoritative config, and leaving a "config_new.json"
	// behind breaks sing-box when it runs in directory mode.
	_ = os.Remove(filepath.Join(dir, legacyStagingFileName))
	m.keepVersions.Store(defaultKeepVersions)
	return m
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
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

// UpdateOutbounds adds multiple outbounds, skipping duplicates based on existing tags.
//
// If every input outbound is already present (or the input slice is empty),
// the function returns without touching disk — there is no point running the
// write-then-validate cycle on a config that has not changed, and doing so
// would surface unrelated pre-existing validation failures.
// firstExistingTag returns the first candidate tag already present in the
// config, if any.
func firstExistingTag(existingTags map[string]bool, candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if existingTags[candidate] {
			return candidate, true
		}
	}
	return "", false
}

func (m *Manager) UpdateOutbounds(outbounds []Outbound) (addedTags []string, skippedTags []string, err error) {
	if len(outbounds) == 0 {
		return nil, nil, nil
	}

	// Pre-flight: compute the diff against the current on-disk config so we
	// can decide whether a save is necessary before we touch anything.
	current, err := m.GetOutboundsConfig()
	if err != nil {
		return nil, nil, err
	}
	existingTags := make(map[string]bool, len(current.Outbounds))
	for _, existing := range current.Outbounds {
		existingTags[existing.Tag] = true
	}

	toAdd := make([]Outbound, 0, len(outbounds))
	for _, outbound := range outbounds {
		// The first candidate is the tag to mint; the rest are older shapes the
		// same node may already be stored under, so re-adding stays a skip rather
		// than producing a second copy under a new tag.
		candidates := OutboundTagCandidates(outbound.Tag, outbound)
		outbound.Tag = candidates[0]

		if existing, found := firstExistingTag(existingTags, candidates); found {
			skippedTags = append(skippedTags, existing)
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

	err = m.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
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
