package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// Migration represents a database migration
type Migration struct {
	ID          string    `xorm:"pk varchar(255)" json:"id"`
	Name        string    `xorm:"notnull" json:"name"`
	Executed    bool      `xorm:"notnull default(0)" json:"executed"`
	ExecutedAt  time.Time `xorm:"'executed_at' null" json:"executed_at,omitempty"`
	CreatedAt   time.Time `xorm:"created" json:"created_at"`
}

// TableName specifies the table name for Migration
func (Migration) TableName() string {
	return "schema_migrations"
}

// MigrationFunc represents a migration function
type MigrationFunc func(*xorm.Session) error

// migrations contains all available migrations
var migrations = map[string]MigrationFunc{
	"001_add_enabled_to_subscriptions": migrate001AddEnabledToSubscriptions,
}

// runMigrations executes pending migrations
func runMigrations() error {
	if engine == nil {
		return fmt.Errorf("database engine not initialized")
	}

	// Create migrations table if not exists
	if err := engine.Sync2(new(Migration)); err != nil {
		return fmt.Errorf("failed to sync migrations table: %w", err)
	}

	// Get executed migrations
	var executedMigrations []Migration
	if err := engine.Where("executed = 1").Find(&executedMigrations); err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	executedMap := make(map[string]bool)
	for _, m := range executedMigrations {
		executedMap[m.ID] = true
	}

	// Run pending migrations in order
	for id, migrationFunc := range migrations {
		if !executedMap[id] {
			logger.Info("Running migration", zap.String("id", id))

			migration := &Migration{
				ID:        id,
				Name:      getMigrationName(id),
				Executed:  true,
				ExecutedAt: time.Now(),
			}

			// Start transaction
			session := engine.NewSession()
			defer session.Close()

			if err := session.Begin(); err != nil {
				return fmt.Errorf("failed to begin transaction for migration %s: %w", id, err)
			}

			// Run migration
			if err := migrationFunc(session); err != nil {
				session.Rollback()
				return fmt.Errorf("migration %s failed: %w", id, err)
			}

			// Record migration
			if _, err := session.Insert(migration); err != nil {
				session.Rollback()
				return fmt.Errorf("failed to record migration %s: %w", id, err)
			}

			// Commit transaction
			if err := session.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %s: %w", id, err)
			}

			logger.Info("Migration completed successfully", zap.String("id", id))
		}
	}

	return nil
}

// getMigrationName returns a human-readable name for a migration
func getMigrationName(id string) string {
	switch id {
	case "001_add_enabled_to_subscriptions":
		return "Add enabled field to subscriptions table"
	default:
		return id
	}
}

// migrate001AddEnabledToSubscriptions adds the enabled field to the subscriptions table
func migrate001AddEnabledToSubscriptions(session *xorm.Session) error {
	// Try to add the column directly - if it exists, this will fail and we can ignore it
	sql := `ALTER TABLE subscriptions ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT 1`
	_, err := session.Exec(sql)

	if err != nil {
		// Check if the error is because the column already exists (SQLite error: duplicate column name)
		if isDuplicateColumnError(err) {
			logger.Info("Enabled column already exists in subscriptions table")
			return nil
		}
		return fmt.Errorf("failed to add enabled column: %w", err)
	}

	logger.Info("Added enabled column to subscriptions table")
	return nil
}

// isDuplicateColumnError checks if the error indicates a duplicate column name
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// SQLite error patterns for duplicate column name
	return strings.Contains(strings.ToLower(errStr), "duplicate column name") ||
		   strings.Contains(strings.ToLower(errStr), "column exists") ||
		   strings.Contains(strings.ToLower(errStr), "no such table")
}