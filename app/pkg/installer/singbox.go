package installer

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// InstallTask represents an installation task
type InstallTask struct {
	ID        string
	Status    string // running, completed, failed
	Message   string
	Error     string
	mu        sync.RWMutex
}

// Manager manages installation tasks
type Manager struct {
	tasks map[string]*InstallTask
	mu    sync.RWMutex
}

// NewManager creates a new installer manager
func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*InstallTask),
	}
}

// InstallSingBox installs sing-box
func (m *Manager) InstallSingBox(version string, beta bool) (*InstallTask, error) {
	taskID := fmt.Sprintf("install_%d", getCurrentTimestamp())

	task := &InstallTask{
		ID:      taskID,
		Status:  "running",
		Message: "Installing sing-box...",
	}

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Run installation in background
	go func() {
		err := m.runInstallScript(version, beta)

		task.mu.Lock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.Message = "Installation failed"
		} else {
			task.Status = "completed"
			task.Message = "Installation completed successfully"
		}
		task.mu.Unlock()
	}()

	return task, nil
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*InstallTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

// runInstallScript runs the sing-box installation script
func (m *Manager) runInstallScript(version string, beta bool) error {
	// Build install command
	var cmd *exec.Cmd

	if beta {
		// Install beta version
		cmd = exec.Command("sh", "-c", "curl -fsSL https://sing-box.app/install.sh | sh -s -- --beta")
	} else if version != "" {
		// Install specific version
		cmd = exec.Command("sh", "-c", fmt.Sprintf("curl -fsSL https://sing-box.app/install.sh | sh -s -- --version %s", version))
	} else {
		// Install latest version
		cmd = exec.Command("sh", "-c", "curl -fsSL https://sing-box.app/install.sh | sh")
	}

	// Run command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %s, output: %s", err.Error(), string(output))
	}

	return nil
}

// GetInstallStatus checks if sing-box is installed and returns version
func (m *Manager) GetInstallStatus() (bool, string, error) {
	// Check if sing-box is installed
	cmd := exec.Command("sing-box", "version")
	output, err := cmd.Output()
	if err != nil {
		return false, "", nil
	}

	// Parse version from output
	version := parseVersion(string(output))
	return true, version, nil
}

// UpdateSingBox updates sing-box
func (m *Manager) UpdateSingBox(version string, beta bool) (*InstallTask, error) {
	// Update is same as install
	return m.InstallSingBox(version, beta)
}

// parseVersion parses version from sing-box version output
func parseVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "sing-box version") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return "unknown"
}

// getCurrentTimestamp returns current unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
