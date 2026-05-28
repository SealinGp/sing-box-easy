package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/process"
	"go.uber.org/zap"
)

// SystemType represents the type of operating system
type SystemType string

const (
	SystemDebian  SystemType = "debian"
	SystemOpenWRT SystemType = "openwrt"
	SystemUnknown SystemType = "unknown"
)

// Signal flags passed to the `kill` command when terminating a sing-box process
// that this controller does not own (c.pm == nil).
const (
	signalTerm = "-TERM" // graceful shutdown
	signalKill = "-KILL" // force kill
)

// Timing for waiting on a process to actually exit after a stop signal.
const (
	stopGracePeriod  = 5 * time.Second
	stopPollInterval = 100 * time.Millisecond
)

// singBoxServiceName is the systemd unit name used by the official sing-box
// Debian/Linux installation.
const singBoxServiceName = "sing-box.service"

// Controller manages sing-box service lifecycle.
//
// HTTP endpoints can call Start/Stop/Restart/ForceStop concurrently, so all
// access to pm (the live process handle) is serialized via mu.
type Controller struct {
	configManager *config.Manager
	singBoxPath   string
	systemType    SystemType

	// useSystemd is true when sing-box is installed as a systemd unit (e.g. the
	// official Debian install). In that case lifecycle operations are delegated
	// to systemctl instead of being managed directly via ProcessManager.
	useSystemd bool

	mu sync.Mutex
	pm *process.ProcessManager
}

// NewController creates a new service controller
func NewController(configManager *config.Manager, singBoxPath string) *Controller {
	if singBoxPath == "" {
		singBoxPath = "sing-box"
	}

	controller := &Controller{
		configManager: configManager,
		singBoxPath:   singBoxPath,
		systemType:    detectSystemType(),
		useSystemd:    detectSystemd(),
	}

	logger.Info("Service controller initialized",
		zap.String("system", string(controller.systemType)),
		zap.Bool("systemd_managed", controller.useSystemd),
		zap.String("os", runtime.GOOS),
		zap.String("arch", runtime.GOARCH))

	return controller
}

// detectSystemd reports whether sing-box is managed by a systemd unit on this
// host. It is true only when `systemctl` is available AND a sing-box unit file
// exists, which is the case for the official Debian/Linux installation. OpenWRT
// (procd-based) and hosts without systemd return false and fall back to direct
// process management.
func detectSystemd() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	// `systemctl cat <unit>` exits 0 only when the unit file exists; output is
	// irrelevant, we only care about the exit status.
	if err := exec.Command("systemctl", "cat", singBoxServiceName).Run(); err != nil {
		return false
	}
	return true
}

// systemctl runs `systemctl <args...> sing-box.service` and wraps failures with
// the captured output for easier diagnosis.
func (c *Controller) systemctl(args ...string) error {
	full := append(append([]string{}, args...), singBoxServiceName)
	out, err := exec.Command("systemctl", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// systemctlIsActive reports whether the sing-box systemd unit is currently
// active. `systemctl is-active` exits non-zero for inactive/failed/unknown
// states, which is expected and not treated as an operational error.
func (c *Controller) systemctlIsActive() (bool, error) {
	out, _ := exec.Command("systemctl", "is-active", singBoxServiceName).Output()
	return strings.TrimSpace(string(out)) == "active", nil
}

// detectSystemType detects whether the system is OpenWRT, Debian, or other
func detectSystemType() SystemType {
	// Check for OpenWRT first
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		return SystemOpenWRT
	}

	// Check for Debian-based systems
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return SystemDebian
	}

	// Check /etc/os-release for more info
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		if strings.Contains(strings.ToLower(content), "openwrt") {
			return SystemOpenWRT
		}
		if strings.Contains(strings.ToLower(content), "debian") ||
			strings.Contains(strings.ToLower(content), "ubuntu") {
			return SystemDebian
		}
	}

	// Default to unknown, but will try compatible commands
	return SystemUnknown
}

// getPID returns the PID of the sing-box process, or empty string if not running
func (c *Controller) getPID() (string, error) {
	var cmd *exec.Cmd
	var output []byte
	var err error

	switch c.systemType {
	case SystemOpenWRT:
		// Use pidof for OpenWRT (more commonly available)
		cmd = exec.Command("pidof", "sing-box")
		output, err = cmd.Output()
		if err != nil {
			// Process not found
			return "", nil
		}
		// pidof might return multiple PIDs, take the first one
		pids := strings.Fields(strings.TrimSpace(string(output)))
		if len(pids) > 0 {
			return pids[0], nil
		}
		return "", nil

	case SystemDebian, SystemUnknown:
		// Try pgrep first (most efficient)
		cmd = exec.Command("pgrep", "-x", "sing-box")
		output, err = cmd.Output()
		if err == nil {
			pid := strings.TrimSpace(string(output))
			if pid != "" {
				return strings.Split(pid, "\n")[0], nil // Return first PID if multiple
			}
		}

		// Fallback to ps with grep (most compatible)
		cmd = exec.Command("sh", "-c", "ps aux | grep -v grep | grep 'sing-box\\s*$' | awk '{print $2}'")
		output, err = cmd.Output()
		if err == nil {
			pid := strings.TrimSpace(string(output))
			if pid != "" {
				return strings.Split(pid, "\n")[0], nil // Return first PID if multiple
			}
		}

		return "", nil
	}

	return "", nil
}

// Status returns the current status of sing-box service
func (c *Controller) Status() (bool, error) {
	if c.useSystemd {
		return c.systemctlIsActive()
	}

	pid, err := c.getPID()
	if err != nil {
		return false, fmt.Errorf("failed to check service status: %w", err)
	}
	return pid != "", nil
}

// Start starts the sing-box service
func (c *Controller) Start() error {
	// Log system info
	logger.Info("Starting sing-box service",
		zap.String("system_type", string(c.systemType)),
		zap.String("sing_box_path", c.singBoxPath),
		zap.String("config_path", c.configManager.GetConfigPath()))

	// Check if already running
	running, err := c.Status()
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("service is already running")
	}

	// Validate config before starting
	cfg, err := c.configManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := c.configManager.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Delegate to systemd when sing-box is managed by a systemd unit.
	if c.useSystemd {
		if err := c.systemctl("start"); err != nil {
			return err
		}
		logger.Info("Service started via systemd")
		return nil
	}

	return c.serveSingBox()
}

func (c *Controller) serveSingBox() error {
	// Initialize a new ProcessManager for this run
	pm := process.NewProcessManager()

	// Start sing-box service using ProcessManager
	if err := pm.Start(c.singBoxPath, "run", "-c", c.configManager.GetConfigPath()); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	c.mu.Lock()
	c.pm = pm
	c.mu.Unlock()

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
// If the running process was started by this controller, it is stopped via the
// ProcessManager (context cancellation). If sing-box was started outside of
// sing-box-easy (c.pm == nil), it is terminated by PID with SIGTERM. In both
// cases we wait for the process to actually exit so callers such as Restart can
// safely start a new instance, escalating to SIGKILL if it lingers.
func (c *Controller) Stop() error {
	// Delegate to systemd when sing-box is managed by a systemd unit.
	// `systemctl stop` is idempotent and blocks until the unit has stopped.
	if c.useSystemd {
		if err := c.systemctl("stop"); err != nil {
			return err
		}
		logger.Info("Service stopped via systemd")
		return nil
	}

	// Check if running
	running, err := c.Status()
	if err != nil {
		return err
	}
	if !running {
		// return fmt.Errorf("service is not running")
		return nil
	}

	c.mu.Lock()
	pm := c.pm
	c.pm = nil
	c.mu.Unlock()

	if pm != nil {
		pm.Stop()
	} else {
		// Process was started outside of sing-box-easy; terminate it by PID.
		if err := c.signalPID(signalTerm); err != nil {
			return err
		}
	}

	// Wait for the process to actually exit. Escalate to SIGKILL if it does not
	// stop within the grace period (covers both the pm and external cases).
	if err := c.waitForStop(stopGracePeriod); err != nil {
		logger.Warn("Service did not stop gracefully, sending SIGKILL", zap.Error(err))
		if killErr := c.signalPID(signalKill); killErr != nil {
			return killErr
		}
		if err := c.waitForStop(stopGracePeriod); err != nil {
			return err
		}
	}

	logger.Info("Service stopped")
	return nil
}

// signalPID looks up the running sing-box PID and sends it the given signal via
// the `kill` command. It is used to control processes this controller does not
// own. A nil error is returned when no process is running.
func (c *Controller) signalPID(sig string) error {
	pid, err := c.getPID()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %w", err)
	}
	if pid == "" {
		return nil
	}

	// Validate pid is a positive integer before passing it to exec.
	// pid originates from `pgrep`/`pidof`/`ps` output, which is normally safe,
	// but never trust an external string flowing into an exec argument list.
	pidNum, err := strconv.Atoi(pid)
	if err != nil || pidNum <= 0 {
		return fmt.Errorf("invalid pid from process lookup: %q", pid)
	}

	if err := exec.Command("kill", sig, strconv.Itoa(pidNum)).Run(); err != nil {
		return fmt.Errorf("failed to send %s to pid %d: %w", sig, pidNum, err)
	}
	return nil
}

// waitForStop polls process status until the service has stopped or the timeout
// elapses.
func (c *Controller) waitForStop(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		running, err := c.Status()
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("process still running after %s", timeout)
		}
		time.Sleep(stopPollInterval)
	}
}

// Restart restarts the sing-box service
func (c *Controller) Restart() error {
	// Validate config before restarting
	cfg, err := c.configManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := c.configManager.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Delegate to systemd when sing-box is managed by a systemd unit.
	// `systemctl restart` starts the unit even if it was stopped.
	if c.useSystemd {
		if err := c.systemctl("restart"); err != nil {
			return err
		}
		logger.Info("Service restarted via systemd")
		return nil
	}

	// Check if running
	running, err := c.Status()
	if err != nil {
		return err
	}

	if running {
		// Stop first
		if err := c.Stop(); err != nil {
			return err
		}
	}

	// Start
	return c.Start()
}

// Reload sends SIGHUP to reload configuration without restarting
func (c *Controller) Reload() error {
	// Validate config before reloading
	cfg, err := c.configManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := c.configManager.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Delegate to systemd when sing-box is managed by a systemd unit.
	// `reload-or-restart` reloads if the unit defines ExecReload, otherwise it
	// performs a full restart — mirroring the SIGHUP-with-restart-fallback below.
	if c.useSystemd {
		if err := c.systemctl("reload-or-restart"); err != nil {
			return err
		}
		logger.Info("Service reloaded via systemd")
		return nil
	}

	// Check if running
	running, err := c.Status()
	if err != nil {
		return err
	}
	if !running {
		return fmt.Errorf("service is not running")
	}

	// Get PID
	pid, err := c.getPID()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %w", err)
	}
	if pid == "" {
		return fmt.Errorf("service is not running")
	}

	// Validate pid is a positive integer before passing it to exec.
	// pid originates from `pgrep`/`pidof`/`ps` output, which is normally safe,
	// but never trust an external string flowing into an exec argument list.
	pidNum, err := strconv.Atoi(pid)
	if err != nil || pidNum <= 0 {
		return fmt.Errorf("invalid pid from process lookup: %q", pid)
	}

	// Send SIGHUP to reload
	reloadCmd := exec.Command("kill", "-HUP", strconv.Itoa(pidNum))
	if err := reloadCmd.Run(); err != nil {
		// If SIGHUP doesn't work, try restart
		logger.Warn("SIGHUP reload failed, falling back to restart", zap.Error(err))
		return c.Restart()
	}

	logger.Info("Service reloaded", zap.String("pid", pid))
	return nil
}

// ForceStop force stops the sing-box service using SIGKILL.
//
// As with Stop, if the process was started outside of sing-box-easy
// (c.pm == nil) it is killed by PID.
func (c *Controller) ForceStop() error {
	// Delegate to systemd when sing-box is managed by a systemd unit.
	if c.useSystemd {
		if err := c.systemctl("kill", "--signal=SIGKILL"); err != nil {
			return err
		}
		logger.Info("Service force stopped via systemd")
		return nil
	}

	c.mu.Lock()
	pm := c.pm
	c.pm = nil
	c.mu.Unlock()

	if pm != nil {
		pm.Stop()
	} else {
		// Process was started outside of sing-box-easy; kill it by PID.
		if err := c.signalPID(signalKill); err != nil {
			return err
		}
	}
	logger.Info("Service force stopped")
	return nil
}
