package service

import (
	"fmt"
	"os"
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
	running, pid, err := c.backend.ActiveAndPID()
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

// linuxClockTicksPerSecond is the kernel USER_HZ value used for the
// /proc/<pid>/stat starttime field. It has been fixed at 100 on Linux for all
// supported architectures.
const linuxClockTicksPerSecond = 100

// processStartedAtUnix derives a process's wall-clock start time.
//
// It first reads /proc/<pid>/stat (starttime, ticks since boot) plus
// /proc/stat's btime — a pure-file approach that works on every Linux
// including BusyBox userlands (OpenWrt) where `ps` supports no -o flags. On
// non-procfs systems it falls back to `ps -o etimes=`. Returns ok=false when
// the value cannot be read (e.g. the process has exited).
func processStartedAtUnix(pid int) (int64, bool) {
	if pid <= 0 {
		return 0, false
	}
	if started, ok := procStartedAtUnix(pid); ok {
		return started, true
	}
	return psStartedAtUnix(pid)
}

// procStartedAtUnix computes start time from procfs: btime (boot time in Unix
// seconds, from /proc/stat) + starttime (clock ticks after boot, field 22 of
// /proc/<pid>/stat).
func procStartedAtUnix(pid int) (int64, bool) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, false
	}
	startTicks, ok := parseProcStatStartTicks(string(stat))
	if !ok {
		return 0, false
	}

	sysStat, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, false
	}
	btime, ok := parseBootTimeUnix(string(sysStat))
	if !ok {
		return 0, false
	}
	return btime + startTicks/linuxClockTicksPerSecond, true
}

// parseProcStatStartTicks extracts field 22 (starttime) from a
// /proc/<pid>/stat line. The comm field (2) may contain spaces and is wrapped
// in parentheses, so fields are counted after the closing paren. Kept pure
// (no filesystem access) so it is unit-testable.
func parseProcStatStartTicks(stat string) (int64, bool) {
	// Everything after the last ')' is a well-defined space-separated list
	// starting with field 3 (state).
	idx := strings.LastIndexByte(stat, ')')
	if idx < 0 {
		return 0, false
	}
	fields := strings.Fields(stat[idx+1:])
	// starttime is overall field 22 → index 19 relative to field 3.
	const startTimeIdx = 19
	if len(fields) <= startTimeIdx {
		return 0, false
	}
	ticks, err := strconv.ParseInt(fields[startTimeIdx], 10, 64)
	if err != nil || ticks < 0 {
		return 0, false
	}
	return ticks, true
}

// parseBootTimeUnix extracts the "btime <unix-seconds>" line from /proc/stat.
// Kept pure (no filesystem access) so it is unit-testable.
func parseBootTimeUnix(sysStat string) (int64, bool) {
	for _, line := range strings.Split(sysStat, "\n") {
		val, ok := strings.CutPrefix(line, "btime ")
		if !ok {
			continue
		}
		btime, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err != nil || btime <= 0 {
			return 0, false
		}
		return btime, true
	}
	return 0, false
}

// psStartedAtUnix derives the start time from a process's elapsed run time
// via `ps -o etimes=` (whole seconds since start). Fallback for hosts without
// procfs (e.g. macOS during development).
func psStartedAtUnix(pid int) (int64, bool) {
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
