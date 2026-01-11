package routes

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	v1_12_12 "github.com/SealinGp/sing-box-easy/app/routes/v1_12_12"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.uber.org/zap"
)

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

	v1_12_12.RegisterRoutes(r.hz, v1Handler)

	// Serve frontend static files (should be registered last)
	r.hz.StaticFS("/", &app.FS{
		Root:               "./dist",
		PathRewrite:        app.NewPathSlashesStripper(0),
		GenerateIndexPages: true,
		IndexNames:         []string{"index.html"},
	})

	return nil
}

func (r *Route) Start() error {
	if err := r.initEndpoints(); err != nil {
		return err
	}
	r.hz.Run()
	return nil
}
