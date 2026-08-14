package service

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/process"
	"go.uber.org/zap"
)

// processBackend manages sing-box directly: it spawns the process itself via
// ProcessManager and signals it by PID. Used when neither systemd nor a procd
// init script manages sing-box.
//
// mu guards only the pm field (the live process handle). Full lifecycle
// sequences ("is it running → spawn") are NOT serialized here — the
// Controller's lifecycleMu does that across all backends, so concurrent
// Start/Stop/Restart calls never interleave.
type processBackend struct {
	systemType  SystemType
	singBoxPath string
	// configPath returns the path of the active sing-box config file.
	configPath func() string
	// logPath returns the configured sing-box log.output file path, or ""
	// when logging goes to stdout (no file to tail).
	logPath func() string

	mu sync.Mutex
	pm *process.ProcessManager
}

func newProcessBackend(systemType SystemType, singBoxPath string, configPath, logPath func() string) *processBackend {
	return &processBackend{
		systemType:  systemType,
		singBoxPath: singBoxPath,
		configPath:  configPath,
		logPath:     logPath,
	}
}

func (b *processBackend) Kind() string { return BackendProcess }

func (b *processBackend) Start() error {
	// Initialize a new ProcessManager for this run
	pm := process.NewProcessManager()

	// Start sing-box service using ProcessManager
	if err := pm.Start(b.singBoxPath, "run", "-c", b.configPath()); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	b.mu.Lock()
	b.pm = pm
	b.mu.Unlock()

	logChan := pm.GetLogChanel()
	errCh := make(chan error, 1)

	// Monitor logs and catch FATAL errors
	go func() {
		for msg := range logChan {
			logger.Info("sing-box: " + msg)
			if strings.Contains(msg, "FATAL") {
				select {
				case errCh <- errors.New(msg):
				default:
				}
			}
		}
	}()

	// Wait for 2 seconds to check if process fails immediately
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Process seems to be running fine after 2 seconds
		logger.Info("Service started successfully")
		return nil
	}
}

// Stop stops the sing-box service.
//
// If the running process was started by this backend, it is stopped via the
// ProcessManager (context cancellation). If sing-box was started outside of
// sing-box-easy (b.pm == nil), it is terminated by PID with SIGTERM. In both
// cases we wait for the process to actually exit so callers such as Restart
// can safely start a new instance, escalating to SIGKILL if it lingers.
func (b *processBackend) Stop() error {
	running, _, err := b.ActiveAndPID()
	if err != nil {
		return err
	}
	if !running {
		return nil
	}

	b.mu.Lock()
	pm := b.pm
	b.pm = nil
	b.mu.Unlock()

	if pm != nil {
		pm.Stop()
	} else {
		// Process was started outside of sing-box-easy; terminate it by PID.
		if err := signalRunningPID(b.systemType, signalTerm); err != nil {
			return err
		}
	}

	// Wait for the process to actually exit. Escalate to SIGKILL if it does
	// not stop within the grace period (covers both the pm and external
	// cases).
	if err := waitForPIDExit(b.systemType, stopGracePeriod); err != nil {
		logger.Warn("Service did not stop gracefully, sending SIGKILL", zap.Error(err))
		if killErr := signalRunningPID(b.systemType, signalKill); killErr != nil {
			return killErr
		}
		if err := waitForPIDExit(b.systemType, stopGracePeriod); err != nil {
			return err
		}
	}

	logger.Info("Service stopped")
	return nil
}

// ForceStop force stops the sing-box service using SIGKILL. As with Stop, if
// the process was started outside of sing-box-easy (b.pm == nil) it is killed
// by PID.
func (b *processBackend) ForceStop() error {
	b.mu.Lock()
	pm := b.pm
	b.pm = nil
	b.mu.Unlock()

	if pm != nil {
		pm.Stop()
	} else {
		// Process was started outside of sing-box-easy; kill it by PID.
		if err := signalRunningPID(b.systemType, signalKill); err != nil {
			return err
		}
	}
	logger.Info("Service force stopped")
	return nil
}

func (b *processBackend) Restart() error {
	running, _, err := b.ActiveAndPID()
	if err != nil {
		return err
	}
	if running {
		if err := b.Stop(); err != nil {
			return err
		}
	}
	return b.Start()
}

// Reload sends SIGHUP to reload configuration without restarting, falling
// back to a full restart when the signal cannot be delivered.
func (b *processBackend) Reload() error {
	running, pid, err := b.ActiveAndPID()
	if err != nil {
		return err
	}
	if !running || pid <= 0 {
		return fmt.Errorf("service is not running")
	}

	// Send SIGHUP to reload
	if err := exec.Command("kill", "-HUP", strconv.Itoa(pid)).Run(); err != nil {
		// If SIGHUP doesn't work, try restart
		logger.Warn("SIGHUP reload failed, falling back to restart", zap.Error(err))
		return b.Restart()
	}

	logger.Info("Service reloaded", zap.Int("pid", pid))
	return nil
}

func (b *processBackend) ActiveAndPID() (bool, int, error) {
	return lookupRunningPID(b.systemType)
}

// TailLogs best-effort tails the configured log.output file. When sing-box
// logs to stdout only, the lines are captured by the application logger
// instead, so Source="none" tells the UI to explain the situation.
func (b *processBackend) TailLogs(lines int, _ string) (LogChunk, error) {
	path := b.logPath()
	if path == "" {
		return LogChunk{Lines: []string{}, Source: LogSourceNone}, nil
	}
	return tailFile(path, lines)
}
