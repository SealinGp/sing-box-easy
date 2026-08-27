// Package applog keeps a bounded, in-memory tail of this panel's OWN log, so
// the operator can read what sing-box-easy is doing without shell access.
//
// WHY IN MEMORY, RATHER THAN journalctl/logread LIKE THE sing-box VIEW
// ────────────────────────────────────────────────────────────────────
// The panel writes to stdout and nothing else. Where that lands depends
// entirely on how it was started:
//
//	systemd        journald, under whatever unit name the packager chose
//	procd/OpenWrt  syslog, tagged sing-box-easy
//	manual/dev     the terminal it was launched from — unrecoverable
//
// Reading it back would therefore mean three platform-specific paths, a guess
// at the unit name, and a third of installs where it simply does not work. A
// ring buffer filled at the point of writing works identically everywhere,
// needs no child process (so it cannot leak one), and cannot be defeated by a
// packaging choice.
//
// WHAT IT COSTS, STATED PLAINLY
// ─────────────────────────────
// The buffer is the life of the process. A panel restart — including the
// execve at the end of a self-update — starts it empty, and a panel that
// CRASHED cannot serve the log explaining why, because it is not running to
// serve anything. The UI says so rather than presenting an empty buffer as
// "nothing happened". For post-mortem work, the platform's own log is still
// the answer and always was.
package applog

import (
	"context"
	"sync"
)

// DefaultCapacity is how many lines the ring keeps.
//
// Comfortably above the 500 the viewer renders, so switching to this tab shows
// a full window plus history, while staying small enough (~200KB at typical
// line length) to be unremarkable on a 128MB router.
const DefaultCapacity = 1000

// subscriberBuffer is how many pending batches a subscriber may fall behind by.
//
// A slow client must never block the LOGGER — a log write that waits on an SSE
// socket turns a stalled browser tab into a stalled application. Past this
// depth a subscriber's batches are dropped instead, and it simply misses lines.
const subscriberBuffer = 64

// Ring is a bounded FIFO of log lines with fan-out to live subscribers.
//
// Safe for concurrent use: zap writes from any goroutine, and each SSE stream
// reads from its own.
type Ring struct {
	mu       sync.RWMutex
	buf      []string
	capacity int
	start    int
	count    int

	nextID int
	subs   map[int]chan []string
	// dropped counts lines a subscriber never received because it was too far
	// behind. Reported so a gap in a viewer is explainable rather than spooky.
	dropped uint64
}

func New(capacity int) *Ring {
	if capacity < 1 {
		capacity = DefaultCapacity
	}
	return &Ring{
		buf:      make([]string, capacity),
		capacity: capacity,
		subs:     make(map[int]chan []string),
	}
}

// Append records lines and fans them out. Never blocks on a subscriber.
//
// Everything happens under the write lock, including the sends. That is safe
// precisely BECAUSE the sends are non-blocking (`default:` below) — the lock is
// held for a bounded, allocation-free moment. It is also necessary: an earlier
// version snapshotted the subscriber set, released the lock and then sent, which
// let a subscriber be unsubscribed and its channel closed in the gap. Sending on
// a closed channel panics, so that version would take the whole panel down when
// a browser tab closed at the wrong microsecond.
func (r *Ring) Append(lines ...string) {
	if len(lines) == 0 {
		return
	}

	// Copy once: every subscriber receives the same slice and must not be able
	// to observe a later Append mutating the caller's array.
	batch := make([]string, len(lines))
	copy(batch, lines)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, line := range batch {
		idx := (r.start + r.count) % r.capacity
		if r.count < r.capacity {
			r.buf[idx] = line
			r.count++
			continue
		}
		r.buf[r.start] = line
		r.start = (r.start + 1) % r.capacity
	}

	for _, ch := range r.subs {
		select {
		case ch <- batch:
		default:
			// The subscriber is too far behind. Drop rather than wait: a log
			// write that blocked on an SSE socket would turn a stalled browser
			// tab into a stalled application.
			r.dropped++
		}
	}
}

// Tail returns the most recent n lines, oldest first.
func (r *Ring) Tail(n int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || n > r.count {
		n = r.count
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = r.buf[(r.start+r.count-n+i)%r.capacity]
	}
	return out
}

// Len reports how many lines are currently held.
func (r *Ring) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

// Dropped reports how many lines were lost to slow subscribers.
func (r *Ring) Dropped() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dropped
}

// Subscribe streams lines appended from now on. The channel closes when ctx is
// cancelled.
//
// Backlog is deliberately NOT replayed here: the caller has already fetched a
// window through Tail, and replaying would duplicate every line on screen. This
// mirrors the sing-box feed, where the follower starts at the end of the file
// for the same reason.
func (r *Ring) Subscribe(ctx context.Context) <-chan []string {
	ch := make(chan []string, subscriberBuffer)

	r.mu.Lock()
	id := r.nextID
	r.nextID++
	r.subs[id] = ch
	r.mu.Unlock()

	// One goroutine per subscriber, whose only job is teardown. Closing under
	// the same exclusive lock Append sends under is what makes the close safe:
	// the two can never interleave.
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		delete(r.subs, id)
		close(ch)
		r.mu.Unlock()
	}()

	return ch
}
