package configversion

import (
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/configversion/repo"
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// StoreXORM is the database-backed implementation of config.VersionStore.
type StoreXORM struct {
	e *xorm.Engine
}

// NewStoreXORM creates a new XORM-backed config version store.
func NewStoreXORM() *StoreXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}
	return &StoreXORM{e: e}
}

// Init ensures the config_versions table exists.
func (s *StoreXORM) Init() error {
	if err := s.e.Sync2(new(repo.ConfigVersion)); err != nil {
		logger.Error("Failed to sync config_versions table", zap.Error(err))
		return err
	}
	logger.Info("Config version store initialized with XORM")
	return nil
}

// Save persists a snapshot and returns its new id.
func (s *StoreXORM) Save(content []byte) (int64, error) {
	row := &repo.ConfigVersion{
		Content: string(content),
		Size:    len(content),
	}
	if _, err := s.e.Insert(row); err != nil {
		return 0, fmt.Errorf("failed to insert config version: %w", err)
	}
	return row.ID, nil
}

// List returns version metadata (id/size/created_at only), newest first.
// Content is intentionally excluded so listing never loads every blob.
func (s *StoreXORM) List() ([]config.VersionInfo, error) {
	var rows []repo.ConfigVersion
	if err := s.e.Cols("id", "size", "created_at").Desc("id").Find(&rows); err != nil {
		return nil, fmt.Errorf("failed to list config versions: %w", err)
	}
	out := make([]config.VersionInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, config.VersionInfo{
			ID:        r.ID,
			Size:      r.Size,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// Get returns the full content of a single version.
func (s *StoreXORM) Get(id int64) ([]byte, error) {
	var row repo.ConfigVersion
	has, err := s.e.ID(id).Get(&row)
	if err != nil {
		return nil, fmt.Errorf("failed to get config version %d: %w", id, err)
	}
	if !has {
		return nil, fmt.Errorf("config version %d: %w", id, config.ErrVersionNotFound)
	}
	return []byte(row.Content), nil
}

// Prune deletes the oldest versions beyond `keep`. When keep <= 0 it is a no-op
// (the manager clamps keep to a sane minimum before calling).
func (s *StoreXORM) Prune(keep int) error {
	if keep <= 0 {
		return nil
	}

	// Find the id of the newest row that must be kept; delete everything older.
	var keepRows []repo.ConfigVersion
	if err := s.e.Cols("id").Desc("id").Limit(1, keep-1).Find(&keepRows); err != nil {
		return fmt.Errorf("failed to find prune boundary: %w", err)
	}
	if len(keepRows) == 0 {
		return nil // fewer than `keep` rows exist
	}

	boundaryID := keepRows[0].ID
	if _, err := s.e.Where("id < ?", boundaryID).Delete(new(repo.ConfigVersion)); err != nil {
		return fmt.Errorf("failed to prune config versions: %w", err)
	}
	return nil
}
