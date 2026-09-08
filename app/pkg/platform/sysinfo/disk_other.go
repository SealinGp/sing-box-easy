//go:build !unix

package sysinfo

import "errors"

// statfs has no portable equivalent outside unix. Reporting an error here makes
// CollectDisks return no entries, and the About card simply omits the storage
// row — the panel only ships for Linux hosts.
func statfs(string) (fsStat, error) {
	return fsStat{}, errors.New("sysinfo: disk usage is not supported on this platform")
}
