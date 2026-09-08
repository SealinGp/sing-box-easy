package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// collect drains a follower for up to `wait`, returning everything it pushed.
func collect(t *testing.T, events <-chan FollowEvent, want int, wait time.Duration) []string {
	t.Helper()
	var lines []string
	deadline := time.After(wait)
	for len(lines) < want {
		select {
		case event, ok := <-events:
			if !ok {
				return lines
			}
			lines = append(lines, event.Lines...)
		case <-deadline:
			return lines
		}
	}
	return lines
}

func writeLines(t *testing.T, path string, text string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		t.Fatalf("append: %v", err)
	}
}

func TestFollowFileEmitsAppendedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sing-box.log")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := followFile(ctx, path)
	if err != nil {
		t.Fatalf("followFile: %v", err)
	}

	writeLines(t, path, "first\nsecond\n")

	got := collect(t, events, 2, 3*time.Second)
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("got %q, want [first second]", got)
	}
}

// The follower starts at the END of the file: the backlog came from the seed
// request, and replaying it would duplicate every line already on screen.
func TestFollowFileSkipsExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sing-box.log")
	if err := os.WriteFile(path, []byte("backlog a\nbacklog b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := followFile(ctx, path)
	if err != nil {
		t.Fatalf("followFile: %v", err)
	}

	writeLines(t, path, "new\n")

	got := collect(t, events, 1, 3*time.Second)
	if len(got) != 1 || got[0] != "new" {
		t.Fatalf("got %q, want [new] — the backlog must not be replayed", got)
	}
}

// A line written without its newline yet must not be emitted as a half line.
// Without the `pending` buffer the viewer shows the fragment, then shows the
// remainder as a second line, and a fatal message arrives split in two.
func TestFollowFileHoldsPartialLineUntilNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sing-box.log")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := followFile(ctx, path)
	if err != nil {
		t.Fatalf("followFile: %v", err)
	}

	writeLines(t, path, "FATAL start ser")
	if got := collect(t, events, 1, 900*time.Millisecond); len(got) != 0 {
		t.Fatalf("emitted %q before the line was complete", got)
	}

	writeLines(t, path, "vice: bad config\n")
	got := collect(t, events, 1, 3*time.Second)
	if len(got) != 1 || got[0] != "FATAL start service: bad config" {
		t.Fatalf("got %q, want the rejoined line", got)
	}
}

// Log rotation truncates the file. Without the size check the follower sits at
// an offset past the new end and never emits again — the viewer goes silent and
// looks like a sing-box that stopped logging.
func TestFollowFileRecoversFromTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sing-box.log")
	if err := os.WriteFile(path, []byte("before rotation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := followFile(ctx, path)
	if err != nil {
		t.Fatalf("followFile: %v", err)
	}

	if err := os.WriteFile(path, []byte("after rotation\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := collect(t, events, 1, 3*time.Second)
	if len(got) != 1 || got[0] != "after rotation" {
		t.Fatalf("got %q, want [after rotation]", got)
	}
}

// Cancelling must close the channel. This is the contract the SSE handler
// relies on to know the follower is done, and — for the exec-based backends —
// the path on which the child process is killed.
func TestFollowFileClosesOnCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sing-box.log")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	events, err := followFile(ctx, path)
	if err != nil {
		t.Fatalf("followFile: %v", err)
	}

	cancel()

	select {
	case _, ok := <-events:
		if ok {
			// A buffered line may arrive first; drain once more.
			select {
			case _, ok := <-events:
				if ok {
					t.Fatal("channel still open after cancel")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("channel not closed after cancel")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

// startCommand must not leave the child running once the context is cancelled.
// This is the leak the whole design is arranged around: one orphaned
// `journalctl -f` per closed browser tab.
func TestStartCommandKillsChildOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// `sleep` is the portable stand-in for a follower that never exits.
	events, err := startCommand(ctx, "sleep", []string{"60"}, nil)
	if err != nil {
		t.Skipf("sleep unavailable: %v", err)
	}

	cancel()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("unexpected output from sleep")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("follower did not stop after cancel — the child is orphaned")
	}
}

func TestStartCommandAppliesLineFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := startCommand(ctx, "printf", []string{`keep\ndrop\nkeep\n`}, func(line string) bool {
		return line == "keep"
	})
	if err != nil {
		t.Skipf("printf unavailable: %v", err)
	}

	got := collect(t, events, 2, 3*time.Second)
	for _, line := range got {
		if line != "keep" {
			t.Fatalf("filter let %q through", line)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d lines, want 2", len(got))
	}
}
