package routes

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	v1_12_12 "github.com/SealinGp/sing-box-easy/app/routes/v1_12_12"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"
)

// staticRoot is the resolved absolute path to the frontend dist directory.
// It is computed once at startup and used as the containment boundary for
// every static-file request, preventing path traversal via "../" segments.
var staticRoot = mustResolve("./dist")

func mustResolve(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		// Fall back to the cleaned relative path; static serving still works,
		// but containment checks are slightly weaker.
		return filepath.Clean(p)
	}
	return abs
}

type Route struct {
	hz     *server.Hertz
	sl     *sublink.SubLink
	config *appconfig.Config
}

func NewRoute(hp string, config *appconfig.Config) *Route {
	// Configure Hertz to use zap logger
	hlog.SetLogger(logger.NewHertzLogger(logger.Logger))

	return &Route{
		hz: server.New(
			server.WithHostPorts(hp),
			// Increase max request body size to 100MB for dashboard uploads
			server.WithMaxRequestBodySize(100*1024*1024), // 100MB
		),
		sl:     new(sublink.SubLink),
		config: config,
	}
}

func (r *Route) initEndpoints() error {
	r.hz.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"message": "ok"})
	})
	r.hz.SetCustomSignalWaiter(func(err chan error) error {
		return nil
	})

	// Register v1.12.12 API routes with configuration
	v1Handler := v1_12_12.NewHandler(
		r.config.SingBox.ConfigPath,
		r.config.SingBox.BinaryPath,
		r.config.AdminUser,
		r.config.AdminPass,
		r.sl,
	)

	// Initialize handler components
	if err := v1Handler.Init(); err != nil {
		return err
	}

	// Start the auto-updater with default cron expression
	if err := v1Handler.StartAutoUpdater("*/5 * * * *"); err != nil {
		// Log error but don't fail initialization
		logger.L.Error("Failed to start auto-updater", zap.Error(err))
	} else {
		logger.L.Info("Auto-updater started successfully with 5-minute interval")
	}

	// Start the config-version retention sweep (deletes versions older than 60 days).
	if err := v1Handler.StartVersionCleaner(); err != nil {
		logger.L.Error("Failed to start config version cleaner", zap.Error(err))
	} else {
		logger.L.Info("Config version cleaner started (daily, 60-day retention)")
	}

	v1_12_12.RegisterRoutes(r.hz, v1Handler)

	// PWA/SPA fallback handler: serve static files if they exist, otherwise index.html
	// This allows client-side routing to work properly.
	//
	// Path traversal hardening: resolve the requested path against staticRoot
	// and reject anything that escapes the root (e.g. "/../etc/passwd").
	indexPath := filepath.Join(staticRoot, "index.html")
	r.hz.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		requestPath := string(c.Request.URI().Path())
		candidate := filepath.Join(staticRoot, requestPath)

		// Containment check: candidate must live under staticRoot.
		// filepath.Join already calls Clean, which normalises "../" segments.
		if candidate != staticRoot && !strings.HasPrefix(candidate, staticRoot+string(os.PathSeparator)) {
			c.File(indexPath)
			return
		}

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			c.File(candidate)
			return
		}

		// For directories or non-existent files, serve index.html for client-side routing
		c.File(indexPath)
	})

	return nil
}

func (r *Route) Start() error {
	r.hz.Use(recovery.Recovery())

	if err := r.initEndpoints(); err != nil {
		return err
	}
	r.hz.Run()
	return nil
}
