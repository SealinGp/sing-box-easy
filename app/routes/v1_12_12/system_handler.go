package v1_13_0

import (
	"context"
	"path/filepath"

	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/sysinfo"
	"github.com/cloudwego/hertz/pkg/app"
)

// SystemInfoResponse describes the host and the two versions the operator cares
// about. It backs the Settings "About" card, which merges what used to be three
// separate panels (about, update, language) into one place.
//
// Authenticated: hostname and kernel are mild fingerprinting material, so only
// the coarse platform family is exposed publicly (see AuthStatusResponse).
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

// GetSystemInfo returns host and version details for the About card.
//
// GET /system/info
func (h *Handler) GetSystemInfo(ctx context.Context, c *app.RequestContext) {
	// The config directory and the database directory are the two places a
	// write can fail; they are usually the same filesystem, and CollectDisks
	// collapses them when they are.
	host := sysinfo.Collect(
		filepath.Dir(h.configManager.GetConfigPath()),
		filepath.Dir(database.Path()),
	)

	respOK(ctx, c, SystemInfoResponse{
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
	})
}
