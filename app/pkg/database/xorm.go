package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
	"xorm.io/xorm"
	"xorm.io/xorm/log"

	_ "github.com/mattn/go-sqlite3"
)

var (
	engine  *xorm.Engine
	once    sync.Once
	dbMutex sync.RWMutex
)

const (
	DefaultDatabasePath = "/etc/sing-box/sing-box-easy.db"
)

// Init initializes the XORM engine
func Init(dbPath string) error {
	var initErr error
	once.Do(func() {
		if dbPath == "" {
			dbPath = DefaultDatabasePath
		}

		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		// Create XORM engine
		eng, err := xorm.NewEngine("sqlite3", dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to create xorm engine: %w", err)
			return
		}

		// Configure XORM
		eng.SetMaxOpenConns(1) // SQLite works best with single connection
		eng.SetMaxIdleConns(1)

		// Set logger
		eng.SetLogger(&xormLogger{})
		eng.ShowSQL(false) // Set to true for debugging

		// Test connection
		if err := eng.Ping(); err != nil {
			eng.Close()
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		engine = eng

		// Run migrations
		if err := runMigrations(); err != nil {
			engine.Close()
			engine = nil
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		logger.Info("Database initialized successfully with XORM", zap.String("path", dbPath))
	})

	return initErr
}

// GetEngine returns the XORM engine instance
func GetEngine() (*xorm.Engine, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	if engine == nil {
		return nil, fmt.Errorf("database not initialized, call Init() first")
	}
	return engine, nil
}

// Close closes the database connection
func Close() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if engine != nil {
		err := engine.Close()
		engine = nil
		return err
	}
	return nil
}

// xormLogger adapts zap logger to XORM logger interface
type xormLogger struct{}

func (l *xormLogger) Debug(v ...interface{}) {
	logger.Debug(fmt.Sprint(v...))
}

func (l *xormLogger) Debugf(format string, v ...interface{}) {
	logger.Debugf(format, v...)
}

func (l *xormLogger) Info(v ...interface{}) {
	logger.Info(fmt.Sprint(v...))
}

func (l *xormLogger) Infof(format string, v ...interface{}) {
	logger.Infof(format, v...)
}

func (l *xormLogger) Warn(v ...interface{}) {
	logger.Warn(fmt.Sprint(v...))
}

func (l *xormLogger) Warnf(format string, v ...interface{}) {
	logger.Warnf(format, v...)
}

func (l *xormLogger) Error(v ...interface{}) {
	logger.Error(fmt.Sprint(v...))
}

func (l *xormLogger) Errorf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
}

func (l *xormLogger) Level() log.LogLevel {
	return log.LOG_INFO
}

func (l *xormLogger) SetLevel(level log.LogLevel) {}

func (l *xormLogger) ShowSQL(show ...bool) {}

func (l *xormLogger) IsShowSQL() bool {
	return false
}
