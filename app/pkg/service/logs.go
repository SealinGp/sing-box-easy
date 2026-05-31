package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Log tail bounds. The viewer keeps only a recent window, so callers never need
// (or get) more than maxLogLines per request.
const (
	defaultLogLines = 300
	maxLogLines     = 1000
)

// Log sources reported back to the UI so it can explain an empty view.
const (
	LogSourceJournald = "journald"
	LogSourceFile     = "file"
	LogSourceNone     = "none"
)

// LogChunk is a bounded slice of recent log lines plus an opaque cursor for
// incremental polling. When Source is "none" the lines are empty and the UI
// should explain that live logs require systemd (journald) or a configured
// log.output file.
type LogChunk struct {
	Lines  []string `json:"lines"`
	Cursor string   `json:"cursor"`
	Source string   `json:"source"`
}

// clampLines normalizes a requested line count into [1, maxLogLines], applying
// the default when the caller passes 0 or a negative value.
func clampLines(n int) int {
	if n <= 0 {
		return defaultLogLines
	}
	if n > maxLogLines {
		return maxLogLines
	}
	return n
}

// TailLogs returns the most recent sing-box log lines.
//
//   - Under systemd it reads journald via journalctl. Passing a non-empty
//     afterCursor returns only entries newer than that cursor (incremental
//     polling); the returned Cursor should be fed back on the next call.
//   - Without systemd it best-effort tails the configured log.output file.
//   - When neither source is available it returns Source="none" with no error,
//     so the UI can render an explanation rather than a failure.
func (c *Controller) TailLogs(lines int, afterCursor string) (LogChunk, error) {
	lines = clampLines(lines)

	if c.useSystemd {
		return c.tailJournald(lines, afterCursor)
	}
	return c.tailFile(lines)
}

// tailJournald shells out to journalctl. `--show-cursor` appends a trailing
// "-- cursor: <c>" line we parse out; `-n` bounds the result even in the
// incremental (--after-cursor) case so a chatty proxy can't flood one response.
func (c *Controller) tailJournald(lines int, afterCursor string) (LogChunk, error) {
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

// splitJournalOutput separates real log lines from journalctl's noise: the
// trailing "-- cursor: <c>" marker (captured as the cursor) and the
// "-- No entries --" / "-- Logs begin at ... --" banners (dropped).
func splitJournalOutput(out string) ([]string, string) {
	var lines []string
	var cursor string
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "-- cursor:"); ok {
			cursor = strings.TrimSpace(rest)
			continue
		}
		// Drop journald's own "-- ... --" banner lines.
		if strings.HasPrefix(trimmed, "-- ") && strings.HasSuffix(trimmed, " --") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, cursor
}

// tailFile reads the last `lines` lines of the configured log.output file. Used
// only when sing-box is not managed by systemd. Returns Source="none" when no
// usable file is configured so the UI can guide the user.
func (c *Controller) tailFile(lines int) (LogChunk, error) {
	path := c.logOutputPath()
	if path == "" {
		return LogChunk{Lines: []string{}, Source: LogSourceNone}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		// A missing/unreadable file is not fatal: report "none" so the viewer
		// shows guidance instead of an error toast.
		return LogChunk{Lines: []string{}, Source: LogSourceNone}, nil
	}
	defer f.Close()

	tail := newRingBuffer(lines)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		tail.push(scanner.Text())
	}
	return LogChunk{Lines: tail.slice(), Source: LogSourceFile}, nil
}

// logOutputPath returns the configured sing-box log.output file path, or "" when
// logging goes to stdout (no file to tail).
func (c *Controller) logOutputPath() string {
	cfg, err := c.configManager.GetConfig()
	if err != nil || cfg.Log == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Log.Output)
}

// ringBuffer keeps only the most recent n strings pushed into it.
type ringBuffer struct {
	buf   []string
	size  int
	start int
	count int
}

func newRingBuffer(n int) *ringBuffer {
	if n < 1 {
		n = 1
	}
	return &ringBuffer{buf: make([]string, n), size: n}
}

func (r *ringBuffer) push(s string) {
	idx := (r.start + r.count) % r.size
	if r.count < r.size {
		r.buf[idx] = s
		r.count++
		return
	}
	// Full: overwrite oldest and advance start.
	r.buf[r.start] = s
	r.start = (r.start + 1) % r.size
}

func (r *ringBuffer) slice() []string {
	out := make([]string, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.start+i)%r.size]
	}
	return out
}
