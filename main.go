package main

import (
	"flag"
	"os"

	"github.com/SealinGp/sing-box-easy/app"
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

func main() {
	// Initialize logger first
	logger.InitDefault()
	defer logger.Sync()

	// Define command line flags
	configPath := flag.String("c", "app.yml", "Path to configuration file")
	flag.Parse()

	logger.Infof("Starting sing-box-easy with config: %s", *configPath)

	// Load configuration
	config, err := appconfig.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Override port from environment variable if set
	if port := os.Getenv("HTTP_PORT"); port != "" {
		logger.Infof("Overriding port from HTTP_PORT environment variable: %s", port)
		config.Server.Port = port
	}

	logger.Infof("Starting HTTP server on port %s", config.Server.Port)

	// Run application
	app.Run(config)
}
