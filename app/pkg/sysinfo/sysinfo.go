// Package sysinfo reports read-only facts about the host this panel runs on:
// CPU architecture, kernel release, distribution name and hostname.
//
// It exists because the Settings "About" card should tell the operator what
// they are actually running on — an OpenWrt router and a Debian VPS look
// identical from inside the browser, yet they need very different advice when
// something goes wrong.
//
// Every lookup degrades to a zero value instead of returning an error: an
// unreadable /etc/os-release is a cosmetic problem, never a reason to fail the
// request.
package sysinfo

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Info is a snapshot of the host. Fields are empty when the underlying source
// is missing (e.g. Kernel and Distribution on non-Linux dev machines).
type Info struct {
	// OS is the Go runtime target OS ("linux", "darwin", ...).
	OS string `json:"os"`
	// Arch is the Go runtime target architecture ("amd64", "arm64", "arm").
	Arch string `json:"arch"`
	// CPUCores is the number of usable logical CPUs.
	CPUCores int `json:"cpu_cores"`
	// Hostname is the host's network name, empty when it cannot be resolved.
	Hostname string `json:"hostname"`
	// Kernel is the running kernel release, e.g. "5.15.137".
	Kernel string `json:"kernel"`
	// Distribution is a human-readable OS name, e.g. "OpenWrt 23.05.2" or
	// "Debian GNU/Linux 12 (bookworm)".
	Distribution string `json:"distribution"`
	// Disks reports free space on the filesystems this panel writes to, one
	// entry per distinct filesystem. Empty when statfs is unavailable.
	Disks []DiskUsage `json:"disks"`
}

const (
	osReleasePath  = "/etc/os-release"
	kernelInfoPath = "/proc/sys/kernel/osrelease"
)

// Collect gathers host information from the standard Linux locations.
//
// diskPaths are the directories whose free space the operator needs to see —
// typically the sing-box config directory and the panel's database directory.
// Paths sharing a filesystem are reported once.
func Collect(diskPaths ...string) Info {
	info := collect(osReleasePath, kernelInfoPath)
	info.Disks = CollectDisks(diskPaths...)
	return info
}

// collect is the testable core of Collect, with the file locations injected.
func collect(osReleaseFile, kernelFile string) Info {
	info := Info{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}
	if data, err := os.ReadFile(kernelFile); err == nil {
		info.Kernel = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(osReleaseFile); err == nil {
		info.Distribution = parseDistribution(string(data))
	}

	return info
}

// parseDistribution extracts a display name from an os-release file, preferring
// PRETTY_NAME and falling back to NAME (plus VERSION when present). OpenWrt
// ships PRETTY_NAME, so the fallback only matters on unusual images.
func parseDistribution(content string) string {
	fields := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = unquote(value)
	}

	if pretty := fields["PRETTY_NAME"]; pretty != "" {
		return pretty
	}

	name := fields["NAME"]
	if version := fields["VERSION"]; name != "" && version != "" {
		return name + " " + version
	}
	return name
}

// unquote strips the optional quoting os-release values carry. Shell-style
// single quotes are not escape-processed, so they are only trimmed.
func unquote(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	return strings.Trim(value, `"'`)
}
