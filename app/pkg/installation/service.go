package installer

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation/state"
	"io"
	"os"
)

type Service struct {
	installer        *Manager
	dashboardManager *DashboardManager
	configManager    *config.Manager
	initStateManager initstate.InitStateManager
}

func NewService(core *Manager, dashboard *DashboardManager, cm *config.Manager, state initstate.InitStateManager) *Service {
	return &Service{core, dashboard, cm, state}
}
func (s *Service) readClashAPISettings() (config.ClashAPISettings, error) {
	return s.configManager.GetClashAPISettings()
}

// UploadDashboard owns the temporary input until the extraction worker consumes it.
func (s *Service) UploadDashboard(input io.Reader, targetDir, folderName string) (any, error) {
	target, err := sanitizeTargetDir(targetDir)
	if err != nil {
		return nil, fault.New(fault.Invalid, err.Error())
	}
	folder, err := sanitizeFolderName(folderName)
	if err != nil {
		return nil, fault.New(fault.Invalid, err.Error())
	}
	if target == "" {
		if clash, err := s.readClashAPISettings(); err == nil {
			target = clash.ExternalUI
		}
	}
	if target == "" {
		target = "/etc/sing-box/ui"
	}
	file, err := os.CreateTemp("", "dashboard-upload-*.zip")
	if err != nil {
		return nil, err
	}
	handedOff := false
	defer func() {
		file.Close()
		if !handedOff {
			os.Remove(file.Name())
		}
	}()
	if _, err := io.Copy(file, input); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	task, err := s.dashboardManager.UploadDashboard(file.Name(), target, folder)
	if err != nil {
		return nil, err
	}
	handedOff = true
	return map[string]any{"message": "dashboard upload started", "task_id": task.ID}, nil
}
