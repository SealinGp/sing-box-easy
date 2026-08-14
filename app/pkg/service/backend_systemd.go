package service

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

// singBoxServiceName is the systemd unit name used by the official sing-box
// Debian/Linux installation.
const singBoxServiceName = "sing-box.service"

// systemdBackend manages sing-box through its systemd unit. All lifecycle
// operations delegate to systemctl; logs come from journald.
type systemdBackend struct{}

func (b *systemdBackend) Kind() string { return BackendSystemd }

// systemctl runs `systemctl <args...> sing-box.service` and wraps failures
// with the captured output for easier diagnosis.
func (b *systemdBackend) systemctl(args ...string) error {
	full := append(append([]string{}, args...), singBoxServiceName)
	out, err := exec.Command("systemctl", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *systemdBackend) Start() error {
	if err := b.systemctl("start"); err != nil {
		return err
	}
	logger.Info("Service started via systemd")
	return nil
}

// Stop delegates to `systemctl stop`, which is idempotent and blocks until
// the unit has stopped.
func (b *systemdBackend) Stop() error {
	if err := b.systemctl("stop"); err != nil {
		return err
	}
	logger.Info("Service stopped via systemd")
	return nil
}

func (b *systemdBackend) ForceStop() error {
	if err := b.systemctl("kill", "--signal=SIGKILL"); err != nil {
		return err
	}
	logger.Info("Service force stopped via systemd")
	return nil
}

// Restart delegates to `systemctl restart`, which starts the unit even if it
// was stopped.
func (b *systemdBackend) Restart() error {
	if err := b.systemctl("restart"); err != nil {
		return err
	}
	logger.Info("Service restarted via systemd")
	return nil
}

// Reload uses `reload-or-restart`: it reloads if the unit defines ExecReload,
// otherwise it performs a full restart — mirroring the SIGHUP-with-restart
// fallback of the direct process backend.
func (b *systemdBackend) Reload() error {
	if err := b.systemctl("reload-or-restart"); err != nil {
		return err
	}
	logger.Info("Service reloaded via systemd")
	return nil
}

// ActiveAndPID parses a single `systemctl show` call for the unit's active
// state and main pid.
func (b *systemdBackend) ActiveAndPID() (bool, int, error) {
	out, _ := exec.Command("systemctl", "show", singBoxServiceName,
		"--property=ActiveState", "--property=MainPID").Output()
	active, pid := parseActiveAndPID(string(out))
	return active, pid, nil
}

// parseActiveAndPID extracts ActiveState/MainPID from `systemctl show` output.
// Kept pure (no exec) so it is unit-testable. A non-active unit always reports
// pid 0 regardless of the MainPID line.
func parseActiveAndPID(out string) (bool, int) {
	var active bool
	var pid int
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "ActiveState":
			active = val == "active"
		case "MainPID":
			pid, _ = strconv.Atoi(val)
		}
	}
	if !active {
		return false, 0
	}
	return true, pid
}

// TailLogs shells out to journalctl. `--show-cursor` appends a trailing
// "-- cursor: <c>" line we parse out; `-n` bounds the result even in the
// incremental (--after-cursor) case so a chatty proxy can't flood one
// response.
func (b *systemdBackend) TailLogs(lines int, afterCursor string) (LogChunk, error) {
	args := []string{
		"-u", singBoxServiceName,
		"--no-pager",
		"-o", "short-iso",
		"--show-cursor",
		"-n", fmt.Sprintf("%d", lines),
	}
	if afterCursor != "" {
		args = append(args, "--after-cursor", afterCursor)
	}

	out, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		return LogChunk{}, fmt.Errorf("failed to read journald logs: %w", err)
	}

	logLines, cursor := splitJournalOutput(string(out))
	return LogChunk{Lines: logLines, Cursor: cursor, Source: LogSourceJournald}, nil
}
