package v1_13_0

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// sanitizeFolderName ensures a user-supplied folder name is a single,
// non-traversing path component. Returns the cleaned name or an error.
func sanitizeFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if strings.ContainsAny(name, `/\`) || name == ".." || name == "." {
		return "", fmt.Errorf("folder_name must be a single path component, got %q", name)
	}
	// Defence in depth: ensure filepath.Base agrees.
	if filepath.Base(name) != name {
		return "", fmt.Errorf("folder_name must be a single path component, got %q", name)
	}
	return name, nil
}

// sanitizeTargetDir validates a user-supplied target directory.
// Requires an absolute path with no ".." segments after cleaning.
func sanitizeTargetDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", nil
	}
	cleaned := filepath.Clean(dir)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("target_dir must be an absolute path, got %q", dir)
	}
	// filepath.Clean strips redundant ".." pairs, but a path like
	// "/foo/../etc" cleans to "/etc" — still legal absolute. We reject
	// any path whose pre-clean form differs in a way that suggests escape.
	if strings.Contains(dir, "..") {
		return "", fmt.Errorf("target_dir must not contain '..' segments, got %q", dir)
	}
	return cleaned, nil
}

// InstallSingBox installs sing-box
func (h *Handler) InstallSingBox(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Version string `json:"version"`
		Beta    bool   `json:"beta"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	task, err := h.installer.InstallSingBox(req.Version, req.Beta)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "sing-box installation started",
		"task_id": task.ID,
	})
}

// GetInstallTask returns installation task status by task ID
func (h *Handler) GetInstallTask(ctx context.Context, c *app.RequestContext) {
	taskID := c.Param("task_id")
	if taskID == "" {
		respErr(ctx, c, CodeBadRequest, "task_id is required")
		return
	}

	task, err := h.installer.GetTask(taskID)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"id":      task.ID,
		"status":  task.Status,
		"message": task.Message,
		"error":   task.Error,
	})
}

// GetInstallStatus returns installation status (checks if sing-box is installed)
func (h *Handler) GetInstallStatus(ctx context.Context, c *app.RequestContext) {
	installed, version, err := h.installer.GetInstallStatus()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	message := "sing-box is not installed"
	if installed {
		message = "sing-box is installed"
	}

	respOK(ctx, c, map[string]any{
		"installed": installed,
		"version":   version,
		"message":   message,
	})
}

// UpdateSingBox updates sing-box
func (h *Handler) UpdateSingBox(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Version string `json:"version"`
		Beta    bool   `json:"beta"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	task, err := h.installer.UpdateSingBox(req.Version, req.Beta)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "sing-box update started",
		"task_id": task.ID,
	})
}

// DownloadDashboard downloads dashboard UI
func (h *Handler) DownloadDashboard(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		TargetDir   string `json:"target_dir"`
		DownloadURL string `json:"download_url"`
		Proxy       string `json:"proxy"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Get target dir from config if not specified
	targetDir := req.TargetDir
	downloadURL := req.DownloadURL
	if targetDir == "" {
		cfg, err := h.configManager.GetConfig()
		if err == nil && cfg.Experimental != nil && cfg.Experimental.ClashAPI != nil {
			targetDir = cfg.Experimental.ClashAPI.ExternalUI
			// Also try to get download URL from config if not provided
			if downloadURL == "" && cfg.Experimental.ClashAPI.ExternalUIDownloadURL != "" {
				downloadURL = cfg.Experimental.ClashAPI.ExternalUIDownloadURL
			}
		}
	}

	if targetDir == "" {
		targetDir = "/etc/sing-box/ui"
	}

	task, err := h.dashboardManager.DownloadDashboard(targetDir, downloadURL, req.Proxy)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "dashboard download started",
		"task_id": task.ID,
	})
}

// GetDashboardTask returns dashboard download task status by task ID
func (h *Handler) GetDashboardTask(ctx context.Context, c *app.RequestContext) {
	taskID := c.Param("task_id")
	if taskID == "" {
		respErr(ctx, c, CodeBadRequest, "task_id is required")
		return
	}

	task, err := h.dashboardManager.GetTask(taskID)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"id":      task.ID,
		"status":  task.Status,
		"message": task.Message,
		"error":   task.Error,
	})
}

// GetDashboardStatus returns dashboard status (checks if dashboard is installed)
func (h *Handler) GetDashboardStatus(ctx context.Context, c *app.RequestContext) {
	// Get target dir from config
	targetDir := "/etc/sing-box/ui"
	cfg, err := h.configManager.GetConfig()
	if err == nil && cfg.Experimental != nil && cfg.Experimental.ClashAPI != nil {
		if cfg.Experimental.ClashAPI.ExternalUI != "" {
			targetDir = cfg.Experimental.ClashAPI.ExternalUI
		}
	}

	installed, err := h.dashboardManager.GetDashboardStatus(targetDir)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"installed": installed,
		"path":      targetDir,
	})
}

// UploadDashboard uploads and extracts a dashboard zip file
func (h *Handler) UploadDashboard(ctx context.Context, c *app.RequestContext) {
	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	// Get uploaded file
	files := form.File["file"]
	if len(files) == 0 {
		respErr(ctx, c, CodeBadRequest, "no file uploaded")
		return
	}

	uploadedFile := files[0]

	// Get optional parameters. Both are user-supplied and flow into
	// filesystem operations (extraction root + final folder rename),
	// so they must be validated before use.
	rawTargetDir := c.PostForm("target_dir")
	rawFolderName := c.PostForm("folder_name")

	targetDir, err := sanitizeTargetDir(rawTargetDir)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	folderName, err := sanitizeFolderName(rawFolderName)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}

	// Get target dir from config if not specified
	if targetDir == "" {
		cfg, err := h.configManager.GetConfig()
		if err == nil && cfg.Experimental != nil && cfg.Experimental.ClashAPI != nil {
			targetDir = cfg.Experimental.ClashAPI.ExternalUI
		}
	}

	if targetDir == "" {
		targetDir = "/etc/sing-box/ui"
	}

	// Save uploaded file to temp location
	tmpFile, err := os.CreateTemp("", "dashboard-upload-*.zip")
	if err != nil {
		respErr(ctx, c, CodeInternalError, "failed to create temp file: "+err.Error())
		return
	}
	tmpFile.Close()

	// Save uploaded file
	if err := c.SaveUploadedFile(uploadedFile, tmpFile.Name()); err != nil {
		os.Remove(tmpFile.Name())
		respErr(ctx, c, CodeInternalError, "failed to save uploaded file: "+err.Error())
		return
	}

	// Process upload
	task, err := h.dashboardManager.UploadDashboard(tmpFile.Name(), targetDir, folderName)
	if err != nil {
		os.Remove(tmpFile.Name())
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "dashboard upload started",
		"task_id": task.ID,
	})
}
