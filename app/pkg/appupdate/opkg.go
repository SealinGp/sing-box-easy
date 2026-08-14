package appupdate

import (
	"os"
	"strings"
)

// opkgStatusPath is opkg's installed-package registry on OpenWrt.
const opkgStatusPath = "/usr/lib/opkg/status"

// opkgPackageName is the name our OpenWrt ipk installs under.
const opkgPackageName = "sing-box-easy"

// InstalledViaOpkg reports whether this binary was installed by the OpenWrt
// ipk package. Tarball self-updates must not run in that case: they would
// swap files behind opkg's back and desync its file registry, so updates go
// through opkg instead.
func InstalledViaOpkg() bool {
	return installedViaOpkg(opkgStatusPath)
}

// installedViaOpkg is the testable core of InstalledViaOpkg. The opkg status
// file is a sequence of stanzas each starting with a "Package: <name>" line.
func installedViaOpkg(statusPath string) bool {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		// Not an OpenWrt host (or no opkg): tarball updates are fine.
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "Package: "+opkgPackageName {
			return true
		}
	}
	return false
}
