package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

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

	// runInit and probe are seams: nil means "use the real implementation".
	// Tests substitute them to exercise the lifecycle without an init script
	// or a process table.
	runInit func(action string) error
	probe   func() (bool, int, error)
	// startTimeout overrides startGracePeriod; zero means use the default.
	startTimeout time.Duration
}

func (b *procdBackend) Kind() string { return BackendProcd }

// invoke runs an init-script action through the injected seam when present.
func (b *procdBackend) invoke(action string) error {
	if b.runInit != nil {
		return b.runInit(action)
	}
	return b.initd(action)
}

// status reports whether sing-box is running, through the injected seam when
// present.
func (b *procdBackend) status() (bool, int, error) {
	if b.probe != nil {
		return b.probe()
	}
	return lookupRunningPID(SystemOpenWRT)
}

func (b *procdBackend) startWait() time.Duration {
	if b.startTimeout > 0 {
		return b.startTimeout
	}
	return startGracePeriod
}

// confirmStarted verifies sing-box actually came up after the init script
// accepted the request.
//
// This is not paranoia: OpenWrt's UCI-driven /etc/init.d/sing-box checks
// `config_get_bool enabled "main" "enabled" "0"` and returns 0 *without
// starting anything* when the service is disabled in UCI. Treating that exit
// code as success reports a running service that does not exist.
func (b *procdBackend) confirmStarted(action string) error {
	if err := waitForServiceStart(b.status, b.startWait(), startPollInterval); err != nil {
		return fmt.Errorf(
			"%s %s exited successfully but sing-box is not running "+
				"(on OpenWrt check `uci get sing-box.main.enabled` — a value of 0 makes "+
				"the init script a no-op — then `logread | grep sing-box`): %w",
			b.initScript, action, err)
	}
	return nil
}

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
	if err := b.invoke("start"); err != nil {
		return err
	}
	if err := b.confirmStarted("start"); err != nil {
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
	if err := b.invoke("restart"); err != nil {
		return err
	}
	if err := b.confirmStarted("restart"); err != nil {
		return err
	}
	logger.Info("Service restarted via procd")
	return nil
}

// Reload tries the init script's reload action (procd scripts commonly map it
// to SIGHUP via reload_service), falling back to a full restart when the
// script does not support it.
//
// Reload-while-stopped semantics are delegated to the out-of-tree init
// script: rc.common's default reload falls back to restart, and the OpenWrt
// sing-box package's reload_service restarts a stopped instance, so a reload
// on a stopped service brings it up rather than erroring — matching the
// systemd backend's reload-or-restart behavior.
func (b *procdBackend) Reload() error {
	if err := b.invoke("reload"); err != nil {
		logger.Warn("procd reload failed, falling back to restart", zap.Error(err))
		return b.Restart()
	}
	// A reload that leaves nothing running is the same silent no-op as a
	// start against a UCI-disabled instance, so escalate to a full restart
	// rather than reporting success.
	if err := b.confirmStarted("reload"); err != nil {
		logger.Warn("procd reload left sing-box stopped, falling back to restart",
			zap.Error(err))
		return b.Restart()
	}
	logger.Info("Service reloaded via procd")
	return nil
}

// ActiveAndPID resolves the running state from the live process rather than
// procd bookkeeping: `/etc/init.d/<s> running` requires a procd-tracked
// instance, whereas pidof also covers manually started processes.
func (b *procdBackend) ActiveAndPID() (bool, int, error) {
	return b.status()
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

// FollowLogs follows the configured log file when there is one, and otherwise
// follows the syslog ring buffer via `logread -f`.
//
// The same tag filter the tail path uses is applied here, and for the same
// reason: this panel logs to syslog too, so an unfiltered follow interleaves
// our own JSON into the sing-box log view.
func (b *procdBackend) FollowLogs(ctx context.Context) (<-chan FollowEvent, error) {
	if path := b.logPath(); path != "" {
		return followFile(ctx, path)
	}
	return startCommand(ctx, "logread", []string{"-f"}, keepSingBoxSyslogLine)
}
