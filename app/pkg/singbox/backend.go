package singbox

import (
	"context"
	"os"
	"os/exec"

	"github.com/SealinGp/sing-box-easy/app/pkg/platform/sysinfo"
)

// Backend kinds reported in logs and (potentially) status payloads.
const (
	BackendSystemd = "systemd"
	BackendProcd   = "procd"
	BackendProcess = "process"
)

// procdInitScript is the OpenWrt init script installed by the official
// sing-box opkg package (and by common manual installs).
const procdInitScript = "/etc/init.d/sing-box"

// Backend abstracts how the sing-box service lifecycle is managed on this
// host. Three implementations exist:
//
//   - systemdBackend: sing-box installed as a systemd unit (official Debian
//     install) — delegates to systemctl and reads logs from journald.
//   - procdBackend: OpenWrt with an /etc/init.d/sing-box procd script —
//     delegates to the init script and reads logs from the syslog ring
//     buffer via logread.
//   - processBackend: no init system integration — sing-box is spawned and
//     signaled directly by this application.
//
// Backends assume the caller (Controller) has already validated the config
// before any Start/Restart/Reload.
type Backend interface {
	// Kind returns one of the Backend* constants.
	Kind() string
	Start() error
	// Stop is idempotent and blocks until the process has exited.
	Stop() error
	ForceStop() error
	Restart() error
	// Reload applies the current config without a full restart when the
	// backend supports it, falling back to Restart internally otherwise.
	Reload() error
	// ActiveAndPID reports whether sing-box is running and its main PID
	// (0 when not running or unknown).
	ActiveAndPID() (bool, int, error)
	// TailLogs returns the most recent log lines from this backend's log
	// source. lines is already clamped by the caller.
	TailLogs(lines int, afterCursor string) (LogChunk, error)
	// FollowLogs pushes new lines as they are written, until ctx is
	// cancelled. The returned channel is closed when the follower stops.
	//
	// A backend with no readable log source returns (nil, nil) rather than an
	// error: "there is nowhere to read logs from" is a supported state the UI
	// already explains via LogChunk.Source, not a failure.
	//
	// Callers MUST cancel ctx when they stop reading. Two of the three
	// implementations hold a child process open for the life of the follow —
	// see follow.go.
	FollowLogs(ctx context.Context) (<-chan FollowEvent, error)
}

// detectBackend picks the service backend for this host. Detection order:
// systemd unit → procd init script → direct process management.
func detectBackend(systemType sysinfo.SystemType, singBoxPath string, configPath, logPath func() string) Backend {
	if detectSystemd() {
		return &systemdBackend{}
	}
	if hasProcdInitScript(systemType) {
		return &procdBackend{initScript: procdInitScript, logPath: logPath}
	}
	return newProcessBackend(systemType, singBoxPath, configPath, logPath)
}

// detectSystemd reports whether sing-box is managed by a systemd unit on this
// host. It is true only when `systemctl` is available AND a sing-box unit file
// exists, which is the case for the official Debian/Linux installation.
// OpenWrt (procd-based) and hosts without systemd return false.
func detectSystemd() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	// `systemctl cat <unit>` exits 0 only when the unit file exists; output is
	// irrelevant, we only care about the exit status.
	return exec.Command("systemctl", "cat", singBoxServiceName).Run() == nil
}

// hasProcdInitScript reports whether this is an OpenWrt host with an
// executable /etc/init.d/sing-box script to delegate to.
func hasProcdInitScript(systemType sysinfo.SystemType) bool {
	if systemType != sysinfo.SystemOpenWRT {
		return false
	}
	fi, err := os.Stat(procdInitScript)
	if err != nil || fi.IsDir() {
		return false
	}
	return fi.Mode().Perm()&0o111 != 0
}
