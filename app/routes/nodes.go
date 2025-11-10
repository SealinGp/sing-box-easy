package routes

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	v1_13_0 "github.com/SealinGp/sing-box-easy/app/routes/v1_13_0"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type Route struct {
	hz     *server.Hertz
	sl     *sublink.SubLink
	config *appconfig.Config
}

func NewRoute(hp string, config *appconfig.Config) *Route {
	return &Route{
		hz: server.New(
			server.WithHostPorts(hp),
		),
		sl:     new(sublink.SubLink),
		config: config,
	}
}

func (r *Route) initEndpoints() {
	r.hz.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(consts.StatusOK, utils.H{"message": "ok"})
	})
	r.hz.SetCustomSignalWaiter(func(err chan error) error {
		return nil
	})

	// Legacy v1 API
	r.hz.POST("/v1/parse-nodes", r.ListNodes)

	// Register v1.13.0 API routes with configuration
	v1Handler := v1_13_0.NewHandler(
		r.config.SingBox.ConfigPath,
		r.config.SingBox.BinaryPath,
		r.config.SingBox.SubscriptionPath,
		r.config.SingBox.InitStatePath,
	)
	v1_13_0.RegisterRoutes(r.hz, v1Handler)

	// Serve frontend static files (should be registered last)
	r.hz.StaticFS("/", &app.FS{Root: "./frontend/dist", PathRewrite: app.NewPathSlashesStripper(0)})
}

func (r *Route) Start() {
	r.initEndpoints()
	r.hz.Run()
}
