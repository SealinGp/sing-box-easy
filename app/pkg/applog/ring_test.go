package applog

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTailReturnsOldestFirst(t *testing.T) {
	r := New(10)
	r.Append("a", "b", "c")

	got := r.Tail(10)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestRingDropsOldestPastCapacity(t *testing.T) {
	r := New(3)
	r.Append("1", "2", "3", "4", "5")

	got := r.Tail(10)
	if len(got) != 3 || got[0] != "3" || got[2] != "5" {
		t.Fatalf("got %q, want [3 4 5]", got)
	}
}

func TestTailBoundedByRequest(t *testing.T) {
	r := New(10)
	r.Append("1", "2", "3", "4")

	got := r.Tail(2)
	if len(got) != 2 || got[0] != "3" || got[1] != "4" {
		t.Fatalf("got %q, want [3 4]", got)
	}
}

func TestSubscribeReceivesNewLines(t *testing.T) {
	r := New(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := r.Subscribe(ctx)
	r.Append("hello")

	select {
	case batch := <-sub:
		if len(batch) != 1 || batch[0] != "hello" {
			t.Fatalf("got %q, want [hello]", batch)
		}
	case <-time.After(time.Second):
		t.Fatal("no batch delivered")
	}
}

// The backlog belongs to Tail, not to Subscribe. Replaying it here would
// duplicate on screen every line the client already fetched.
func TestSubscribeDoesNotReplayBacklog(t *testing.T) {
	r := New(10)
	r.Append("old one", "old two")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := r.Subscribe(ctx)

	select {
	case batch := <-sub:
		t.Fatalf("replayed backlog: %q", batch)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSubscribeClosesOnCancel(t *testing.T) {
	r := New(10)
	ctx, cancel := context.WithCancel(context.Background())
	sub := r.Subscribe(ctx)

	cancel()

	select {
	case _, ok := <-sub:
		if ok {
			t.Fatal("channel yielded a value instead of closing")
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

// A cancelled subscriber must stop costing anything. Left registered, the
// logger would fan out to a channel nobody reads for the life of the process.
func TestCancelledSubscriberIsUnregistered(t *testing.T) {
	r := New(10)
	ctx, cancel := context.WithCancel(context.Background())
	_ = r.Subscribe(ctx)

	cancel()

	deadline := time.Now().Add(time.Second)
	for {
		r.mu.RLock()
		n := len(r.subs)
		r.mu.RUnlock()
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("subscriber still registered after cancel (%d left)", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A stalled reader must never stall the logger. This is the property that keeps
// a wedged browser tab from wedging the application.
func TestSlowSubscriberIsDroppedNotBlocking(t *testing.T) {
	r := New(10_000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = r.Subscribe(ctx) // never read from

	done := make(chan struct{})
	go func() {
		for i := 0; i < subscriberBuffer*4; i++ {
			r.Append(fmt.Sprintf("line %d", i))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Append blocked on a subscriber that stopped reading")
	}

	if r.Dropped() == 0 {
		t.Fatal("expected drops to be counted for the stalled subscriber")
	}
	// The buffer itself must be unaffected: dropping is about delivery, not
	// about recording.
	if r.Len() != subscriberBuffer*4 {
		t.Fatalf("ring holds %d lines, want %d", r.Len(), subscriberBuffer*4)
	}
}

// The race this test exists for: an earlier implementation released the lock
// between snapshotting subscribers and sending to them, so a subscriber
// cancelled in that window had its channel closed under a live sender. Sending
// on a closed channel panics — this would have taken the whole panel down when
// a tab closed at the wrong moment. Run with -race.
func TestConcurrentSubscribeAppendCancel(t *testing.T) {
	r := New(256)
	var wg sync.WaitGroup

	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			r.Append(fmt.Sprintf("line %d", i))
		}
	}()

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			sub := r.Subscribe(ctx)
			// Read a little, then leave abruptly — the browser-tab pattern.
			select {
			case <-sub:
			case <-time.After(20 * time.Millisecond):
			}
			cancel()
		}()
	}

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestSinkSplitsMultiLineEntries(t *testing.T) {
	r := New(10)
	s := sink{ring: r}

	// What zap's development encoder produces for an entry with a stack trace.
	if _, err := s.Write([]byte("ERROR failed\ngoroutine 1:\n\tmain.go:12\n")); err != nil {
		t.Fatal(err)
	}

	got := r.Tail(10)
	if len(got) != 3 {
		t.Fatalf("got %d lines %q, want 3 — a stack trace stored whole renders as one unbreakable row", len(got), got)
	}
	if got[0] != "ERROR failed" {
		t.Fatalf("first line = %q", got[0])
	}
}

func TestSinkIgnoresEmptyWrites(t *testing.T) {
	r := New(10)
	s := sink{ring: r}

	if _, err := s.Write([]byte("\n")); err != nil {
		t.Fatal(err)
	}
	if r.Len() != 0 {
		t.Fatalf("ring holds %d lines, want 0", r.Len())
	}
}
