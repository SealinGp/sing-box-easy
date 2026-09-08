package service

// Following the log, rather than asking for it every 1.5 seconds.
//
// THE THING THAT WILL BITE
// ────────────────────────
// Two of the three backends follow by holding a child process open
// (`journalctl -f`, `logread -f`). A browser tab that closes, a laptop that
// sleeps, a reverse proxy that times out — none of those tell us anything
// except that a write eventually fails. If the child is not killed at that
// moment it keeps running, keeps holding a pipe, and the next visit starts
// another one. On a router with 128MB of RAM that is not a slow leak, it is an
// outage.
//
// So every follower here is built around one rule: the goroutine that starts
// the child also owns killing it, and it kills it on EVERY exit path —
// including the ones that look impossible. `startCommand` is the only place a
// child is spawned, and its cleanup is deferred at the spawn site.
//
// The file backend does not spawn anything; it polls with a seek. That is still
// worth streaming: the client holds one connection instead of re-authenticating
// and re-serializing a 300-line window every 1.5s, and the poll interval can be
// far tighter than a client-driven one because there is no request overhead.

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// filePollInterval is how often the file follower checks for growth. Tighter
// than the 1.5s the client used to poll at, because there is no request,
// no auth check and no JSON envelope per check — just an fstat.
const filePollInterval = 400 * time.Millisecond

// keepAliveInterval bounds how long a stream can stay silent. sing-box at
// `level: info` on an idle link logs nothing for minutes, which an intermediate
// proxy reads as a dead connection and drops.
const keepAliveInterval = 20 * time.Second

// FollowEvent is one push from a follower.
//
// Lines and KeepAlive are mutually exclusive: a keep-alive exists precisely
// because there are no lines to send.
type FollowEvent struct {
	Lines     []string
	Cursor    string
	KeepAlive bool
}

// startCommand spawns a following child and streams its stdout as lines.
//
// The returned channel closes when the child exits, the context is cancelled,
// or stdout ends. The child is killed on every one of those paths — see the
// file header for why that is the whole point of this function.
func startCommand(ctx context.Context, name string, args []string, keep func(string) bool) (<-chan FollowEvent, error) {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("pipe %s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	out := make(chan FollowEvent)

	go func() {
		defer close(out)
		// The child dies with this goroutine, no matter how it ends. Kill
		// rather than Signal: journalctl -f ignores a plain SIGTERM when its
		// stdout is a pipe nobody is reading.
		defer func() {
			_ = stdout.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		}()

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			if keep != nil && !keep(line) {
				continue
			}
			select {
			case out <- FollowEvent{Lines: []string{line}}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Cancellation has to be able to interrupt a scanner that is blocked on a
	// read, which closing stdout does and cancelling alone does not.
	go func() {
		<-ctx.Done()
		_ = stdout.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	return out, nil
}

// followFile tails a growing file from its current end.
//
// Handles truncation (log rotation) by detecting a size smaller than the last
// offset and restarting from the new beginning — without that check, a rotated
// log stops producing output entirely and the viewer silently goes dead.
func followFile(ctx context.Context, path string) (<-chan FollowEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("seek %s: %w", path, err)
	}

	out := make(chan FollowEvent)

	go func() {
		defer close(out)
		defer f.Close()

		ticker := time.NewTicker(filePollInterval)
		defer ticker.Stop()

		reader := bufio.NewReaderSize(f, 64*1024)
		var pending strings.Builder

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			info, err := f.Stat()
			if err != nil {
				return
			}
			// Rotated or truncated: everything after the old offset is gone.
			if info.Size() < offset {
				offset = 0
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					return
				}
				reader.Reset(f)
				pending.Reset()
			}
			if info.Size() == offset {
				continue
			}

			var lines []string
			for {
				chunk, err := reader.ReadString('\n')
				if chunk != "" {
					offset += int64(len(chunk))
				}
				if err != nil {
					// A partial final line: hold it until its newline arrives,
					// or the viewer shows half a line and then the other half.
					pending.WriteString(chunk)
					break
				}
				full := pending.String() + strings.TrimRight(chunk, "\r\n")
				pending.Reset()
				if strings.TrimSpace(full) != "" {
					lines = append(lines, full)
				}
			}

			if len(lines) == 0 {
				continue
			}
			select {
			case out <- FollowEvent{Lines: lines}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}
