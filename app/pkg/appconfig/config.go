package appconfig

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	SingBox SingBoxConfig `yaml:"sing_box"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port string `yaml:"port"`
}

// SingBoxConfig represents sing-box related configuration
type SingBoxConfig struct {
	ConfigPath   string `yaml:"config_path"`
	BinaryPath   string `yaml:"binary_path"`
	DatabasePath string `yaml:"database_path"`
}

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path is required")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values if not specified
	if config.Server.Port == "" {
		config.Server.Port = "8080"
	}

	if config.SingBox.ConfigPath == "" {
		config.SingBox.ConfigPath = "/etc/sing-box/config.json"
	}

	if config.SingBox.BinaryPath == "" {
		config.SingBox.BinaryPath = "sing-box"
	}

	if config.SingBox.DatabasePath == "" {
		config.SingBox.DatabasePath = "/etc/sing-box/sing-box-easy.db"
	}

	return &config, nil
}
