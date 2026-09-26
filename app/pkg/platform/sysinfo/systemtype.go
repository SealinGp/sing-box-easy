package sysinfo

import (
	"os"
	"strings"
)

// SystemType is the host's distribution family, as far as this panel needs to
// tell them apart. Distribution (in Info) is the display name; this is the
// classification code branches on.
type SystemType string

const (
	SystemDebian  SystemType = "debian"
	SystemOpenWRT SystemType = "openwrt"
	SystemUnknown SystemType = "unknown"
)

// DetectSystemType detects whether the system is OpenWrt, Debian, or other.
// It is a host fact, not a sing-box one: the service controller picks its
// init-system backend from it, the installer picks opkg vs the install
// script, and the HTTP layer picks the login default and navigation layout.
func DetectSystemType() SystemType {
	// Check for OpenWrt first
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		return SystemOpenWRT
	}

	// Check for Debian-based systems
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return SystemDebian
	}

	// Check /etc/os-release for more info
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "openwrt") {
			return SystemOpenWRT
		}
		if strings.Contains(content, "debian") || strings.Contains(content, "ubuntu") {
			return SystemDebian
		}
	}

	// Default to unknown, but compatible commands will still be attempted.
	return SystemUnknown
}
