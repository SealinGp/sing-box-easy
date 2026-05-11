package installer

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// versionPattern restricts the user-supplied version string to a strict
// semver-ish shape before it is interpolated into a shell command.
// Allows: 1.2.3, 1.12.12, 1.2.3-rc1, 1.2.3-beta.1
var versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// InstallTask represents an installation task
type InstallTask struct {
	ID      string
	Status  string // running, completed, failed
	Message string
	Error   string
	mu      sync.RWMutex
}

// Manager manages installation tasks
type Manager struct {
	tasks            map[string]*InstallTask
	mu               sync.RWMutex
	initStateManager initstate.InitStateManager
	configManager    *config.Manager
}

// NewManager creates a new installer manager
func NewManager(initStateManager initstate.InitStateManager, configManager *config.Manager) *Manager {
	return &Manager{
		tasks:            make(map[string]*InstallTask),
		initStateManager: initStateManager,
		configManager:    configManager,
	}
}

func (m *Manager) Init() error {
	installed, ver, _ := m.GetInstallStatus()
	if installed {
		err := m.initStateManager.SetSingBoxInstalled(ver)
		logger.Info("Installer manager initialized",
			zap.Bool("installed", installed),
			zap.String("version", ver),
			zap.Error(err),
		)
	}

	return nil
}

// InstallSingBox installs sing-box.
// version, when non-empty, must match versionPattern — it is interpolated
// into a shell command run as root, so any non-matching input is rejected.
func (m *Manager) InstallSingBox(version string, beta bool) (*InstallTask, error) {
	if version != "" && !versionPattern.MatchString(version) {
		return nil, fmt.Errorf("invalid version format: %q (expected semver like 1.12.12)", version)
	}

	taskID := fmt.Sprintf("install_%d", getCurrentTimestamp())

	task := &InstallTask{
		ID:      taskID,
		Status:  "running",
		Message: "Preparing to install sing-box...",
	}

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Run installation in background
	go func() {
		err := m.runInstallScript(task, version, beta)

		task.mu.Lock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.Message = "Installation failed"
		} else {
			task.Status = "completed"
			task.Message = "Installation completed successfully"

			// Update initialization state
			if m.initStateManager != nil {
				installedVersion := version
				if installedVersion == "" {
					// Get actual installed version
					_, installedVersion, _ = m.GetInstallStatus()
				}
				if err := m.initStateManager.SetSingBoxInstalled(installedVersion); err != nil {
					logger.Error("Failed to update initialization state",
						zap.Error(err),
						zap.String("version", installedVersion),
					)
					// Don't fail the task, just log the error
				} else {
					logger.Info("Initialization state updated",
						zap.String("version", installedVersion),
					)

					// Initialize config file after successful installation
					if m.configManager != nil {
						if err := m.configManager.InitializeConfig(); err != nil {
							logger.Error("Failed to initialize config file",
								zap.Error(err),
							)
							// Don't fail the task, just log the error
						} else {
							logger.Info("Config file initialized successfully")
						}
					}
				}
			}
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

// runInstallScript runs the sing-box installation script with real-time output
func (m *Manager) runInstallScript(task *InstallTask, version string, beta bool) error {
	// Build install command with more robust curl options
	var cmdStr string
	curlOpts := "curl --retry 3 --retry-delay 2 --connect-timeout 30 --max-time 300 -fsSL"

	if beta {
		cmdStr = fmt.Sprintf("%s https://sing-box.app/install.sh | sh -s -- --beta", curlOpts)
	} else if version != "" {
		cmdStr = fmt.Sprintf("%s https://sing-box.app/install.sh | sh -s -- --version %s", curlOpts, version)
	} else {
		cmdStr = fmt.Sprintf("%s https://sing-box.app/install.sh | sh", curlOpts)
	}

	// Log command
	logger.Info("Starting sing-box installation",
		zap.String("command", cmdStr),
		zap.String("version", version),
		zap.Bool("beta", beta),
	)

	cmd := exec.Command("sh", "-c", cmdStr)

	// Inherit environment variables from parent process
	// Set working directory to avoid permission issues
	cmd.Dir = "/tmp"
	cmd.Env = append(os.Environ(),
		"DEBIAN_FRONTEND=noninteractive",                    // Avoid interactive prompts
		"CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt", // Ensure CA bundle is used
	)

	logger.Debug("Installation environment",
		zap.String("workdir", cmd.Dir),
		zap.String("path", os.Getenv("PATH")),
		zap.String("user", os.Getenv("USER")),
	)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	logger.Info("Starting installation process")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start installation: %w", err)
	}
	logger.Info("Installation process started", zap.Int("pid", cmd.Process.Pid))

	// Update task message with real-time output
	outputChan := make(chan string, 100)
	done := make(chan error, 1)

	// Read stdout
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				outputChan <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Read stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				outputChan <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for command to complete
	go func() {
		done <- cmd.Wait()
	}()

	// Collect last few lines for task message
	lastLines := make([]string, 0, 5)

	// Process output in real-time
	for {
		select {
		case line := <-outputChan:
			// Keep last 3-5 lines
			lines := strings.Split(strings.TrimSpace(line), "\n")
			for _, l := range lines {
				if l != "" {
					lastLines = append(lastLines, l)
					if len(lastLines) > 5 {
						lastLines = lastLines[1:]
					}
				}
			}

			// Update task message
			task.mu.Lock()
			if len(lastLines) > 0 {
				task.Message = strings.Join(lastLines, "\n")
			}
			task.mu.Unlock()

		case err := <-done:
			// Command finished
			if err != nil {
				task.mu.Lock()
				finalMsg := task.Message
				task.mu.Unlock()
				logger.Error("Installation failed",
					zap.Error(err),
					zap.String("last_output", finalMsg),
				)
				return fmt.Errorf("installation failed: %w\nLast output: %s", err, finalMsg)
			}
			logger.Info("Installation completed successfully")
			return nil
		}
	}
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
