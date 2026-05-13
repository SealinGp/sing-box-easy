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

// migrationEntry pairs an ID with its function. We use a slice rather than
// a map so migrations always run in declared order — Go map iteration is
// randomised, which would silently break ordering-dependent migrations.
type migrationEntry struct {
	ID string
	Fn MigrationFunc
}

// migrations contains all available migrations, in execution order.
// Append new migrations to the END; never reorder existing entries.
var migrations = []migrationEntry{
	{"001_add_enabled_to_subscriptions", migrate001AddEnabledToSubscriptions},
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

	// Run pending migrations in declared order. Each migration is wrapped in
	// its own function so `defer session.Close()` fires per iteration rather
	// than accumulating to the end of runMigrations.
	for _, entry := range migrations {
		if executedMap[entry.ID] {
			continue
		}
		if err := runOneMigration(entry); err != nil {
			return err
		}
	}

	return nil
}

// runOneMigration executes a single migration in its own transaction.
// Isolating the body lets defer session.Close() fire when this function
// returns, freeing the database connection between migrations.
func runOneMigration(entry migrationEntry) error {
	logger.Info("Running migration", zap.String("id", entry.ID))

	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction for migration %s: %w", entry.ID, err)
	}

	if err := entry.Fn(session); err != nil {
		if rbErr := session.Rollback(); rbErr != nil {
			logger.Error("rollback failed after migration error",
				zap.String("id", entry.ID), zap.Error(rbErr))
		}
		return fmt.Errorf("migration %s failed: %w", entry.ID, err)
	}

	record := &Migration{
		ID:         entry.ID,
		Name:       getMigrationName(entry.ID),
		Executed:   true,
		ExecutedAt: time.Now(),
	}
	if _, err := session.Insert(record); err != nil {
		if rbErr := session.Rollback(); rbErr != nil {
			logger.Error("rollback failed after insert error",
				zap.String("id", entry.ID), zap.Error(rbErr))
		}
		return fmt.Errorf("failed to record migration %s: %w", entry.ID, err)
	}

	if err := session.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", entry.ID, err)
	}

	logger.Info("Migration completed successfully", zap.String("id", entry.ID))
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

// migrate001AddEnabledToSubscriptions adds the enabled field to the subscriptions
// table. Three cases this handles, all of which are valid no-ops or successes:
//
//  1. Table exists, column missing  → ALTER TABLE succeeds (real migration).
//  2. Table exists, column present  → ALTER TABLE fails with "duplicate column";
//     the model already has it, treat as success.
//  3. Table does not exist yet      → migrations run before manager Sync2 on a
//     fresh DB. The table will be created by Sync2 *with* the enabled column
//     already in the struct, so there is nothing for this migration to do —
//     treat as success.
func migrate001AddEnabledToSubscriptions(session *xorm.Session) error {
	sql := `ALTER TABLE subscriptions ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT 1`
	_, err := session.Exec(sql)
	if err == nil {
		logger.Info("Added enabled column to subscriptions table")
		return nil
	}

	if isDuplicateColumnError(err) {
		logger.Info("Enabled column already exists in subscriptions table")
		return nil
	}

	if isNoSuchTableError(err) {
		// Fresh database: subscriptions table will be created by the
		// subscription manager's Sync2 later, with the enabled column already
		// declared on the model. Nothing to migrate.
		logger.Info("subscriptions table does not exist yet — fresh DB, will be created with enabled column by Sync2")
		return nil
	}

	return fmt.Errorf("failed to add enabled column: %w", err)
}

// isDuplicateColumnError checks if the error indicates a duplicate column name.
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate column name") ||
		strings.Contains(errStr, "column exists")
}

// isNoSuchTableError matches SQLite's "no such table: X" error. This indicates
// the migration is running against a fresh database where the base schema has
// not been created yet — managers do their Sync2 lazily after database.Init()
// returns. For migrations that are merely adding columns the manager's model
// already declares, this is a no-op rather than a failure.
func isNoSuchTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}