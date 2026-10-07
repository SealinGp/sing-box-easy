//go:build windows

package appupdate

import "errors"

// The package install path is OpenWrt-only; these exist so the package builds.

func launchDetached(string, string) error {
	return errors.New("package install is not supported on windows")
}

func freeBytes(string) (uint64, error) {
	return 0, errors.New("free space lookup is not supported on windows")
}
