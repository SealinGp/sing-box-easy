//go:build !windows

package appupdate

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// launchDetached starts a shell script that must outlive this process.
//
// Three things make it independent, and each one is load-bearing:
//   - Setsid: its own session and process group, so nothing aimed at this
//     process's group reaches it.
//   - Its own descriptors: stdin from /dev/null, output to a file. Inheriting
//     this process's stdout would hand it a procd pipe that breaks the moment
//     procd frees the stopped instance.
//   - Release, never Wait: nobody is left to reap it, and init adopts it.
//
// The script leaves procd's cgroup itself — that cannot be done from here
// before the exec.
func launchDetached(scriptPath, logPath string) error {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open helper log: %w", err)
	}
	defer logFile.Close()

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	command := exec.Command("/bin/sh", scriptPath)
	command.Dir = "/"
	command.Stdin = devNull
	command.Stdout = logFile
	command.Stderr = logFile
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}

// freeBytes reports the space available to root on the filesystem holding path.
func freeBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
