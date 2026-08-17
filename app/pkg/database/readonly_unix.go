//go:build unix

package database

import "syscall"

// roMountFlag is the read-only bit in the statfs mount flags: ST_RDONLY on
// Linux, MNT_RDONLY on Darwin — both are 1. Left untyped so it composes with
// Flags regardless of its platform-dependent integer type.
const roMountFlag = 1

// isReadOnly reports whether the filesystem at dir rejects writes. A failure to
// determine this is reported as "not read-only": the caller only uses it to add
// detail to an error that is already being returned.
func isReadOnly(dir string) bool {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(dir, &fs); err != nil {
		return false
	}
	return fs.Flags&roMountFlag != 0
}
