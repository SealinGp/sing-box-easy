package appupdate

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Version is the running application version. It is stamped at build time via:
//
//	go build -ldflags "-X github.com/SealinGp/sing-box-easy/app/pkg/appupdate.Version=v1.2.3"
//
// Local/dev builds leave it empty and fall back to the on-disk version file (if
// a previous self-update wrote one) and finally to DevVersion.
var Version = ""

// DevVersion is reported when the running binary carries no version stamp.
const DevVersion = "dev"

// VersionFileName is written next to the binary after a successful self-update
// so that binaries built without the ldflags stamp still know what they are.
const VersionFileName = ".sing-box-easy-version"

var (
	currentOnce sync.Once
	currentVal  string
)

// Current returns the running application version, resolved once per process:
// build-time stamp → version file next to the executable → DevVersion.
func Current() string {
	currentOnce.Do(func() {
		currentVal = resolveCurrent()
	})
	return currentVal
}

// IsKnown reports whether the running version could be determined precisely.
// A dev build cannot be compared against release tags.
func IsKnown() bool {
	return Current() != DevVersion
}

func resolveCurrent() string {
	if v := strings.TrimSpace(Version); v != "" {
		return v
	}
	if v := readVersionFile(); v != "" {
		return v
	}
	return DevVersion
}

func readVersionFile() string {
	dir, err := InstallDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, VersionFileName))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// writeVersionFile records the freshly installed version next to the binary.
func writeVersionFile(dir, version string) error {
	return os.WriteFile(filepath.Join(dir, VersionFileName), []byte(version+"\n"), 0o644)
}

// backupSuffix is appended to the live binary when it is moved aside during an
// update. See canonicalBinaryName for why it has to be stripped again.
const backupSuffix = ".old"

// startupExecutable is os.Executable() captured at process start.
//
// This MUST NOT be re-read later. On Linux os.Executable() reads
// /proc/self/exe, which resolves to the *current* name of the running inode —
// so once an update renames the live binary to "<name>.old", every later call
// returns the backup path instead of the install path. Re-reading it after the
// swap is what previously made the updater re-exec the OLD binary (leaving the
// version unchanged) and then, on the next update, install to
// "<name>.old.old" and so on.
//
// Package-level initialisation runs before main, and therefore before any
// update can rename anything, so this is always the true path.
var startupExecutable = func() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}()

// canonicalBinaryName strips any trailing ".old" segments from a binary name.
//
// A process started from an already-renamed binary (the state a previously
// broken update leaves behind: /proc/self/exe -> sing-box-easy.old.old) would
// otherwise keep installing to ever-deeper backup paths. Stripping the suffixes
// makes the next update repair the install instead of corrupting it further.
func canonicalBinaryName(name string) string {
	for strings.HasSuffix(name, backupSuffix) {
		trimmed := strings.TrimSuffix(name, backupSuffix)
		if trimmed == "" {
			break
		}
		name = trimmed
	}
	return name
}

// BinaryName returns the file name the binary is installed under, with any
// ".old" backup suffixes removed.
func BinaryName() string {
	if startupExecutable == "" {
		return defaultBinaryName
	}
	base := canonicalBinaryName(filepath.Base(startupExecutable))
	if base == "" || base == "." || base == string(os.PathSeparator) {
		return defaultBinaryName
	}
	return base
}

// InstallDir returns the directory containing the running executable, with
// symlinks resolved. This is where the binary, dist/ and app.yml live.
func InstallDir() (string, error) {
	if startupExecutable == "" {
		return "", errors.New("could not determine the running executable path")
	}
	resolved, err := filepath.EvalSymlinks(startupExecutable)
	if err != nil {
		// EvalSymlinks fails when the path no longer exists — which is exactly
		// the case for a binary that a previous update moved aside. The
		// directory is still correct, so fall back to the raw path.
		resolved = startupExecutable
	}
	return filepath.Dir(resolved), nil
}

// InstalledBinaryPath returns where the *current* binary lives on disk: the
// install directory joined with the canonical binary name.
//
// This is the path a restart must exec. It deliberately does not use
// os.Executable(), which after an update points at the displaced backup.
func InstalledBinaryPath() (string, error) {
	dir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, BinaryName()), nil
}

// semver is a parsed, comparable version. Missing components default to 0 so
// "v1.2" and "1.2.0" compare equal.
type semver struct {
	major, minor, patch int
	prerelease          string
}

// parseSemver parses tags like "v1.2.3", "1.2.3-rc.1" or "v1.2". It never
// fails; unparseable components become 0, which makes comparison degrade
// gracefully instead of erroring out on an unexpected tag shape.
func parseSemver(v string) semver {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")

	// Strip build metadata ("+build"), it carries no ordering.
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}

	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}

	parts := strings.Split(s, ".")
	out := semver{prerelease: pre}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			n = 0
		}
		switch i {
		case 0:
			out.major = n
		case 1:
			out.minor = n
		case 2:
			out.patch = n
		}
	}
	return out
}

// CompareVersions returns -1 when a < b, 0 when equal, and 1 when a > b.
// A release without a prerelease suffix outranks the same numeric version
// with one (1.2.0 > 1.2.0-rc1), matching semver ordering.
func CompareVersions(a, b string) int {
	x, y := parseSemver(a), parseSemver(b)

	for _, pair := range [][2]int{
		{x.major, y.major},
		{x.minor, y.minor},
		{x.patch, y.patch},
	} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}

	switch {
	case x.prerelease == y.prerelease:
		return 0
	case x.prerelease == "":
		return 1 // release beats prerelease
	case y.prerelease == "":
		return -1
	case x.prerelease < y.prerelease:
		return -1
	default:
		return 1
	}
}

// IsNewer reports whether candidate is strictly newer than current.
// A dev/unstamped current version is treated as "anything is newer" so that
// existing installs without a version stamp can still be upgraded.
func IsNewer(candidate, current string) bool {
	if candidate == "" {
		return false
	}
	if strings.TrimSpace(current) == "" || current == DevVersion {
		return true
	}
	return CompareVersions(candidate, current) > 0
}
