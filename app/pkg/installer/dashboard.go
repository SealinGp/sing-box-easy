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

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/initstate"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

const (
	DashboardURL = "https://gh-proxy.com/https://github.com/Zephyruso/zashboard/archive/refs/heads/gh-pages.zip"
)

// DashboardInitStateManager interface for updating initialization state

// DashboardTask represents a dashboard download task
type DashboardTask struct {
	ID      string
	Status  string // running, completed, failed
	Message string
	Error   string
	mu      sync.RWMutex
}

// DashboardManager manages dashboard download tasks
type DashboardManager struct {
	tasks            map[string]*DashboardTask
	mu               sync.RWMutex
	initStateManager initstate.InitStateManager
	// Optional. When set, a successful install writes the directory it landed
	// in back to `experimental.clash_api.external_ui` — see persistExternalUI.
	configManager *config.Manager
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager(initStateManager initstate.InitStateManager, configManager *config.Manager) *DashboardManager {
	return &DashboardManager{
		tasks:            make(map[string]*DashboardTask),
		initStateManager: initStateManager,
		configManager:    configManager,
	}
}

// persistExternalUI points sing-box at the directory the dashboard was just
// installed into.
//
// Downloading used to leave `external_ui` untouched, which made the button a
// no-op from sing-box's point of view: the files were on disk, the config never
// mentioned them, and the Clash API served 404 for /ui. Nothing else in the
// panel writes this key on the operator's behalf — the wizard asks for it, and
// an already-initialised install had no path to it except typing it and
// remembering to press Save.
//
// It is written unconditionally rather than only-when-empty: the operator asked
// for the dashboard to be installed HERE, so a stale path from a previous
// install is exactly the value that should be replaced.
func (m *DashboardManager) persistExternalUI(dir string) {
	if m.configManager == nil || dir == "" {
		return
	}

	// UpdateConfig validates and snapshots a version on every call, so a no-op
	// write would add a history entry saying nothing changed.
	if cfg, err := m.configManager.GetConfig(); err == nil &&
		cfg.Experimental != nil && cfg.Experimental.ClashAPI != nil &&
		cfg.Experimental.ClashAPI.ExternalUI == dir {
		return
	}

	err := m.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Experimental == nil {
			cfg.Experimental = &config.ExperimentalConfig{}
		}
		if cfg.Experimental.ClashAPI == nil {
			cfg.Experimental.ClashAPI = &config.ClashAPIConfig{}
		}
		cfg.Experimental.ClashAPI.ExternalUI = dir
		return nil
	})
	if err != nil {
		// The files are installed either way; failing the task here would say
		// the download failed, which is not what happened.
		logger.Error("Failed to write external_ui to config",
			zap.String("external_ui", dir), zap.Error(err))
		return
	}
	logger.Info("external_ui updated", zap.String("external_ui", dir))
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
		err := m.downloadAndExtract(task, targetDir, downloadURL, proxy)
		m.finishTask(task, targetDir, err)
	}()

	return task, nil
}

// finishTask records the outcome of an install shared by both entry points:
// mark the task, flag the init state, and point sing-box at installDir.
func (m *DashboardManager) finishTask(task *DashboardTask, installDir string, err error) {
	task.mu.Lock()
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.Message = "Dashboard installation failed"
		task.mu.Unlock()
		return
	}
	task.Status = "completed"
	task.Message = fmt.Sprintf("Dashboard installed to: %s", installDir)
	task.mu.Unlock()

	if m.initStateManager != nil {
		if err := m.initStateManager.SetDashboardInstalled(); err != nil {
			// Don't fail the task, just log the error
			logger.Error("Failed to update initialization state", zap.Error(err))
		} else {
			logger.Info("Dashboard installation state updated")
		}
	}

	// The whole point of the download button: without this, the files sit on
	// disk and sing-box never learns where they are.
	m.persistExternalUI(installDir)
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

// downloadAndExtract downloads the dashboard zip and extracts it into targetDir.
func (m *DashboardManager) downloadAndExtract(task *DashboardTask, targetDir, downloadURL, proxy string) error {
	// Update progress: Creating temp file
	task.mu.Lock()
	task.Message = "Creating temporary file..."
	task.mu.Unlock()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "dashboard-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
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
			return fmt.Errorf("invalid proxy URL: %w", err)
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
		return fmt.Errorf("failed to download dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
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
		return fmt.Errorf("failed to write dashboard file: %w", err)
	}

	// Update progress: Download complete
	task.mu.Lock()
	task.Message = fmt.Sprintf("Download complete (%.2f MB). Extracting...", float64(written)/(1024*1024))
	task.mu.Unlock()

	// Extract, stripping the archive's wrapper directory (see extractDashboardZip).
	if err := extractDashboardZip(tmpFile.Name(), targetDir); err != nil {
		return fmt.Errorf("failed to extract dashboard: %w", err)
	}

	// Update progress: Extraction complete
	task.mu.Lock()
	task.Message = fmt.Sprintf("Dashboard extracted to: %s", targetDir)
	task.mu.Unlock()

	return nil
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
		installDir, err := m.extractUpload(task, zipPath, targetDir, folderName)
		m.finishTask(task, installDir, err)

		// Clean up uploaded file
		os.Remove(zipPath)
	}()

	return task, nil
}

// extractUpload extracts an uploaded zip and returns the directory it landed in.
//
// folderName, when given, installs one level down (<targetDir>/<folderName>) so
// several dashboards can live side by side; external_ui is then pointed at that
// subdirectory rather than at targetDir.
func (m *DashboardManager) extractUpload(task *DashboardTask, zipPath, targetDir, folderName string) (string, error) {
	task.mu.Lock()
	task.Message = "Extracting uploaded dashboard..."
	task.mu.Unlock()

	installDir := targetDir
	if folderName != "" {
		installDir = filepath.Join(targetDir, folderName)
	}

	if err := extractDashboardZip(zipPath, installDir); err != nil {
		return "", fmt.Errorf("failed to extract dashboard: %w", err)
	}

	task.mu.Lock()
	task.Message = fmt.Sprintf("Dashboard extracted to: %s", installDir)
	task.mu.Unlock()

	return installDir, nil
}

// GetDashboardStatus checks if dashboard is installed
func (m *DashboardManager) GetDashboardStatus(targetDir string) (bool, error) {
	// Check if dashboard directory exists and has files
	if _, err := os.Stat(filepath.Join(targetDir, "index.html")); err == nil {
		return true, nil
	}

	return false, nil
}

// extractDashboardZip extracts a dashboard archive into targetDir, stripping the
// archive's single wrapper directory when it has one.
//
// This is what sing-box itself does when it downloads external_ui
// (zipIsInSingleDirectory in experimental/clashapi/server_resources.go), and it
// has to match: sing-box serves external_ui as a plain static root and expects
// index.html directly inside it. GitHub's "archive/<branch>.zip" wraps
// everything in "<repo>-<branch>/", so extracting it verbatim landed the app at
// <targetDir>/zashboard-gh-pages/index.html — a 404 dashboard, and a
// GetDashboardStatus that reported "not installed" while the files sat right
// there on disk.
func extractDashboardZip(zipPath, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	root := zipRootDir(reader.File)
	cleanTarget := filepath.Clean(targetDir)

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := strings.TrimPrefix(file.Name, root)
		if name == "" {
			continue
		}

		targetPath := filepath.Join(cleanTarget, name)

		// Zip-slip: reject any entry that resolves outside the target root.
		if !strings.HasPrefix(targetPath, cleanTarget+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if err := writeZipEntry(file, targetPath); err != nil {
			return err
		}
	}

	// A build before the stripping above left the wrapper directory behind.
	// Removing it keeps the operator from finding two copies and guessing which
	// one external_ui points at.
	if root != "" {
		if stale := filepath.Join(cleanTarget, strings.TrimSuffix(root, "/")); stale != cleanTarget {
			os.RemoveAll(stale)
		}
	}

	return nil
}

// writeZipEntry copies one zip entry to savePath.
func writeZipEntry(file *zip.File, savePath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// zipRootDir returns the archive's common top-level directory as a "name/"
// prefix, or "" when its files do not all share one.
func zipRootDir(files []*zip.File) string {
	var root string
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		idx := strings.Index(file.Name, "/")
		if idx <= 0 {
			// A file at the archive root: nothing to strip.
			return ""
		}
		dir := file.Name[:idx+1]
		// "../" is not a wrapper directory. Stripping it would quietly turn a
		// traversal entry into an in-tree write instead of the rejection the
		// path check below is there to produce.
		if dir == "../" || dir == "./" {
			return ""
		}
		if root == "" {
			root = dir
		} else if root != dir {
			return ""
		}
	}
	return root
}
