package service

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/openwrtnet"
	"go.uber.org/zap"
)

// hostNetStateFile records what the OpenWrt network integration changed, so a
// later Stop can undo exactly that. It sits next to config.json — a persistent
// directory the service already owns — and is dot-prefixed so a sing-box
// running in directory mode never tries to merge it as config.
const hostNetStateFile = ".openwrt-net-state.json"

// Controller manages the sing-box service lifecycle. It owns config
// validation and delegates the actual lifecycle operations to the detected
// Backend (systemd, procd, or direct process management).
type Controller struct {
	configManager *config.Manager
	singBoxPath   string
	systemType    SystemType
	backend       Backend

	// hostNet applies the OpenWrt-side integration a TUN config needs — a
	// firewall zone for the tun device and a dnsmasq upstream pointing at
	// sing-box — and takes it back down on stop. A no-op everywhere else.
	hostNet *openwrtnet.Manager

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
	// State lives beside config.json so it survives a panel restart while
	// sing-box keeps running.
	controller.hostNet = openwrtnet.NewManager(
		systemType == SystemOpenWRT,
		filepath.Join(filepath.Dir(configManager.GetConfigPath()), hostNetStateFile),
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

	// Host integration goes up BEFORE sing-box.
	//
	// The tun device appears the moment sing-box starts, and until its
	// firewall zone exists fw4 answers everything coming back off it with a
	// TCP reset. Applying afterwards would leave a window — potentially
	// permanent, if the apply then failed — where the proxy is running and
	// every connection through it dies instantly.
	if err := c.applyHostNetwork(); err != nil {
		return err
	}

	if err := c.backend.Start(); err != nil {
		// Do not leave the host pointing at a proxy that failed to come up:
		// dnsmasq would be forwarding to a closed port, taking LAN DNS down
		// with it.
		if revertErr := c.hostNet.Revert(); revertErr != nil {
			logger.Error("failed to roll back the host network integration after a failed start",
				zap.Error(revertErr))
		}
		return err
	}
	return nil
}

// applyHostNetwork derives what the current config needs from the host and
// applies it. Failing to read the config here is not fatal: a config the
// backend can still start is better than refusing to start over an integration
// step, so the error is logged and the start proceeds.
func (c *Controller) applyHostNetwork() error {
	cfg, err := c.configManager.GetConfig()
	if err != nil {
		logger.Warn("could not read the config for host network integration; skipping",
			zap.Error(err))
		return nil
	}
	return c.hostNet.Apply(openwrtnet.DerivePlan(cfg))
}

// Stop stops the sing-box service. Stopping an already-stopped service is not
// an error.
func (c *Controller) Stop() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()

	if err := c.backend.Stop(); err != nil {
		return err
	}

	// Revert AFTER the process is down: undoing it first would blackhole
	// traffic that sing-box is still serving. A revert failure is reported but
	// does not make Stop fail — the service really did stop, and hiding that
	// would be worse than a leftover firewall zone.
	if err := c.hostNet.Revert(); err != nil {
		logger.Error("failed to revert the host network integration", zap.Error(err))
	}
	return nil
}

// Restart validates the config and restarts the sing-box service, starting it
// if it was stopped.
func (c *Controller) Restart() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if err := c.validateConfig(); err != nil {
		return err
	}
	// Re-apply first: a restart usually follows a config edit, which may have
	// renamed the tun device or moved the DNS port.
	if err := c.applyHostNetwork(); err != nil {
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

	if err := c.backend.ForceStop(); err != nil {
		return err
	}
	if err := c.hostNet.Revert(); err != nil {
		logger.Error("failed to revert the host network integration", zap.Error(err))
	}
	return nil
}

// TailLogs returns the most recent sing-box log lines from the backend's log
// source. See LogChunk for cursor semantics.
func (c *Controller) TailLogs(lines int, afterCursor string) (LogChunk, error) {
	return c.backend.TailLogs(clampLines(lines), afterCursor)
}

// FollowLogs pushes new sing-box log lines until ctx is cancelled.
//
// Returns (nil, nil) when this backend has no readable log source — the caller
// should fall back to polling TailLogs rather than reporting a failure, because
// there is nothing wrong: sing-box is simply logging to stdout with no init
// system capturing it.
//
// The caller MUST cancel ctx when it stops reading. The systemd and procd
// backends hold a child process open for the life of the follow.
func (c *Controller) FollowLogs(ctx context.Context) (<-chan FollowEvent, error) {
	return c.backend.FollowLogs(ctx)
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
