package logger

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"
)

// HertzLogger is a wrapper around zap.Logger that implements Hertz's logger interface
type HertzLogger struct {
	logger *zap.Logger
}

// NewHertzLogger creates a new Hertz logger using zap
func NewHertzLogger(logger *zap.Logger) hlog.FullLogger {
	return &HertzLogger{
		logger: logger,
	}
}

func (l *HertzLogger) Trace(v ...interface{}) {
	l.logger.Debug(formatArgs(v...))
}

func (l *HertzLogger) Debug(v ...interface{}) {
	l.logger.Debug(formatArgs(v...))
}

func (l *HertzLogger) Info(v ...interface{}) {
	l.logger.Info(formatArgs(v...))
}

func (l *HertzLogger) Notice(v ...interface{}) {
	l.logger.Info(formatArgs(v...))
}

func (l *HertzLogger) Warn(v ...interface{}) {
	l.logger.Warn(formatArgs(v...))
}

func (l *HertzLogger) Error(v ...interface{}) {
	l.logger.Error(formatArgs(v...))
}

func (l *HertzLogger) Fatal(v ...interface{}) {
	l.logger.Fatal(formatArgs(v...))
}

func (l *HertzLogger) Tracef(format string, v ...interface{}) {
	l.logger.Sugar().Debugf(format, v...)
}

func (l *HertzLogger) Debugf(format string, v ...interface{}) {
	l.logger.Sugar().Debugf(format, v...)
}

func (l *HertzLogger) Infof(format string, v ...interface{}) {
	l.logger.Sugar().Infof(format, v...)
}

func (l *HertzLogger) Noticef(format string, v ...interface{}) {
	l.logger.Sugar().Infof(format, v...)
}

func (l *HertzLogger) Warnf(format string, v ...interface{}) {
	l.logger.Sugar().Warnf(format, v...)
}

func (l *HertzLogger) Errorf(format string, v ...interface{}) {
	l.logger.Sugar().Errorf(format, v...)
}

func (l *HertzLogger) Fatalf(format string, v ...interface{}) {
	l.logger.Sugar().Fatalf(format, v...)
}

func (l *HertzLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Debugf(format, v...)
}

func (l *HertzLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Debugf(format, v...)
}

func (l *HertzLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Infof(format, v...)
}

func (l *HertzLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Infof(format, v...)
}

func (l *HertzLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Warnf(format, v...)
}

func (l *HertzLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Errorf(format, v...)
}

func (l *HertzLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	l.logger.Sugar().Fatalf(format, v...)
}

func (l *HertzLogger) SetLevel(level hlog.Level) {
	// Note: zap level is set at initialization, can't be changed dynamically
}

func (l *HertzLogger) SetOutput(writer io.Writer) {
	// Note: zap output is configured at initialization
}

// Helper function to format args
func formatArgs(v ...interface{}) string {
	if len(v) == 0 {
		return ""
	}
	return fmt.Sprint(v...)
}
