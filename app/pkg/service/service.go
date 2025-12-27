package service

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// SystemType represents the type of operating system
type SystemType string

const (
	SystemDebian  SystemType = "debian"
	SystemOpenWRT SystemType = "openwrt"
	SystemUnknown SystemType = "unknown"
)

// Controller manages sing-box service lifecycle
type Controller struct {
	configManager *config.Manager
	singBoxPath   string
	systemType    SystemType
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
	}

	logger.Info("Service controller initialized",
		zap.String("system", string(controller.systemType)),
		zap.String("os", runtime.GOOS),
		zap.String("arch", runtime.GOARCH))

	return controller
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

	// Start sing-box service
	cmd := exec.Command(c.singBoxPath, "run", "-c", c.configManager.GetConfigPath())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	logger.Info("Service started successfully", zap.Int("pid", cmd.Process.Pid))
	return nil
}

// Stop stops the sing-box service
func (c *Controller) Stop() error {
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

	// Send SIGTERM to gracefully stop
	stopCmd := exec.Command("kill", "-TERM", pid)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	logger.Info("Service stopped", zap.String("pid", pid))
	return nil
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

	// Send SIGHUP to reload
	reloadCmd := exec.Command("kill", "-HUP", pid)
	if err := reloadCmd.Run(); err != nil {
		// If SIGHUP doesn't work, try restart
		logger.Warn("SIGHUP reload failed, falling back to restart", zap.Error(err))
		return c.Restart()
	}

	logger.Info("Service reloaded", zap.String("pid", pid))
	return nil
}

// ForceStop force stops the sing-box service using SIGKILL
func (c *Controller) ForceStop() error {
	// Get PID
	pid, err := c.getPID()
	if err != nil {
		// Log error but don't fail
		logger.Warn("Error getting PID during force stop", zap.Error(err))
		return nil
	}
	if pid == "" {
		// Process not running
		return nil
	}

	// Send SIGKILL
	stopCmd := exec.Command("kill", "-9", pid)
	if err := stopCmd.Run(); err != nil {
		// Try using syscall directly
		var pidInt int
		fmt.Sscanf(pid, "%d", &pidInt)
		if pidInt > 0 {
			syscall.Kill(pidInt, syscall.SIGKILL)
		}
	}

	logger.Info("Service force stopped", zap.String("pid", pid))
	return nil
}
