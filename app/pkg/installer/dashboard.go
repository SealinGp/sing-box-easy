package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	DashboardURL = "https://gh-proxy.com/https://github.com/Zephyruso/zashboard/archive/refs/heads/gh-pages.zip"
)

// DashboardTask represents a dashboard download task
type DashboardTask struct {
	ID        string
	Status    string // running, completed, failed
	Message   string
	Error     string
	mu        sync.RWMutex
}

// DashboardManager manages dashboard download tasks
type DashboardManager struct {
	tasks map[string]*DashboardTask
	mu    sync.RWMutex
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager() *DashboardManager {
	return &DashboardManager{
		tasks: make(map[string]*DashboardTask),
	}
}

// DownloadDashboard downloads and installs dashboard
func (m *DashboardManager) DownloadDashboard(targetDir string) (*DashboardTask, error) {
	taskID := fmt.Sprintf("download_%d", getCurrentTimestamp())

	task := &DashboardTask{
		ID:      taskID,
		Status:  "running",
		Message: "Downloading dashboard...",
	}

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Run download in background
	go func() {
		err := m.downloadAndExtract(targetDir)

		task.mu.Lock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.Message = "Download failed"
		} else {
			task.Status = "completed"
			task.Message = "Download completed successfully"
		}
		task.mu.Unlock()
	}()

	return task, nil
}

// GetTask returns a task by ID
func (m *DashboardManager) GetTask(taskID string) (*DashboardTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

// downloadAndExtract downloads dashboard zip and extracts it
func (m *DashboardManager) downloadAndExtract(targetDir string) error {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "dashboard-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download file
	resp, err := http.Get(DashboardURL)
	if err != nil {
		return fmt.Errorf("failed to download dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Write to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write dashboard file: %w", err)
	}

	// Extract zip
	err = extractZip(tmpFile.Name(), targetDir)
	if err != nil {
		return fmt.Errorf("failed to extract dashboard: %w", err)
	}

	return nil
}

// GetDashboardStatus checks if dashboard is installed
func (m *DashboardManager) GetDashboardStatus(targetDir string) (bool, error) {
	// Check if dashboard directory exists and has files
	if _, err := os.Stat(filepath.Join(targetDir, "index.html")); err == nil {
		return true, nil
	}

	return false, nil
}

// extractZip extracts a zip file to target directory
func extractZip(zipPath, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		// Skip the root directory (zashboard-gh-pages/)
		fileName := file.Name
		if idx := strings.Index(fileName, "/"); idx != -1 {
			fileName = fileName[idx+1:]
		}

		if fileName == "" {
			continue
		}

		targetPath := filepath.Join(targetDir, fileName)

		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, file.Mode())
			continue
		}

		// Create file
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
