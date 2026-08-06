//go:build !windows

package appupdate

import (
	"fmt"
	"os"
	"syscall"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// restartProcess replaces the running process image with the freshly installed
// binary via execve(2).
//
// This is preferred over `systemctl restart`: the PID is preserved, so systemd
// (which would otherwise kill us mid-command) sees an ordinary long-running
// service, and it works identically under the nohup fallback. Go marks every
// socket O_CLOEXEC, so the HTTP listener is released automatically and the new
// image rebinds it.
//
// On success this function never returns.
func restartProcess() error {
	// Deliberately NOT os.Executable(). By this point installRelease has moved
	// the running binary aside to "<name>.old", and on Linux os.Executable()
	// reads /proc/self/exe, which follows that rename — so exec'ing it would
	// re-launch the OLD binary and the update would silently have no effect.
	// InstalledBinaryPath() names the freshly installed file instead.
	target, err := InstalledBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to locate the installed binary for restart: %w", err)
	}

	// Fail loudly rather than exec'ing something that is missing or not a file:
	// a failed restart leaves the current process serving, which is recoverable.
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("installed binary %s is not accessible: %w", target, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("installed binary %s is not a regular file", target)
	}

	// argv[0] is the target path so the new image reports a sane name; the
	// original flags (e.g. "-c /etc/sing-box/app.yml") are preserved.
	argv := append([]string{target}, os.Args[1:]...)

	logger.Info("Re-executing after update",
		zap.String("executable", target),
		zap.Strings("args", argv),
	)
	logger.Sync()

	if err := syscall.Exec(target, argv, os.Environ()); err != nil {
		return fmt.Errorf("failed to re-execute %s: %w", target, err)
	}
	return nil // unreachable
}
