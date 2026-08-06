//go:build windows

package appupdate

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// restartProcess spawns a fresh instance and exits the current one. Windows has
// no execve(2) equivalent, so the PID changes; any supervisor must tolerate that.
func restartProcess() error {
	// See the note in restart_unix.go: os.Executable() points at the displaced
	// backup once installRelease has run, so the install path is used instead.
	exe, err := InstalledBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to locate the installed binary for restart: %w", err)
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start the updated binary: %w", err)
	}

	logger.Info("Started the updated binary; exiting", zap.Int("pid", cmd.Process.Pid))
	logger.Sync()
	os.Exit(0)
	return nil // unreachable
}
