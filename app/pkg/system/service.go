package system

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/platform/sysinfo"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox"
	"path/filepath"
)

type Service struct {
	configPath, databasePath string
	serviceController        *service.Controller
	systemType               service.SystemType
}

func New(configPath, databasePath string, controller *service.Controller) *Service {
	return &Service{configPath, databasePath, controller, controller.SystemType()}
}

type SystemInfoResponse struct {
	// SystemType is the distribution family: "openwrt", "debian" or "unknown".
	SystemType string `json:"system_type"`
	// ServiceBackend is how sing-box lifecycle is driven: "systemd", "procd"
	// or "process".
	ServiceBackend string `json:"service_backend"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	CPUCores       int    `json:"cpu_cores"`
	Hostname       string `json:"hostname"`
	Kernel         string `json:"kernel"`
	Distribution   string `json:"distribution"`
	// AppVersion is this panel's version. AppVersionKnown is false for dev
	// builds where the release ldflag was never stamped in.
	AppVersion      string `json:"app_version"`
	AppVersionKnown bool   `json:"app_version_known"`
	// SingBoxVersion is the installed sing-box binary's version, or "unknown"
	// when the binary is missing or not executable.
	SingBoxVersion string `json:"sing_box_version"`
	// Disks reports free space on the filesystems the panel writes to. A full
	// filesystem otherwise surfaces as an opaque driver error (SQLite reports
	// SQLITE_CANTOPEN when it cannot create its journal), so the operator needs
	// to see this before they go hunting for a permissions bug.
	Disks []sysinfo.DiskUsage `json:"disks"`
}

func (h *Service) Info() SystemInfoResponse {
	// The config directory and the database directory are the two places a
	// write can fail; they are usually the same filesystem, and CollectDisks
	// collapses them when they are.
	host := sysinfo.Collect(
		filepath.Dir(h.configPath),
		filepath.Dir(h.databasePath),
	)

	return SystemInfoResponse{
		SystemType:      string(h.systemType),
		ServiceBackend:  h.serviceController.BackendKind(),
		OS:              host.OS,
		Arch:            host.Arch,
		CPUCores:        host.CPUCores,
		Hostname:        host.Hostname,
		Kernel:          host.Kernel,
		Distribution:    host.Distribution,
		AppVersion:      appupdate.Current(),
		AppVersionKnown: appupdate.IsKnown(),
		SingBoxVersion:  h.serviceController.Version(),
		Disks:           host.Disks,
	}
}
