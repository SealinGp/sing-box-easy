package appupdate

import (
	"os"
	"path/filepath"
	"strings"
)

// opkgStatusPath is opkg's installed-package registry on OpenWrt.
const opkgStatusPath = "/usr/lib/opkg/status"

// opkgPackageName is the name our OpenWrt ipk installs under.
const opkgPackageName = "sing-box-easy"

// opkgConfPath is opkg's main configuration file, which declares where the
// downloaded feed indexes are cached (`lists_dir <type> <path>`).
const opkgConfPath = "/etc/opkg.conf"

// defaultOpkgListsDir is opkg's built-in lists location, used when opkg.conf
// declares no lists_dir.
const defaultOpkgListsDir = "/var/opkg-lists"

// OpkgInstall describes how this binary is registered with opkg.
//
// Note what is deliberately absent: where the package came from. opkg records
// no provenance — a package pulled from a configured feed and one installed
// from a hand-downloaded .ipk produce byte-identical stanzas (verified against
// an ImmortalWrt 23.05.1 box; the full field set is Package, Version, Depends,
// Status, Architecture, Installed-Time, Conffiles, Provides, Conflicts,
// Alternatives, Essential, ABIVersion, Auto-Installed). So "how was this
// installed" is unanswerable after the fact, and the useful question is
// instead "can a configured feed upgrade it right now" — see feedProvides.
type OpkgInstall struct {
	// Managed is true when opkg owns this package.
	Managed bool
	// Architecture is the opkg arch of the installed package (e.g. "x86_64",
	// "aarch64_generic"). It names the exact ipk variant to fetch, and is not
	// derivable from runtime.GOARCH, which only knows "amd64"/"arm64".
	Architecture string
	// Version is the package version opkg has on record (no leading "v").
	Version string
}

// InspectOpkg reports how this binary is registered with opkg.
func InspectOpkg() OpkgInstall {
	return inspectOpkg(opkgStatusPath)
}

// inspectOpkg is the testable core of InspectOpkg.
//
// The status file is a sequence of blank-line-separated stanzas. Fields are
// accumulated per stanza and only committed at the boundary, so a neighbouring
// package's Architecture cannot leak into ours. Real boxes carry duplicate
// stanzas for one package (an installed one plus a leftover not-installed one
// from an interrupted transaction), so the installed stanza wins.
func inspectOpkg(statusPath string) OpkgInstall {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		// Not an OpenWrt host, or no opkg.
		return OpkgInstall{}
	}

	var result OpkgInstall
	var stanza OpkgInstall
	var matchesPackage bool

	commit := func() {
		if matchesPackage && stanza.Managed && !result.Managed {
			result = stanza
		}
		stanza, matchesPackage = OpkgInstall{}, false
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			commit()
			continue
		}

		key, value, found := strings.Cut(trimmed, ":")
		if !found {
			// Continuation line (e.g. a Conffiles entry).
			continue
		}
		value = strings.TrimSpace(value)

		switch key {
		case "Package":
			matchesPackage = value == opkgPackageName
		case "Architecture":
			stanza.Architecture = value
		case "Version":
			stanza.Version = value
		case "Status":
			// e.g. "install user installed" / "install prefer,user not-installed".
			fields := strings.Fields(value)
			stanza.Managed = len(fields) > 0 && fields[len(fields)-1] == "installed"
		}
	}
	commit()

	return result
}

// opkgListsDir resolves where opkg caches downloaded feed indexes, by reading
// the `lists_dir <type> <path>` directive from opkg.conf.
func opkgListsDir(confPath string) string {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return defaultOpkgListsDir
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		// "lists_dir ext /var/opkg-lists"
		if len(fields) >= 3 && fields[0] == "lists_dir" {
			return fields[2]
		}
	}
	return defaultOpkgListsDir
}

// feedProvides reports whether any configured feed offers the named package.
//
// The second return value distinguishes "no feed has it" from "we cannot tell":
// the lists directory is tmpfs on OpenWrt and starts empty after every boot
// until `opkg update` runs, so an empty cache means unknown, not absent.
// Reporting absence from an unpopulated cache would send the operator down the
// wrong upgrade path.
func feedProvides(listsDir, pkg string) (provides bool, known bool) {
	entries, err := os.ReadDir(listsDir)
	if err != nil {
		return false, false
	}

	needle := "Package: " + pkg
	for _, entry := range entries {
		// Feed indexes are flat files; opkg writes signature sidecars and
		// subdirectories that carry no package data.
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".sig") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(listsDir, entry.Name()))
		if err != nil {
			continue
		}
		known = true

		for _, line := range strings.Split(string(data), "\n") {
			// Exact match on the whole field: "Package: sing-box-easy-extras"
			// and "Package: sing-box" must not count.
			if strings.TrimSpace(line) == needle {
				return true, true
			}
		}
	}
	return false, known
}

// FeedProvidesSelf reports whether a configured feed can upgrade this package.
func FeedProvidesSelf() (provides bool, known bool) {
	return feedProvides(opkgListsDir(opkgConfPath), opkgPackageName)
}

// InstalledViaOpkg reports whether this binary was installed by the OpenWrt
// ipk package. Tarball self-updates must not run in that case: they would
// swap files behind opkg's back and desync its file registry, so updates go
// through opkg instead.
func InstalledViaOpkg() bool {
	return installedViaOpkg(opkgStatusPath)
}

// installedViaOpkg is the testable core of InstalledViaOpkg. It reuses the
// stanza parser in inspectOpkg rather than scanning the status file a second
// way — two parsers over the same format would eventually disagree about the
// interesting edge cases (leftover not-installed stanzas, field bleed across
// stanza boundaries).
func installedViaOpkg(statusPath string) bool {
	return inspectOpkg(statusPath).Managed
}
