package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger
var Sugar *zap.SugaredLogger
var L *zap.Logger // Alias for Logger for convenience

// ParseLevel converts a string level to zapcore.Level.
// Accepted values (case-insensitive): debug, info, warn, warning, error.
// Returns an error for unknown levels.
func ParseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "", "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %q (expected debug|info|warn|error)", level)
	}
}

// Init initializes the global logger with the given level string.
// Debug level uses the development encoder (colorized, human-readable);
// info/warn/error use the production encoder (JSON, ISO8601 timestamps).
func Init(level string) error {
	lvl, err := ParseLevel(level)
	if err != nil {
		return err
	}

	var config zap.Config
	if lvl == zapcore.DebugLevel {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}
	config.Level = zap.NewAtomicLevelAt(lvl)

	// Always output to stdout
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	built, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	Logger = built
	L = Logger // Set alias
	Sugar = Logger.Sugar()
	return nil
}

// InitDefault initializes with default settings.
// Reads DEBUG=true for debug level, otherwise defaults to info.
// Falls back to a basic example logger on failure so the app can still start.
func InitDefault() {
	level := "info"
	if os.Getenv("DEBUG") == "true" {
		level = "debug"
	}

	if err := Init(level); err != nil {
		// Fallback to a basic logger
		Logger = zap.NewExample()
		L = Logger
		Sugar = Logger.Sugar()
	}
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// Convenience functions
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// Sugar variants for easier formatting
func Debugf(template string, args ...interface{}) {
	Sugar.Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	Sugar.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	Sugar.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	Sugar.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	Sugar.Fatalf(template, args...)
}
