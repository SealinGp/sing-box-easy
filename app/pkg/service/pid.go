package service

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Signal flags passed to the `kill` command when terminating a sing-box
// process that this application does not own.
const (
	signalTerm = "-TERM" // graceful shutdown
	signalKill = "-KILL" // force kill
)

// Timing for waiting on a process to actually exit after a stop signal.
const (
	stopGracePeriod  = 5 * time.Second
	stopPollInterval = 100 * time.Millisecond
)

// Timing for confirming a process actually appeared after a start request.
// An init system returning 0 only means it accepted the request, not that
// the service is up.
const (
	startGracePeriod  = 3 * time.Second
	startPollInterval = 100 * time.Millisecond
)

// lookupPID returns the PID of the sing-box process as a string, or "" if not
// running. It prefers pidof (native BusyBox applet on OpenWrt) and falls back
// to pgrep and a POSIX ps pipeline on other systems.
func lookupPID(systemType SystemType) (string, error) {
	switch systemType {
	case SystemOpenWRT:
		// Use pidof for OpenWrt (a native BusyBox applet).
		output, err := exec.Command("pidof", "sing-box").Output()
		if err != nil {
			// Process not found
			return "", nil
		}
		// pidof might return multiple PIDs, take the first one
		pids := strings.Fields(strings.TrimSpace(string(output)))
		if len(pids) > 0 {
			return pids[0], nil
		}
		return "", nil

	default:
		// Try pgrep first (most efficient)
		output, err := exec.Command("pgrep", "-x", "sing-box").Output()
		if err == nil {
			pid := strings.TrimSpace(string(output))
			if pid != "" {
				return strings.Split(pid, "\n")[0], nil // first PID if multiple
			}
		}

		// Fallback to pidof, present on BusyBox and most Linux distros.
		output, err = exec.Command("pidof", "sing-box").Output()
		if err == nil {
			pids := strings.Fields(strings.TrimSpace(string(output)))
			if len(pids) > 0 {
				return pids[0], nil
			}
		}

		return "", nil
	}
}

// lookupRunningPID resolves both the running state and the numeric PID in one
// pass, shared by the non-systemd backends.
func lookupRunningPID(systemType SystemType) (bool, int, error) {
	pidStr, err := lookupPID(systemType)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check service status: %w", err)
	}
	if pidStr == "" {
		return false, 0, nil
	}
	pid, _ := strconv.Atoi(pidStr)
	return true, pid, nil
}

// signalRunningPID looks up the running sing-box PID and sends it the given
// signal via the `kill` command. It is used to control processes this
// application does not own. A nil error is returned when no process is
// running.
func signalRunningPID(systemType SystemType, sig string) error {
	pid, err := lookupPID(systemType)
	if err != nil {
		return fmt.Errorf("failed to get service PID: %w", err)
	}
	if pid == "" {
		return nil
	}

	// Validate pid is a positive integer before passing it to exec.
	// pid originates from `pgrep`/`pidof` output, which is normally safe,
	// but never trust an external string flowing into an exec argument list.
	pidNum, err := strconv.Atoi(pid)
	if err != nil || pidNum <= 0 {
		return fmt.Errorf("invalid pid from process lookup: %q", pid)
	}

	if err := exec.Command("kill", sig, strconv.Itoa(pidNum)).Run(); err != nil {
		return fmt.Errorf("failed to send %s to pid %d: %w", sig, pidNum, err)
	}
	return nil
}

// waitForServiceStart polls check until it reports the service running, the
// check itself fails, or the timeout elapses. It is the start-side
// counterpart to waitForPIDExit.
//
// The probe is injected rather than hardcoded to lookupPID so backends can
// supply their own notion of "running" and tests can drive it without
// touching the process table.
func waitForServiceStart(check func() (bool, int, error), timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		running, _, err := check()
		if err != nil {
			return err
		}
		if running {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("service did not start within %s", timeout)
		}
		time.Sleep(interval)
	}
}

// waitForPIDExit polls the process table until sing-box has exited or the
// timeout elapses.
func waitForPIDExit(systemType SystemType, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := lookupPID(systemType)
		if err != nil {
			return err
		}
		if pid == "" {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("process still running after %s", timeout)
		}
		time.Sleep(stopPollInterval)
	}
}
