package service

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ServiceInfo is a richer status snapshot than the bare running bool returned by
// Status(). It is what the HTTP status endpoint serializes for the UI: the
// sidebar uses Running, the overview page additionally shows PID and the last
// start time.
type ServiceInfo struct {
	Running bool `json:"running"`
	// PID is the main sing-box process id, or 0 when not running / unknown.
	PID int `json:"pid"`
	// StartedAtUnix is the wall-clock start time of the process in Unix seconds,
	// or 0 when not running / could not be determined. The frontend renders the
	// relative "14 min ago" from this so locale formatting stays client-side.
	StartedAtUnix int64 `json:"started_at"`
	// Uptime is a coarse human-readable duration (e.g. "14m", "2h3m"), provided
	// as a server-side fallback. Empty when not running / unknown.
	Uptime string `json:"uptime"`
	// Version is the version of the installed sing-box binary.
	Version string `json:"version"`
}

// Version returns the version of the installed sing-box binary
func (c *Controller) Version() string {
	cmd := exec.Command(c.singBoxPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return parseVersion(string(output))
}

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

// Info returns an enriched status snapshot. It never fails just because the
// start time could not be resolved — those fields degrade to zero values so the
// caller always learns at least whether the service is running.
func (c *Controller) Info() (ServiceInfo, error) {
	running, pid, err := c.runningAndPID()
	if err != nil {
		return ServiceInfo{}, err
	}

	info := ServiceInfo{
		Running: running,
		PID:     pid,
		Version: c.Version(),
	}
	if running && pid > 0 {
		if started, ok := processStartedAtUnix(pid); ok {
			info.StartedAtUnix = started
			info.Uptime = humanizeDuration(time.Since(time.Unix(started, 0)))
		}
	}
	return info, nil
}

// runningAndPID resolves both the running state and the main PID in one pass.
// Under systemd a single `systemctl show` yields ActiveState + MainPID; without
// systemd it falls back to the existing pgrep/pidof-based lookup.
func (c *Controller) runningAndPID() (bool, int, error) {
	if c.useSystemd {
		active, pid := c.systemctlActiveAndPID()
		return active, pid, nil
	}

	pidStr, err := c.getPID()
	if err != nil {
		return false, 0, fmt.Errorf("failed to check service status: %w", err)
	}
	if pidStr == "" {
		return false, 0, nil
	}
	pid, _ := strconv.Atoi(pidStr)
	return true, pid, nil
}

// systemctlActiveAndPID parses `systemctl show` for the unit's active state and
// main pid. A failed/inactive unit reports active=false; MainPID is 0 when the
// unit is not running.
func (c *Controller) systemctlActiveAndPID() (bool, int) {
	out, _ := exec.Command("systemctl", "show", singBoxServiceName,
		"--property=ActiveState", "--property=MainPID").Output()
	return parseActiveAndPID(string(out))
}

// parseActiveAndPID extracts ActiveState/MainPID from `systemctl show` output.
// Kept pure (no exec) so it is unit-testable. A non-active unit always reports
// pid 0 regardless of the MainPID line.
func parseActiveAndPID(out string) (bool, int) {
	var active bool
	var pid int
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "ActiveState":
			active = val == "active"
		case "MainPID":
			pid, _ = strconv.Atoi(val)
		}
	}
	if !active {
		return false, 0
	}
	return true, pid
}

// processStartedAtUnix derives a process's wall-clock start time from its
// elapsed run time via `ps -o etimes=` (whole seconds since start). This avoids
// parsing systemd's locale-formatted timestamps and works for both the systemd
// and the directly-managed process paths. Returns ok=false when the value
// cannot be read (e.g. unsupported `ps`, or the process has exited).
func processStartedAtUnix(pid int) (int64, bool) {
	if pid <= 0 {
		return 0, false
	}
	out, err := exec.Command("ps", "-o", "etimes=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, false
	}
	elapsed, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || elapsed < 0 {
		return 0, false
	}
	return time.Now().Unix() - elapsed, true
}

// humanizeDuration renders a duration coarsely: "<1m", "14m", "2h3m", "1d2h".
// Used only as a server-side fallback for the Uptime field.
func humanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh%dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}
