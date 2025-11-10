package app

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/routes"
)

func Run(config *appconfig.Config) {
	hp := ":" + config.Server.Port
	route := routes.NewRoute(hp, config)
	route.Start()
}
