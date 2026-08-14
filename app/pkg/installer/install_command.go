package installer

import (
	"fmt"
	"runtime"

	"github.com/SealinGp/sing-box-easy/app/pkg/service"
)

// curlOpts are the robust download options used by the script-based install
// path on full Linux distributions.
const curlOpts = "curl --retry 3 --retry-delay 2 --connect-timeout 30 --max-time 300 -fsSL"

// buildInstallCommand returns the shell command used to install sing-box on
// this host. version has already been validated against versionPattern by the
// caller, so it is safe to interpolate.
//
//   - Debian/other Linux: the official install script (curl | sh), which
//     resolves the latest/beta/pinned version itself.
//   - OpenWrt: `opkg install sing-box` from the packages feed (23.05+). The
//     install script does not support opkg, and curl may be absent. A pinned
//     version instead downloads the static release tarball for this
//     architecture with BusyBox wget.
func buildInstallCommand(systemType service.SystemType, version string, beta bool) (string, error) {
	if systemType == service.SystemOpenWRT {
		return buildOpenwrtInstallCommand(version, beta)
	}

	switch {
	case beta:
		return fmt.Sprintf("%s https://sing-box.app/install.sh | sh -s -- --beta", curlOpts), nil
	case version != "":
		return fmt.Sprintf("%s https://sing-box.app/install.sh | sh -s -- --version %s", curlOpts, version), nil
	default:
		return fmt.Sprintf("%s https://sing-box.app/install.sh | sh", curlOpts), nil
	}
}

// buildOpenwrtInstallCommand builds the opkg (default) or static-tarball
// (pinned version) install command for OpenWrt hosts.
func buildOpenwrtInstallCommand(version string, beta bool) (string, error) {
	if beta {
		return "", fmt.Errorf("beta installs are not supported on OpenWrt; " +
			"install a specific version or use `opkg install sing-box`")
	}
	if version == "" {
		return "opkg update && opkg install sing-box", nil
	}

	// Pinned version: the opkg feed only carries one version, so download the
	// official static build for this architecture instead. BusyBox wget is
	// always available on OpenWrt (curl is not).
	arch := singBoxReleaseArch()
	dir := fmt.Sprintf("sing-box-%s-linux-%s", version, arch)
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/v%s/%s.tar.gz", version, dir)
	return fmt.Sprintf(
		"wget -qO /tmp/sing-box.tar.gz %s"+
			" && tar -xzf /tmp/sing-box.tar.gz -C /tmp"+
			" && /etc/init.d/sing-box stop 2>/dev/null"+
			"; install -m 755 /tmp/%s/sing-box /usr/bin/sing-box"+
			" && rm -rf /tmp/sing-box.tar.gz /tmp/%s",
		url, dir, dir), nil
}

// singBoxReleaseArch maps the Go architecture of this binary onto the
// architecture suffix used by official sing-box release assets.
func singBoxReleaseArch() string {
	switch runtime.GOARCH {
	case "arm":
		return "armv7"
	default:
		// amd64, arm64, 386, mips, mipsle, ... match the asset naming as-is.
		return runtime.GOARCH
	}
}
