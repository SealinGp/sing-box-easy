package main

import (
	"flag"
	"os"

	"github.com/SealinGp/sing-box-easy/app"
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

func main() {
	// Bootstrap logger so we can report config-load errors before
	// the configured level has been applied.
	logger.InitDefault()
	defer logger.Sync()

	// Define command line flags
	configPath := flag.String("c", "app.yml", "Path to configuration file")
	adminUser := flag.String("admin_user", "", "Default admin username to seed")
	adminPass := flag.String("admin_pass", "", "Default admin password to seed")
	flag.Parse()

	logger.Infof("Starting sing-box-easy with config: %s", *configPath)

	// Load configuration
	config, err := appconfig.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	config.AdminUser = *adminUser
	config.AdminPass = *adminPass

	// Re-initialize logger with the configured level now that we have it.
	// DEBUG=true env var still wins so devs can crank up verbosity without
	// editing the config file.
	logLevel := config.Log.Level
	if os.Getenv("DEBUG") == "true" {
		logLevel = "debug"
	}
	if err := logger.Init(logLevel); err != nil {
		logger.Fatalf("Failed to initialize logger with level %q: %v", logLevel, err)
	}
	logger.Infof("Logger initialized at level: %s", logLevel)

	// Override port from environment variable if set
	if port := os.Getenv("HTTP_PORT"); port != "" {
		logger.Infof("Overriding port from HTTP_PORT environment variable: %s", port)
		config.Server.Port = port
	}

	logger.Infof("Starting HTTP server on port %s", config.Server.Port)

	// Run application
	if err := app.Run(config); err != nil {
		logger.Fatalf("Failed to start application: %v", err)
	}
}
