package autoupdate

import (
	"context"
	"testing"
	"time"
)

func TestStopCancelsAndDrainsTriggeredWork(t *testing.T) {
	started, finished := make(chan struct{}), make(chan struct{})
	s := New(func(ctx context.Context) { close(started); <-ctx.Done(); close(finished) })
	if err := s.Start("0 0 * * *"); err != nil {
		t.Fatal(err)
	}
	s.Trigger()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("job did not start")
	}
	s.Stop()
	select {
	case <-finished:
	default:
		t.Fatal("Stop returned before job finished")
	}
	if s.Running() {
		t.Fatal("still running")
	}
	s.Trigger() // Ignored after shutdown.
	s.Stop()
}
func TestStopCancelsDelayedStartup(t *testing.T) {
	called := make(chan struct{}, 1)
	s := New(func(context.Context) { called <- struct{}{} })
	if err := s.Start("0 0 * * *"); err != nil {
		t.Fatal(err)
	}
	s.Stop()
	select {
	case <-called:
		t.Fatal("startup task ran after stop")
	default:
	}
	if err := s.Start("0 0 * * *"); err != nil {
		t.Fatal(err)
	}
	s.Stop()
}
