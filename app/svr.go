package app

import (
	"github.com/SealinGp/sing-box-easy/app/bootstrap"
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/routes"
	"go.uber.org/zap"
)

func Run(config *appconfig.Config) error {
	// Initialize database with XORM
	if err := database.Init(config.SingBox.DatabasePath); err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		return err
	}
	defer database.Close()

	logger.Info("Database initialized with XORM", zap.String("path", config.SingBox.DatabasePath))

	modules, err := bootstrap.New(config.SingBox.ConfigPath, config.SingBox.BinaryPath, config.AdminUser, config.AdminPass, config.GitHub, config.Server.Auth)
	if err != nil {
		return err
	}
	defer modules.Close()
	if err := modules.Init(); err != nil {
		return err
	}
	if err := modules.Start(); err != nil {
		return err
	}
	hp := ":" + config.Server.Port
	route := routes.NewRoute(hp, config, modules)
	return route.Start()
}
