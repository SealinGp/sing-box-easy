package service

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

// Controller manages sing-box service lifecycle
type Controller struct {
	configManager *config.Manager
	singBoxPath   string
}

// NewController creates a new service controller
func NewController(configManager *config.Manager, singBoxPath string) *Controller {
	if singBoxPath == "" {
		singBoxPath = "sing-box"
	}
	return &Controller{
		configManager: configManager,
		singBoxPath:   singBoxPath,
	}
}

// Status returns the current status of sing-box service
func (c *Controller) Status() (bool, error) {
	// Check if sing-box process is running
	cmd := exec.Command("pgrep", "-x", "sing-box")
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// pgrep returns exit code 1 if no process found
			if exitError.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check service status: %w", err)
	}
	return true, nil
}

// Start starts the sing-box service
func (c *Controller) Start() error {
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
	cmd := exec.Command("pgrep", "-x", "sing-box")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %w", err)
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return fmt.Errorf("failed to get service PID")
	}

	// Send SIGTERM to gracefully stop
	stopCmd := exec.Command("kill", "-TERM", pid)
	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

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
	cmd := exec.Command("pgrep", "-x", "sing-box")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %w", err)
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return fmt.Errorf("failed to get service PID")
	}

	// Send SIGHUP to reload
	reloadCmd := exec.Command("kill", "-HUP", pid)
	if err := reloadCmd.Run(); err != nil {
		// If SIGHUP doesn't work, try restart
		return c.Restart()
	}

	return nil
}

// ForceStop force stops the sing-box service using SIGKILL
func (c *Controller) ForceStop() error {
	// Get PID
	cmd := exec.Command("pgrep", "-x", "sing-box")
	output, err := cmd.Output()
	if err != nil {
		// Process might not be running
		return nil
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
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

	return nil
}
