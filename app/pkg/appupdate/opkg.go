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
// file is a sequence of blank-line-separated stanzas, each with a
// "Package: <name>" line and a "Status: <want> <flag> <state>" line.
//
// The Status line must confirm the package is actually installed: a stanza
// left behind by an interrupted transaction (e.g. "deinstall ok
// not-installed") must not permanently lock out tarball self-updates.
func installedViaOpkg(statusPath string) bool {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		// Not an OpenWrt host (or no opkg): tarball updates are fine.
		return false
	}

	var matchesPackage, isInstalled bool
	flush := func() bool { return matchesPackage && isInstalled }

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Stanza boundary.
			if flush() {
				return true
			}
			matchesPackage, isInstalled = false, false
			continue
		}
		if trimmed == "Package: "+opkgPackageName {
			matchesPackage = true
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "Status:"); ok {
			// e.g. "install user installed" / "deinstall ok not-installed".
			fields := strings.Fields(rest)
			isInstalled = len(fields) > 0 && fields[len(fields)-1] == "installed"
		}
	}
	return flush()
}
