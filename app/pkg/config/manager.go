package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	DefaultConfigPath    = "/etc/sing-box/config.json"
	DefaultBackupPath    = "/etc/sing-box/config.old.json"
	DefaultNewConfigPath = "/etc/sing-box/config_new.json"
)

// Manager handles sing-box configuration management
type Manager struct {
	configPath    string
	backupPath    string
	newConfigPath string
	singBoxPath   string // Path to sing-box binary
}

// NewManager creates a new configuration manager
func NewManager(configPath, singBoxPath string) *Manager {
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	if singBoxPath == "" {
		singBoxPath = "sing-box" // Use PATH
	}

	dir := filepath.Dir(configPath)
	return &Manager{
		configPath:    configPath,
		backupPath:    filepath.Join(dir, "config.old.json"),
		newConfigPath: filepath.Join(dir, "config_new.json"),
		singBoxPath:   singBoxPath,
	}
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetConfig reads and returns the current configuration
func (m *Manager) GetConfig() (*SingBoxConfig, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SingBoxConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetBackupConfig reads and returns the backup configuration
func (m *Manager) GetBackupConfig() (*SingBoxConfig, error) {
	data, err := os.ReadFile(m.backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup config file: %w", err)
	}

	var config SingBoxConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse backup config file: %w", err)
	}

	return &config, nil
}

// ValidateConfig validates the configuration using sing-box binary
func (m *Manager) ValidateConfig(config *SingBoxConfig) error {
	// Save to temporary file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.newConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	// Validate using sing-box check command
	cmd := exec.Command(m.singBoxPath, "check", "-c", m.newConfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up temp file on validation error
		os.Remove(m.newConfigPath)
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}

// SaveConfig saves the configuration after validation
// It creates a backup of the current config before saving
func (m *Manager) SaveConfig(config *SingBoxConfig) error {
	// Validate first
	if err := m.ValidateConfig(config); err != nil {
		return err
	}

	// Backup current config if it exists
	if _, err := os.Stat(m.configPath); err == nil {
		if err := m.copyFile(m.configPath, m.backupPath); err != nil {
			os.Remove(m.newConfigPath)
			return fmt.Errorf("failed to backup config: %w", err)
		}
	}

	// Move new config to main config
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Rollback restores the backup configuration
func (m *Manager) Rollback() error {
	if _, err := os.Stat(m.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup config found")
	}

	// Read backup config first to validate it exists
	backupConfig, err := m.GetBackupConfig()
	if err != nil {
		return fmt.Errorf("failed to read backup config: %w", err)
	}

	// Validate backup config
	if err := m.ValidateConfig(backupConfig); err != nil {
		os.Remove(m.newConfigPath)
		return fmt.Errorf("backup config validation failed: %w", err)
	}

	// Move validated backup to main config
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		return fmt.Errorf("failed to rollback config: %w", err)
	}

	return nil
}

// UpdateConfig updates the configuration with a function
// This is useful for partial updates
func (m *Manager) UpdateConfig(updateFn func(*SingBoxConfig) error) error {
	config, err := m.GetConfig()
	if err != nil {
		return err
	}

	if err := updateFn(config); err != nil {
		return err
	}

	return m.SaveConfig(config)
}

// copyFile copies a file from src to dst
func (m *Manager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
