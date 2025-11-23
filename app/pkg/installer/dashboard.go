package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

const (
	DashboardURL = "https://gh-proxy.com/https://github.com/Zephyruso/zashboard/archive/refs/heads/gh-pages.zip"
)

// DashboardInitStateManager interface for updating initialization state
type DashboardInitStateManager interface {
	SetDashboardInstalled() error
}

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
	tasks            map[string]*DashboardTask
	mu               sync.RWMutex
	initStateManager DashboardInitStateManager
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager(initStateManager DashboardInitStateManager) *DashboardManager {
	return &DashboardManager{
		tasks:            make(map[string]*DashboardTask),
		initStateManager: initStateManager,
	}
}

// DownloadDashboard downloads and installs dashboard
func (m *DashboardManager) DownloadDashboard(targetDir, downloadURL, proxy string) (*DashboardTask, error) {
	taskID := fmt.Sprintf("download_%d", getCurrentTimestamp())

	// Use default URL if not specified
	if downloadURL == "" {
		downloadURL = DashboardURL
	}

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
		extractedFolder, err := m.downloadAndExtract(task, targetDir, downloadURL, proxy)

		task.mu.Lock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.Message = "Download failed"
		} else {
			task.Status = "completed"
			// Return meaningful message with extracted folder info
			if extractedFolder != "" {
				task.Message = fmt.Sprintf("Dashboard extracted to: %s", filepath.Join(targetDir, extractedFolder))
			} else {
				task.Message = fmt.Sprintf("Dashboard installed to: %s", targetDir)
			}

			// Update initialization state
			if m.initStateManager != nil {
				if err := m.initStateManager.SetDashboardInstalled(); err != nil {
					logger.Error("Failed to update initialization state", zap.Error(err))
					// Don't fail the task, just log the error
				} else {
					logger.Info("Dashboard installation state updated")
				}
			}
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
// Returns the name of the extracted folder
func (m *DashboardManager) downloadAndExtract(task *DashboardTask, targetDir, downloadURL, proxy string) (string, error) {
	// Update progress: Creating temp file
	task.mu.Lock()
	task.Message = "Creating temporary file..."
	task.mu.Unlock()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "dashboard-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Update progress: Starting download
	task.mu.Lock()
	task.Message = "Connecting to dashboard server..."
	task.mu.Unlock()

	// Create HTTP client with optional proxy
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return "", fmt.Errorf("invalid proxy URL: %w", err)
		}

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		client.Transport = transport

		logger.Info("Using proxy for dashboard download", zap.String("proxy", proxy))
	}

	// Download file
	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Update progress: Downloading
	task.mu.Lock()
	if resp.ContentLength > 0 {
		task.Message = fmt.Sprintf("Downloading dashboard... (%.2f MB)", float64(resp.ContentLength)/(1024*1024))
	} else {
		task.Message = "Downloading dashboard..."
	}
	task.mu.Unlock()

	// Write to temp file with progress tracking
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write dashboard file: %w", err)
	}

	// Update progress: Download complete
	task.mu.Lock()
	task.Message = fmt.Sprintf("Download complete (%.2f MB). Extracting...", float64(written)/(1024*1024))
	task.mu.Unlock()

	// Extract zip and get the extracted folder name
	extractedFolder, err := extractZipAndGetFolder(tmpFile.Name(), targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to extract dashboard: %w", err)
	}

	// Update progress: Extraction complete
	task.mu.Lock()
	if extractedFolder != "" {
		task.Message = fmt.Sprintf("Dashboard extracted to folder: %s", extractedFolder)
	} else {
		task.Message = "Dashboard extracted successfully"
	}
	task.mu.Unlock()

	return extractedFolder, nil
}

// UploadDashboard uploads and extracts a dashboard zip file
func (m *DashboardManager) UploadDashboard(zipPath, targetDir, folderName string) (*DashboardTask, error) {
	taskID := fmt.Sprintf("upload_%d", getCurrentTimestamp())

	task := &DashboardTask{
		ID:      taskID,
		Status:  "running",
		Message: "Uploading dashboard...",
	}

	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Run upload in background
	go func() {
		extractedFolder, err := m.extractAndRename(task, zipPath, targetDir, folderName)

		task.mu.Lock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			task.Message = "Upload failed"
		} else {
			task.Status = "completed"
			// Return meaningful message with extracted folder info
			if extractedFolder != "" {
				task.Message = fmt.Sprintf("Dashboard uploaded and extracted to: %s", filepath.Join(targetDir, extractedFolder))
			} else {
				task.Message = fmt.Sprintf("Dashboard uploaded and installed to: %s", targetDir)
			}

			// Update initialization state
			if m.initStateManager != nil {
				if err := m.initStateManager.SetDashboardInstalled(); err != nil {
					logger.Error("Failed to update initialization state", zap.Error(err))
					// Don't fail the task, just log the error
				} else {
					logger.Info("Dashboard installation state updated")
				}
			}
		}
		task.mu.Unlock()

		// Clean up uploaded file
		os.Remove(zipPath)
	}()

	return task, nil
}

// extractAndRename extracts zip and optionally renames the extracted folder
// Returns the name of the extracted folder
func (m *DashboardManager) extractAndRename(task *DashboardTask, zipPath, targetDir, folderName string) (string, error) {
	// Update progress: Starting extraction
	task.mu.Lock()
	task.Message = "Extracting uploaded dashboard..."
	task.mu.Unlock()

	// Extract zip directly to target directory, preserving folder structure
	extractedFolder, err := extractZipAndGetFolder(zipPath, targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to extract dashboard: %w", err)
	}

	// If folderName is specified and different from extracted folder, rename it
	if folderName != "" && extractedFolder != "" && folderName != extractedFolder {
		oldPath := filepath.Join(targetDir, extractedFolder)
		newPath := filepath.Join(targetDir, folderName)

		// Update progress: Renaming folder
		task.mu.Lock()
		task.Message = fmt.Sprintf("Renaming folder from %s to %s...", extractedFolder, folderName)
		task.mu.Unlock()

		logger.Info("Renaming extracted folder",
			zap.String("from", extractedFolder),
			zap.String("to", folderName),
			zap.String("oldPath", oldPath),
			zap.String("newPath", newPath),
		)

		// Rename the directory
		err = os.Rename(oldPath, newPath)
		if err != nil {
			return "", fmt.Errorf("failed to rename folder: %w", err)
		}

		extractedFolder = folderName
	}

	// Update progress: Complete
	task.mu.Lock()
	if extractedFolder != "" {
		task.Message = fmt.Sprintf("Dashboard extracted to folder: %s", extractedFolder)
	} else {
		task.Message = "Dashboard uploaded and extracted successfully"
	}
	task.mu.Unlock()

	return extractedFolder, nil
}

// GetDashboardStatus checks if dashboard is installed
func (m *DashboardManager) GetDashboardStatus(targetDir string) (bool, error) {
	// Check if dashboard directory exists and has files
	if _, err := os.Stat(filepath.Join(targetDir, "index.html")); err == nil {
		return true, nil
	}

	return false, nil
}

// extractZipAndGetFolder extracts a zip file to target directory and returns the extracted folder name
func extractZipAndGetFolder(zipPath, targetDir string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	// Track the root folder name (first directory in the zip)
	var rootFolder string

	for _, file := range reader.File {
		// Preserve the full path from the zip file
		targetPath := filepath.Join(targetDir, file.Name)

		// Security check: ensure the file path doesn't escape the target directory
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", file.Name)
		}

		// Get the root folder name (first component of the path)
		if rootFolder == "" && file.Name != "" {
			parts := strings.Split(file.Name, "/")
			if len(parts) > 0 && parts[0] != "" {
				rootFolder = parts[0]
			}
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, file.Mode())
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", err
		}

		// Create and write file
		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return "", err
		}
	}

	return rootFolder, nil
}

// extractZip extracts a zip file to target directory, preserving the folder structure
func extractZip(zipPath, targetDir string) error {
	_, err := extractZipAndGetFolder(zipPath, targetDir)
	return err
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	// Get properties of source directory
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
