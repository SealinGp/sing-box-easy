package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/sagernet/sing/common/json"
	"go.uber.org/zap"
)

// ErrVersionNotFound is returned (wrapped) when a requested version id does not
// exist. Callers use errors.Is to distinguish a missing version from a real
// store/parse failure.
var ErrVersionNotFound = errors.New("config version not found")

// VersionInfo is metadata about a stored historical config (without content).
type VersionInfo struct {
	ID        int64     `json:"id"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// VersionStore persists historical config snapshots. It is implemented by the
// configversion package (DB-backed). The config package depends only on this
// interface so it stays free of the database package and remains unit-testable
// with a mock.
type VersionStore interface {
	// Save persists a snapshot and returns its new id.
	Save(content []byte) (int64, error)
	// List returns version metadata, newest first (no content).
	List() ([]VersionInfo, error)
	// Get returns the full content of a single version.
	Get(id int64) ([]byte, error)
	// Delete removes a single version by id, returning ErrVersionNotFound when
	// no such version exists.
	Delete(id int64) error
	// Prune deletes the oldest versions beyond `keep`.
	Prune(keep int) error
}

// SetVersionStore attaches a history store. When nil, version features are
// no-ops (and Rollback/GetBackupConfig report no backup available).
func (m *Manager) SetVersionStore(store VersionStore) {
	m.store = store
}

// SetKeepVersions sets how many historical versions to retain, clamped to a
// sane range. Safe to call concurrently with config saves.
func (m *Manager) SetKeepVersions(n int) {
	if n < minKeepVersions {
		n = minKeepVersions
	}
	if n > maxKeepVersions {
		n = maxKeepVersions
	}
	m.keepVersions.Store(int64(n))
}

// snapshotCurrent stores the current on-disk config into history, then prunes.
// Best-effort: a history failure is logged but never blocks the caller (a user
// must always be able to save/rollback their config even if bookkeeping fails).
func (m *Manager) snapshotCurrent() {
	if m.store == nil {
		return
	}
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("version snapshot skipped: cannot read current config", zap.Error(err))
		}
		return
	}
	if _, err := m.store.Save(data); err != nil {
		logger.Warn("version snapshot failed", zap.Error(err))
		return
	}
	if err := m.store.Prune(int(m.keepVersions.Load())); err != nil {
		logger.Warn("version prune failed", zap.Error(err))
	}
}

// ListVersions returns historical config metadata, newest first.
func (m *Manager) ListVersions() ([]VersionInfo, error) {
	if m.store == nil {
		return []VersionInfo{}, nil
	}
	return m.store.List()
}

// GetVersion returns a stored historical config parsed with the typed registry.
func (m *Manager) GetVersion(id int64) (*SingBoxConfig, error) {
	if m.store == nil {
		return nil, fmt.Errorf("version store not configured")
	}
	content, err := m.store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read version %d: %w", id, err)
	}
	var cfg SingBoxConfig
	jsonCtx := CreateContext(context.Background())
	if err := json.UnmarshalContext(jsonCtx, content, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse version %d: %w", id, err)
	}
	return &cfg, nil
}

// DeleteVersion removes a single historical version by id. It only touches the
// history store and never affects the live config, so it is safe regardless of
// which version is the latest.
func (m *Manager) DeleteVersion(id int64) error {
	if m.store == nil {
		return fmt.Errorf("version store not configured")
	}
	return m.store.Delete(id)
}

// RollbackToVersion restores a specific historical version to the live config.
//
// It snapshots the current config first (so a rollback is itself reversible),
// then writes the chosen version over config.json. Like Rollback, it
// deliberately skips `sing-box check` — rollback is a recovery escape hatch,
// and the version was valid when first saved. A structural parse is still done
// so a corrupt row can never clobber the live config.
func (m *Manager) RollbackToVersion(id int64) error {
	if m.store == nil {
		return fmt.Errorf("version store not configured")
	}
	content, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to read version %d: %w", id, err)
	}

	// Structural sanity: a parse failure means the stored row is unusable.
	var cfg SingBoxConfig
	jsonCtx := CreateContext(context.Background())
	if err := json.UnmarshalContext(jsonCtx, content, &cfg); err != nil {
		return fmt.Errorf("version %d is not a valid config: %w", id, err)
	}

	// Snapshot current so the user can undo the rollback.
	m.snapshotCurrent()

	// Write atomically (stage to the temp file, then rename) so a crash
	// mid-write can never leave a truncated live config — same guarantee as
	// SaveConfig.
	if err := os.WriteFile(m.newConfigPath, content, 0600); err != nil {
		return fmt.Errorf("failed to stage version %d for restore: %w", id, err)
	}
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		os.Remove(m.newConfigPath)
		return fmt.Errorf("failed to restore version %d: %w", id, err)
	}
	return nil
}
