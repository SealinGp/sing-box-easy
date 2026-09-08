//go:build unix

package sysinfo

import "syscall"

// statfs asks the kernel for the usage of the filesystem containing path.
//
// Bsize is int64 on Linux and uint32 on Darwin, so every field is widened to
// uint64 here and the rest of the package stays platform-agnostic.
func statfs(path string) (fsStat, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return fsStat{}, err
	}

	blockSize := uint64(fs.Bsize)

	return fsStat{
		total: uint64(fs.Blocks) * blockSize,
		// Bfree counts blocks free to root; the difference from Blocks is what
		// is genuinely occupied, which is what df reports as "Used".
		used: (uint64(fs.Blocks) - uint64(fs.Bfree)) * blockSize,
		// Bavail excludes the root reserve — the honest "can I still write".
		free: uint64(fs.Bavail) * blockSize,
	}, nil
}
