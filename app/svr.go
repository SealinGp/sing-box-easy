package app

import (
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

	hp := ":" + config.Server.Port
	route := routes.NewRoute(hp, config)
	return route.Start()
}
