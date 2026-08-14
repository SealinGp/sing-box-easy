package service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// procdBackend manages sing-box through its OpenWrt procd init script
// (/etc/init.d/sing-box). Logs come from the syslog ring buffer via logread,
// unless the sing-box config points log.output at a file.
type procdBackend struct {
	initScript string
	// logPath returns the configured sing-box log.output file path, or ""
	// when logging goes to stdout/syslog.
	logPath func() string
}

func (b *procdBackend) Kind() string { return BackendProcd }

// initd runs `/etc/init.d/sing-box <action>` and wraps failures with the
// captured output for easier diagnosis.
func (b *procdBackend) initd(action string) error {
	out, err := exec.Command(b.initScript, action).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s",
			b.initScript, action, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *procdBackend) Start() error {
	if err := b.initd("start"); err != nil {
		return err
	}
	logger.Info("Service started via procd")
	return nil
}

// Stop delegates to the init script and then waits for the process to
// actually exit, since procd stop returns before the child is reaped.
func (b *procdBackend) Stop() error {
	if err := b.initd("stop"); err != nil {
		return err
	}
	if err := waitForPIDExit(SystemOpenWRT, stopGracePeriod); err != nil {
		logger.Warn("Service did not stop gracefully, sending SIGKILL", zap.Error(err))
		if killErr := signalRunningPID(SystemOpenWRT, signalKill); killErr != nil {
			return killErr
		}
		if err := waitForPIDExit(SystemOpenWRT, stopGracePeriod); err != nil {
			return err
		}
	}
	logger.Info("Service stopped via procd")
	return nil
}

func (b *procdBackend) ForceStop() error {
	// Ask procd to stop first so it does not respawn the process, then kill
	// any lingering process directly.
	if err := b.initd("stop"); err != nil {
		logger.Warn("procd stop failed during force stop", zap.Error(err))
	}
	if err := signalRunningPID(SystemOpenWRT, signalKill); err != nil {
		return err
	}
	logger.Info("Service force stopped via procd")
	return nil
}

func (b *procdBackend) Restart() error {
	if err := b.initd("restart"); err != nil {
		return err
	}
	logger.Info("Service restarted via procd")
	return nil
}

// Reload tries the init script's reload action (procd scripts commonly map it
// to SIGHUP via reload_service), falling back to a full restart when the
// script does not support it.
func (b *procdBackend) Reload() error {
	if err := b.initd("reload"); err != nil {
		logger.Warn("procd reload failed, falling back to restart", zap.Error(err))
		return b.Restart()
	}
	logger.Info("Service reloaded via procd")
	return nil
}

// ActiveAndPID resolves the running state from the live process rather than
// procd bookkeeping: `/etc/init.d/<s> running` requires a procd-tracked
// instance, whereas pidof also covers manually started processes.
func (b *procdBackend) ActiveAndPID() (bool, int, error) {
	return lookupRunningPID(SystemOpenWRT)
}

// TailLogs prefers the configured log.output file (when set, sing-box writes
// there and syslog stays empty); otherwise it filters the syslog ring buffer
// via logread. logread has no cursor support, so incremental polling always
// returns the full tail window.
func (b *procdBackend) TailLogs(lines int, _ string) (LogChunk, error) {
	if path := b.logPath(); path != "" {
		return tailFile(path, lines)
	}
	return tailLogread(lines)
}
