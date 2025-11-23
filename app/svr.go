package app

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/routes"
)

func Run(config *appconfig.Config) error {
	hp := ":" + config.Server.Port
	route := routes.NewRoute(hp, config)
	return route.Start()
}
