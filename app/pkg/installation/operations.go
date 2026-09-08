package installer

import (
	"context"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"path/filepath"
	"strings"
)

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

type InstallSingBoxCommand struct {
	Version string `json:"version"`
	Beta    bool   `json:"beta"`
}

func (h *Service) InstallSingBox(ctx context.Context, req InstallSingBoxCommand) (any, error) {

	task, err := h.installer.InstallSingBox(req.Version, req.Beta)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "sing-box installation started",
		"task_id": task.ID,
	}, nil
}
func (h *Service) GetInstallTask(ctx context.Context, taskID string) (any, error) {

	if taskID == "" {
		return nil, fault.New(fault.Input, "task_id is required")
	}

	task, err := h.installer.GetTask(taskID)
	if err != nil {
		return nil, fault.New(fault.Missing, err.Error())
	}

	return map[string]any{
		"id":      task.ID,
		"status":  task.Status,
		"message": task.Message,
		"error":   task.Error,
	}, nil
}
func (h *Service) GetInstallStatus(ctx context.Context) (any, error) {
	installed, version, err := h.installer.GetInstallStatus()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	message := "sing-box is not installed"
	if installed {
		message = "sing-box is installed"
	}

	return map[string]any{
		"installed": installed,
		"version":   version,
		"message":   message,
	}, nil
}

type UpdateSingBoxCommand struct {
	Version string `json:"version"`
	Beta    bool   `json:"beta"`
}

func (h *Service) UpdateSingBox(ctx context.Context, req UpdateSingBoxCommand) (any, error) {

	task, err := h.installer.UpdateSingBox(req.Version, req.Beta)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "sing-box update started",
		"task_id": task.ID,
	}, nil
}

type DownloadDashboardCommand struct {
	TargetDir   string `json:"target_dir"`
	DownloadURL string `json:"download_url"`
	Proxy       string `json:"proxy"`
}

func (h *Service) DownloadDashboard(ctx context.Context, req DownloadDashboardCommand) (any, error) {

	// Get target dir from config if not specified
	targetDir := req.TargetDir
	downloadURL := req.DownloadURL
	if targetDir == "" {
		clash, err := h.readClashAPISettings()
		if err == nil {
			targetDir = clash.ExternalUI
			// Also try to get download URL from config if not provided
			if downloadURL == "" && clash.ExternalUIDownloadURL != "" {
				downloadURL = clash.ExternalUIDownloadURL
			}
		}
	}

	if targetDir == "" {
		targetDir = "/etc/sing-box/ui"
	}

	task, err := h.dashboardManager.DownloadDashboard(targetDir, downloadURL, req.Proxy)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "dashboard download started",
		"task_id": task.ID,
	}, nil
}
func (h *Service) GetDashboardTask(ctx context.Context, taskID string) (any, error) {

	if taskID == "" {
		return nil, fault.New(fault.Input, "task_id is required")
	}

	task, err := h.dashboardManager.GetTask(taskID)
	if err != nil {
		return nil, fault.New(fault.Missing, err.Error())
	}

	return map[string]any{
		"id":      task.ID,
		"status":  task.Status,
		"message": task.Message,
		"error":   task.Error,
	}, nil
}
func (h *Service) GetDashboardStatus(ctx context.Context) (any, error) {
	// Get target dir from config
	targetDir := "/etc/sing-box/ui"
	clash, err := h.readClashAPISettings()
	if err == nil {
		if clash.ExternalUI != "" {
			targetDir = clash.ExternalUI
		}
	}

	installed, err := h.dashboardManager.GetDashboardStatus(targetDir)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"installed": installed,
		"path":      targetDir,
	}, nil
}
