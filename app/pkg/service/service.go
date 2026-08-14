package service

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// Controller manages the sing-box service lifecycle. It owns config
// validation and delegates the actual lifecycle operations to the detected
// Backend (systemd, procd, or direct process management).
type Controller struct {
	configManager *config.Manager
	singBoxPath   string
	systemType    SystemType
	backend       Backend

	// lifecycleMu serializes the state-changing operations
	// (Start/Stop/Restart/Reload/ForceStop) end to end. Without it, two
	// concurrent Starts can both observe "not running" and both spawn a
	// process — the second overwriting the first one's handle, leaving an
	// untracked sing-box behind. Read-only calls (Status/Info/TailLogs) stay
	// lock-free so a slow stop never blocks the status endpoint.
	lifecycleMu sync.Mutex
}

// NewController creates a new service controller.
func NewController(configManager *config.Manager, singBoxPath string) *Controller {
	if singBoxPath == "" {
		singBoxPath = "sing-box"
	}

	systemType := DetectSystemType()
	controller := &Controller{
		configManager: configManager,
		singBoxPath:   singBoxPath,
		systemType:    systemType,
	}
	controller.backend = detectBackend(
		systemType,
		singBoxPath,
		configManager.GetConfigPath,
		controller.logOutputPath,
	)

	logger.Info("Service controller initialized",
		zap.String("system", string(systemType)),
		zap.String("backend", controller.backend.Kind()),
		zap.String("os", runtime.GOOS),
		zap.String("arch", runtime.GOARCH))

	return controller
}

// validateConfig reads and validates the current config; every state-changing
// operation goes through this before touching the service.
func (c *Controller) validateConfig() error {
	cfg, err := c.configManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	if err := c.configManager.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	return nil
}

// BackendKind reports which init system drives sing-box on this host — one of
// the Backend* constants. Surfaced in the UI's "About" panel so an operator can
// tell at a glance whether lifecycle commands go through systemd, procd, or a
// directly-spawned process.
func (c *Controller) BackendKind() string {
	return c.backend.Kind()
}

// SystemType reports the detected distribution family.
func (c *Controller) SystemType() SystemType {
	return c.systemType
}

// Status returns whether the sing-box service is currently running.
func (c *Controller) Status() (bool, error) {
	running, _, err := c.backend.ActiveAndPID()
	return running, err
}

// Start validates the config and starts the sing-box service.
func (c *Controller) Start() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()

	logger.Info("Starting sing-box service",
		zap.String("system_type", string(c.systemType)),
		zap.String("backend", c.backend.Kind()),
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

	if err := c.validateConfig(); err != nil {
		return err
	}

	return c.backend.Start()
}

// Stop stops the sing-box service. Stopping an already-stopped service is not
// an error.
func (c *Controller) Stop() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	return c.backend.Stop()
}

// Restart validates the config and restarts the sing-box service, starting it
// if it was stopped.
func (c *Controller) Restart() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if err := c.validateConfig(); err != nil {
		return err
	}
	return c.backend.Restart()
}

// Reload validates the config and applies it without a full restart when the
// backend supports it (SIGHUP / ExecReload / procd reload), falling back to a
// restart otherwise.
func (c *Controller) Reload() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if err := c.validateConfig(); err != nil {
		return err
	}
	return c.backend.Reload()
}

// ForceStop force stops the sing-box service using SIGKILL.
func (c *Controller) ForceStop() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	return c.backend.ForceStop()
}

// TailLogs returns the most recent sing-box log lines from the backend's log
// source. See LogChunk for cursor semantics.
func (c *Controller) TailLogs(lines int, afterCursor string) (LogChunk, error) {
	return c.backend.TailLogs(clampLines(lines), afterCursor)
}

// logOutputPath returns the configured sing-box log.output file path, or ""
// when logging goes to stdout (no file to tail).
func (c *Controller) logOutputPath() string {
	cfg, err := c.configManager.GetConfig()
	if err != nil || cfg.Log == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Log.Output)
}
