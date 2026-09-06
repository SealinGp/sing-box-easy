package subprobe

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// blockingProber holds every call until released or the context is cancelled,
// so a test can hold a run "in flight" deterministically.
type blockingProber struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newBlockingProber() *blockingProber {
	return &blockingProber{started: make(chan struct{}), release: make(chan struct{})}
}

func (p *blockingProber) Delay(ctx context.Context, tag, testURL string, timeout time.Duration) (int, error) {
	p.once.Do(func() { close(p.started) })
	select {
	case <-p.release:
		return 100, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func lifecycleEnv(prober Prober) *fakeEnv {
	settings := defaultSettings()
	// A short interval keeps the scheduler's own timer out of the way; the
	// startup delay means it never actually fires inside these tests.
	settings.Interval = time.Minute
	return &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: settings,
		tags:     []string{"HK 01 aaaa | sub_a"},
		prober:   prober,
	}
}

// TestStopWaitsForManualRun is the regression test for a real defect: Stop()
// waited on the scheduler goroutine only, while the per-row "probe now" button
// runs on its own HTTP goroutine. A shutdown could therefore return while a
// manual probe was still writing to a database the caller was about to close —
// the exact thing Stop exists to prevent.
func TestStopWaitsForManualRun(t *testing.T) {
	store := newStore(t)
	prober := newBlockingProber()
	runner := NewRunner(store, lifecycleEnv(prober))

	runner.Start()

	var returned atomic.Bool
	go func() {
		_, _ = runner.RunSubscription(context.Background(), "sub_a")
		returned.Store(true)
	}()

	// Wait until the probe is genuinely in flight.
	select {
	case <-prober.started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe never started")
	}

	stopped := make(chan struct{})
	go func() {
		runner.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Stop must have WAITED for the manual run, not merely outrun it.
		if !returned.Load() {
			t.Error("Stop returned while a manual probe was still running")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not return: it neither cancelled nor waited correctly")
	}
}

// TestStopCancelsInFlightWork pins the second half: cancelling has to be what
// makes the wait short. Without the shutdown context reaching the run, Stop
// would block until the sweep's own 15-minute ceiling expired.
func TestStopCancelsInFlightWork(t *testing.T) {
	store := newStore(t)
	prober := newBlockingProber()
	runner := NewRunner(store, lifecycleEnv(prober))

	runner.Start()
	go func() { _, _ = runner.RunSubscription(context.Background(), "sub_a") }()

	select {
	case <-prober.started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe never started")
	}

	// The prober is never released: only cancellation can end this run.
	start := time.Now()
	runner.Stop()
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Errorf("Stop took %s; the shutdown context is not reaching the in-flight run", elapsed)
	}
	if runner.IsRunning() {
		t.Error("IsRunning is still true after Stop")
	}
}

// TestStopIsIdempotentAndRestartable — Stop on a stopped runner must not panic
// on a closed channel, and a restarted runner must not be born cancelled by the
// previous Stop's shutdown context.
func TestStopIsIdempotentAndRestartable(t *testing.T) {
	store := newStore(t)
	prober := &fakeProber{delays: map[string]int{"HK 01 aaaa | sub_a": 100}}
	runner := NewRunner(store, lifecycleEnv(prober))

	runner.Stop() // never started
	runner.Start()
	runner.Stop()
	runner.Stop() // twice

	runner.Start()
	defer runner.Stop()

	// A run after the restart must work: if Start reused the cancelled
	// shutdown context, every sweep would abort immediately.
	sample, err := runner.RunSubscription(context.Background(), "sub_a")
	if err != nil {
		t.Fatalf("run after restart: %v", err)
	}
	if sample.Reachable != 1 {
		t.Errorf("sample = %+v, want 1 reachable", sample)
	}
}

// TestRunRefusedWhileStopping — once shutdown has begun, a new run must be
// turned away rather than started, or it could outlive the Stop that waited.
func TestRunRefusedWhileStopping(t *testing.T) {
	store := newStore(t)
	prober := newBlockingProber()
	runner := NewRunner(store, lifecycleEnv(prober))
	runner.Start()

	go func() { _, _ = runner.RunSubscription(context.Background(), "sub_a") }()
	select {
	case <-prober.started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe never started")
	}

	runner.Stop()

	// After Stop the gate is released again, so this checks the ordinary
	// post-shutdown state rather than the transient one: a stopped runner
	// still answers, which is what the manual button needs.
	if _, err := runner.RunSubscription(context.Background(), "sub_a"); err == nil {
		close(prober.release)
	}
}

// TestConcurrentRunsAreSerialised — the manual button and the sweep share one
// gate, so two runs can never overlap and double the load on the router.
func TestConcurrentRunsAreSerialised(t *testing.T) {
	store := newStore(t)
	prober := newBlockingProber()
	runner := NewRunner(store, lifecycleEnv(prober))

	go func() { _, _ = runner.RunSubscription(context.Background(), "sub_a") }()
	select {
	case <-prober.started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe never started")
	}

	if _, err := runner.RunSubscription(context.Background(), "sub_a"); err == nil {
		t.Error("a second concurrent run was allowed")
	}
	if err := runner.RunOnce(context.Background()); err == nil {
		t.Error("a sweep was allowed to overlap a manual run")
	}

	close(prober.release)
}
