//go:build !unix

package database

// isReadOnly cannot be determined without statfs, so the read-only hint is
// simply never added on these platforms.
func isReadOnly(string) bool { return false }
