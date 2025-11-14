package initstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

const (
	DefaultStatePath = "/etc/sing-box/init_state.json"
)

// State represents the initialization state
type State struct {
	Initialized        bool   `json:"initialized"`
	SingBoxInstalled   bool   `json:"sing_box_installed"`
	ConfigGenerated    bool   `json:"config_generated"`
	DashboardInstalled bool   `json:"dashboard_installed"`
	SingBoxVersion     string `json:"sing_box_version"`
	InitTime           string `json:"init_time,omitempty"`
	mu                 sync.RWMutex
	statePath          string
}

// Manager manages initialization state
type Manager struct {
	state *State
}

// NewManager creates a new initialization state manager
func NewManager(statePath string) *Manager {
	if statePath == "" {
		statePath = DefaultStatePath
	}

	state := &State{
		statePath: statePath,
	}

	// Load existing state
	if err := state.load(); err != nil {
		logger.Warn("Failed to load state file", zap.Error(err))
	}

	// If state file doesn't exist, create it with default values
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		// Ensure directory exists
		dir := filepath.Dir(statePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Warn("Failed to create state directory", zap.Error(err), zap.String("dir", dir))
		} else {
			// Create initial state file
			if err := state.save(); err != nil {
				logger.Warn("Failed to create initial state file", zap.Error(err), zap.String("path", statePath))
			} else {
				logger.Info("Created initial state file", zap.String("path", statePath))
			}
		}
	}

	return &Manager{
		state: state,
	}
}

// GetState returns the current state
func (m *Manager) GetState() *State {
	m.state.mu.RLock()
	defer m.state.mu.RUnlock()

	// Return a copy
	return &State{
		Initialized:        m.state.Initialized,
		SingBoxInstalled:   m.state.SingBoxInstalled,
		ConfigGenerated:    m.state.ConfigGenerated,
		DashboardInstalled: m.state.DashboardInstalled,
		SingBoxVersion:     m.state.SingBoxVersion,
		InitTime:           m.state.InitTime,
	}
}

// SetSingBoxInstalled marks sing-box as installed
func (m *Manager) SetSingBoxInstalled(version string) error {
	m.state.mu.Lock()
	m.state.SingBoxInstalled = true
	m.state.SingBoxVersion = version
	m.state.mu.Unlock()

	return m.state.save()
}

// SetDashboardInstalled marks dashboard as installed
func (m *Manager) SetDashboardInstalled() error {
	m.state.mu.Lock()
	m.state.DashboardInstalled = true
	m.state.mu.Unlock()

	return m.state.save()
}

// CompleteInitialization marks initialization as complete
// This also sets ConfigGenerated to true as they always happen together
func (m *Manager) CompleteInitialization() error {
	m.state.mu.Lock()
	m.state.Initialized = true
	m.state.ConfigGenerated = true // Always set when completing initialization
	m.state.InitTime = getCurrentTimeISO()
	m.state.mu.Unlock()

	return m.state.save()
}

// Reset resets the initialization state
func (m *Manager) Reset() error {
	m.state.mu.Lock()
	m.state.Initialized = false
	m.state.SingBoxInstalled = false
	m.state.ConfigGenerated = false
	m.state.DashboardInstalled = false
	m.state.SingBoxVersion = ""
	m.state.InitTime = ""
	m.state.mu.Unlock()

	return m.state.save()
}

// load loads state from file
func (s *State) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(s.statePath); os.IsNotExist(err) {
		// File doesn't exist, use default state
		return nil
	}

	data, err := os.ReadFile(s.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	return nil
}

// save saves state to file
func (s *State) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// getCurrentTimeISO returns current time in ISO format
func getCurrentTimeISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
