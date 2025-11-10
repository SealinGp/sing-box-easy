package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SealinGp/sing-box-easy/app"
	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
)

func main() {
	// Define command line flags
	configPath := flag.String("c", "app.yml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := appconfig.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override port from environment variable if set
	if port := os.Getenv("HTTP_PORT"); port != "" {
		config.Server.Port = port
	}

	// Run application
	app.Run(config)
}
