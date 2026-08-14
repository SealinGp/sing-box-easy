package routes

import (
	"context"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	v1_12_12 "github.com/SealinGp/sing-box-easy/app/routes/v1_12_12"
	"github.com/SealinGp/sing-box-easy/app/webui"
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
		logger.L.Warn("Failed to resolve static root to an absolute path; using relative path",
			zap.String("path", p), zap.Error(err))
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
		r.config.GitHub,
		r.config.Server.Auth,
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

	// PWA/SPA fallback handler: serve static files if they exist, otherwise
	// index.html so client-side routing works.
	//
	// An on-disk ./dist (dev builds, tarball installs) takes precedence; the
	// frontend embedded into the binary (webui.FS) serves everything else —
	// single-binary installs such as the OpenWrt ipk ship no dist directory.
	if diskDistAvailable(staticRoot) {
		logger.L.Info("Serving frontend from on-disk dist", zap.String("path", staticRoot))
		r.hz.NoRoute(diskSPAHandler(staticRoot))
	} else {
		logger.L.Info("Serving frontend embedded in the binary")
		r.hz.NoRoute(embeddedSPAHandler(webui.FS()))
	}

	return nil
}

// diskDistAvailable reports whether a usable frontend build exists on disk.
func diskDistAvailable(root string) bool {
	info, err := os.Stat(filepath.Join(root, "index.html"))
	return err == nil && !info.IsDir()
}

// diskSPAHandler serves static files from the on-disk dist directory.
//
// Path traversal hardening: resolve the requested path against staticRoot and
// reject anything that escapes the root (e.g. "/../etc/passwd").
func diskSPAHandler(root string) app.HandlerFunc {
	indexPath := filepath.Join(root, "index.html")
	return func(ctx context.Context, c *app.RequestContext) {
		requestPath := string(c.Request.URI().Path())
		candidate := filepath.Join(root, requestPath)

		// Containment check: candidate must live under root.
		// filepath.Join already calls Clean, which normalises "../" segments.
		if candidate != root && !strings.HasPrefix(candidate, root+string(os.PathSeparator)) {
			c.File(indexPath)
			return
		}

		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			c.File(candidate)
			return
		}

		// For directories or non-existent files, serve index.html for client-side routing
		c.File(indexPath)
	}
}

// embeddedSPAHandler serves static files from the frontend bundle embedded in
// the binary. fs.FS paths are always slash-separated and rooted, so
// path.Clean plus the fs.ValidPath check inside ReadFile give the same
// traversal containment the disk handler enforces with filepath.
func embeddedSPAHandler(dist fs.FS) app.HandlerFunc {
	serve := func(c *app.RequestContext, name string) bool {
		data, err := fs.ReadFile(dist, name)
		if err != nil {
			return false
		}
		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Data(consts.StatusOK, contentType, data)
		return true
	}

	return func(ctx context.Context, c *app.RequestContext) {
		requestPath := path.Clean(strings.TrimPrefix(string(c.Request.URI().Path()), "/"))
		if requestPath != "." && requestPath != "" && fs.ValidPath(requestPath) {
			if serve(c, requestPath) {
				return
			}
		}
		// Directories, non-existent files and the root fall back to index.html
		// for client-side routing.
		if !serve(c, "index.html") {
			c.String(consts.StatusNotFound, "frontend bundle missing")
		}
	}
}

func (r *Route) Start() error {
	r.hz.Use(recovery.Recovery())

	if err := r.initEndpoints(); err != nil {
		return err
	}
	r.hz.Run()
	return nil
}
