package appconfig

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig  `yaml:"server"`
	SingBox   SingBoxConfig `yaml:"sing_box"`
	Log       LogConfig     `yaml:"log"`
	GitHub    GitHubConfig  `yaml:"github"`
	AdminUser string        `yaml:"-"`
	AdminPass string        `yaml:"-"`
}

// GitHubConfig configures GitHub sign-in, which lifts the API rate limit on
// update checks from 60 to 5000 requests per hour.
type GitHubConfig struct {
	// OAuthClientID is the client ID of a GitHub OAuth App with "Enable
	// Device Flow" ticked. Device flow uses no client secret, so this value
	// is public by design and safe to ship in a config file or a binary.
	//
	// When empty, sign-in is unavailable and update checks fall back to
	// anonymous (rate-limited) access.
	OAuthClientID string `yaml:"oauth_client_id"`

	// Proxy optionally routes GitHub traffic through a proxy,
	// e.g. "http://127.0.0.1:7890".
	Proxy string `yaml:"proxy"`
}

// EnvGitHubOAuthClientID overrides github.oauth_client_id from the environment.
const EnvGitHubOAuthClientID = "GITHUB_OAUTH_CLIENT_ID"

// DefaultGitHubOAuthClientID is the client ID shipped with official builds. It
// is empty until an OAuth App is registered for the project; sign-in is simply
// unavailable (and the UI says so) while it is blank.
//
// This is NOT a secret: the device flow authenticates the user, not the client,
// so a public client ID is exactly what the spec expects.
const DefaultGitHubOAuthClientID = ""

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

// LogConfig represents logger configuration.
// Level accepts: debug, info (default), warn/warning, error.
type LogConfig struct {
	Level string `yaml:"level"`
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

	if config.Log.Level == "" {
		config.Log.Level = "info"
	}

	// The OAuth client ID is public (device flow has no secret), but allowing
	// an env override keeps container deployments from having to template the
	// YAML file.
	if v := strings.TrimSpace(os.Getenv(EnvGitHubOAuthClientID)); v != "" {
		config.GitHub.OAuthClientID = v
	}
	if config.GitHub.OAuthClientID == "" {
		config.GitHub.OAuthClientID = DefaultGitHubOAuthClientID
	}

	return &config, nil
}
