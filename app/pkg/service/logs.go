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
	LogSourceSyslog   = "syslog"
	LogSourceFile     = "file"
	LogSourceNone     = "none"
)

// LogChunk is a bounded slice of recent log lines plus an opaque cursor for
// incremental polling. Only the journald source supports cursors; other
// sources return an empty cursor and always serve the full tail window. When
// Source is "none" the lines are empty and the UI should explain that live
// logs require an init-system log (journald/syslog) or a configured
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

// tailFile reads the last `lines` lines of the given log file. Returns
// Source="none" when the file is missing or unreadable so the viewer shows
// guidance instead of an error toast.
func tailFile(path string, lines int) (LogChunk, error) {
	f, err := os.Open(path)
	if err != nil {
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

// tailLogread reads the syslog ring buffer via BusyBox `logread` (OpenWrt)
// and keeps only sing-box lines. logread prints the whole buffer, so the
// filter and tail window are applied in-process rather than relying on
// version-specific logread flags.
func tailLogread(lines int) (LogChunk, error) {
	out, err := exec.Command("logread").Output()
	if err != nil {
		return LogChunk{}, fmt.Errorf("failed to read syslog via logread: %w", err)
	}
	return LogChunk{
		Lines:  filterSyslogLines(string(out), lines),
		Source: LogSourceSyslog,
	}, nil
}

// filterSyslogLines keeps the last `lines` syslog lines mentioning sing-box.
// Kept pure (no exec) so it is unit-testable.
func filterSyslogLines(out string, lines int) []string {
	tail := newRingBuffer(lines)
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "sing-box") {
			tail.push(line)
		}
	}
	return tail.slice()
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
