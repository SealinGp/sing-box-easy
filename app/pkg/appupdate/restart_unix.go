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
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate the executable for restart: %w", err)
	}

	argv := os.Args
	if len(argv) == 0 {
		argv = []string{exe}
	}

	logger.Info("Re-executing after update",
		zap.String("executable", exe),
		zap.Strings("args", argv),
	)
	logger.Sync()

	if err := syscall.Exec(exe, argv, os.Environ()); err != nil {
		return fmt.Errorf("failed to re-execute %s: %w", exe, err)
	}
	return nil // unreachable
}
