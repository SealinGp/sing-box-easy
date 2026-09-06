package subprobe

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// ErrNoSuchTarget is returned by RunSubscription when the id names no probeable
// subscription — it does not exist, or probing is turned off for it. Reported
// rather than swallowed: a "probe now" button that quietly does nothing and
// then says it succeeded is worse than one that errors.
var ErrNoSuchTarget = errors.New("subscription is not configured for probing")

// ErrNoOwnedNodes is returned when a subscription has probing enabled but no
// outbound in the config carries its ownership suffix — its nodes have not
// been imported yet.
var ErrNoOwnedNodes = errors.New("no outbounds belong to this subscription")

// ErrNoNodesTestable is returned when every one of a subscription's nodes was
// present in the config but absent from the RUNNING sing-box.
//
// A distinct error from ErrNoOwnedNodes because the two have different fixes
// and look identical from the outside: the first means "refresh the
// subscription", the second means "apply/restart the config". Collapsing them
// (as an earlier version of this file did, reporting both as "no outbounds
// belong") sends an operator looking for a subscription problem that is really
// a config-not-applied problem.
var ErrNoNodesTestable = errors.New("none of this subscription's nodes are in the running sing-box")

// Target is one subscription to probe.
type Target struct {
	SubID string
	// URL is the already-normalized https test target for this subscription.
	URL string
}

// Prober runs one URL test through one outbound. Satisfied by *clashapi.Client.
type Prober interface {
	Delay(ctx context.Context, tag, testURL string, timeout time.Duration) (int, error)
}

// Environment is everything the runner reads that can change between runs.
//
// An interface rather than concrete managers so the runner is testable without
// a sing-box, a config file or a database, and so the config-shaped
// dependencies stay resolved fresh per run: the operator can add a
// subscription, edit the interval or restart sing-box between two ticks, and
// the next tick must see all three.
type Environment interface {
	// Targets returns the subscriptions with probing enabled.
	Targets() ([]Target, error)
	// OutboundTags returns every outbound tag in the config on disk. Ownership
	// is decided from these, so a node the operator has not applied yet is
	// still counted as belonging to its subscription — and then reported as
	// untestable when the running sing-box does not have it.
	OutboundTags() ([]string, error)
	// Prober builds a prober against the running sing-box. Returns
	// clashapi.ErrDisabled when there is no controller to ask.
	Prober() (Prober, error)
	// Settings returns the current, already-normalized knobs.
	Settings() Settings
}

// Snapshot is the per-node detail of the most recent run for one subscription.
// Kept in memory only: the aggregate is the thing worth keeping over time, and
// per-node history would be ~60x the rows to answer a question ("which node is
// down?") that is only ever asked about right now.
type Snapshot struct {
	At      time.Time    `json:"at"`
	Sample  Sample       `json:"sample"`
	Results []NodeResult `json:"results"`
}

// Runner owns the probe schedule.
type Runner struct {
	store *StoreXORM
	env   Environment

	mu       sync.RWMutex
	running  bool
	inFlight bool
	// stopping blocks new runs from starting once Stop has begun. It is what
	// makes inFlightWG safe: an acquire either takes the lock before Stop and
	// has already called Add, or takes it after and is refused — so Add is
	// never called concurrently with Wait on a zero counter.
	stopping  bool
	lastRun   time.Time
	lastError string
	snapshots map[string]*Snapshot

	// shutdown is cancelled by Stop, aborting whatever is in flight — a
	// scheduled sweep OR an operator-triggered single probe. Without it a
	// graceful shutdown would have to wait out the sweep's 15-minute ceiling.
	shutdown  context.Context
	cancelAll context.CancelFunc
	// inFlightWG counts runs, not loop iterations. Stop waits on it because a
	// manual probe runs on its own HTTP goroutine, NOT inside loop() — so
	// waiting only for loop() would return while a probe was still writing.
	inFlightWG sync.WaitGroup

	stop chan struct{}
	// done is closed when the loop goroutine has exited, so Stop can be
	// synchronous — an asynchronous Stop makes tests flaky and lets a sweep
	// write to a database the caller is closing.
	done chan struct{}
}

// NewRunner builds a runner. Nothing is scheduled until Start.
func NewRunner(store *StoreXORM, env Environment) *Runner {
	shutdown, cancelAll := context.WithCancel(context.Background())
	return &Runner{
		store:     store,
		env:       env,
		snapshots: make(map[string]*Snapshot),
		shutdown:  shutdown,
		cancelAll: cancelAll,
	}
}

// Start begins the periodic sweep.
//
// The first sweep is delayed: sing-box has usually just been started alongside
// the panel, and probing every node of every subscription while it is still
// coming up measures the startup, not the provider.
func (r *Runner) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stop = make(chan struct{})
	r.done = make(chan struct{})
	// A previous Stop cancelled the old one; a restarted runner whose context
	// is already cancelled would abort every sweep immediately.
	r.shutdown, r.cancelAll = context.WithCancel(context.Background())
	stop, done, shutdown := r.stop, r.done, r.shutdown
	interval := r.env.Settings().Normalize().Interval
	r.mu.Unlock()

	logger.Info("Subscription prober started", zap.Duration("interval", interval))
	go r.loop(stop, done, shutdown)
}

// Stop halts the scheduler and waits for any run in flight to finish.
//
// "Any run", not "the scheduled sweep": a manual probe (the per-row button)
// executes on its own HTTP goroutine outside loop(), so waiting only for loop()
// to return would let Stop succeed while a probe was still writing to a
// database the caller is about to close — which is the exact thing this method
// exists to prevent.
//
// The shutdown context is cancelled FIRST so the wait is short. Without it, a
// sweep that had just started would hold Stop for up to its 15-minute ceiling.
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.stopping = true
	close(r.stop)
	r.cancelAll()
	done := r.done
	r.mu.Unlock()

	<-done
	r.inFlightWG.Wait()

	r.mu.Lock()
	r.stopping = false
	r.mu.Unlock()

	logger.Info("Subscription prober stopped")
}

// IsRunning reports whether the sweep is scheduled.
func (r *Runner) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// startupDelay keeps the first sweep off a just-booted sing-box.
const startupDelay = 45 * time.Second

// loop drives the sweeps.
//
// The interval is re-read after every sweep rather than captured once, so an
// edit in Settings takes effect on the next tick. A plain time.Timer is used
// instead of a Ticker for the same reason: a Ticker's period is fixed at
// construction, and re-creating one per iteration is the same code with an
// extra object.
func (r *Runner) loop(stop <-chan struct{}, done chan<- struct{}, shutdown context.Context) {
	defer close(done)

	timer := time.NewTimer(startupDelay)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
		}

		// Bound the sweep so a hung controller cannot hold the loop forever.
		// The ceiling is generous: 233 nodes at 8-way concurrency and a 5s
		// timeout is ~2.5 minutes if every single one times out.
		// Derived from shutdown, not Background: Stop must be able to abort a
		// sweep that has only just begun rather than wait out this ceiling.
		ctx, cancel := context.WithTimeout(shutdown, 15*time.Minute)
		err := r.RunOnce(ctx)
		cancel()

		if err != nil && !errors.Is(err, clashapi.ErrDisabled) {
			// ErrDisabled is the ordinary state of a panel whose sing-box has
			// no clash_api, not an incident — logging it every 10 minutes
			// would bury the log in something the operator chose.
			logger.Warn("Subscription probe sweep failed", zap.Error(err))
		}

		timer.Reset(r.env.Settings().Normalize().Interval)
	}
}

// RunOnce probes every enabled subscription, sequentially.
//
// Sequential across subscriptions, concurrent within one: this runs on home
// routers, and the pool bound is per-subscription, so probing four
// subscriptions at once would quietly multiply the real concurrency by four.
//
// Returns an error only for conditions that prevented the sweep as a whole
// (no controller, bad secret, unreadable config). One subscription failing is
// logged and does not abandon the rest — the same contract the subscription
// updater's bulk refresh has.
func (r *Runner) RunOnce(ctx context.Context) error {
	targets, err := r.env.Targets()
	if err != nil {
		return fmt.Errorf("failed to list probe targets: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}

	release, err := r.acquire()
	if err != nil {
		return err
	}
	defer release()

	prober, tags, settings, err := r.prepare()
	if err != nil {
		r.setLastError(err)
		return err
	}

	var firstErr error
	for _, target := range targets {
		err := r.runTarget(ctx, prober, tags, target, settings)
		switch {
		case err == nil:
		case errors.Is(err, ErrNoOwnedNodes), errors.Is(err, ErrNoNodesTestable):
			// Ordinary states of a sweep across many subscriptions — one not
			// yet imported, one not yet applied — and neither says anything
			// about the others, so the sweep moves on. RunSubscription reports
			// them, because there the operator asked about that one.
			continue
		case errors.Is(err, clashapi.ErrUnauthorized):
			// Fails every node identically and is the panel's own
			// misconfiguration. Abandon rather than write a total outage into
			// every provider's permanent history.
			r.setLastError(err)
			return err
		default:
			logger.Warn("Failed to probe subscription",
				zap.String("subscription", target.SubID), zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	r.setLastError(firstErr)
	return firstErr
}

// RunSubscription probes exactly one subscription now — the per-row button.
//
// Unlike a sweep, every reason for producing no sample is reported: the
// operator asked about this subscription specifically, so "nothing happened"
// is not an acceptable answer.
func (r *Runner) RunSubscription(ctx context.Context, subID string) (Sample, error) {
	targets, err := r.env.Targets()
	if err != nil {
		return Sample{}, fmt.Errorf("failed to list probe targets: %w", err)
	}

	var target Target
	found := false
	for _, candidate := range targets {
		if candidate.SubID == subID {
			target, found = candidate, true
			break
		}
	}
	if !found {
		return Sample{}, fmt.Errorf("%w: %q", ErrNoSuchTarget, subID)
	}

	release, err := r.acquire()
	if err != nil {
		return Sample{}, err
	}
	defer release()

	ctx, cancel := r.withShutdown(ctx)
	defer cancel()

	prober, tags, settings, err := r.prepare()
	if err != nil {
		r.setLastError(err)
		return Sample{}, err
	}

	if err := r.runTarget(ctx, prober, tags, target, settings); err != nil {
		r.setLastError(err)
		return Sample{}, err
	}

	snapshot := r.LatestNodes(subID)
	if snapshot == nil {
		return Sample{}, ErrNoOwnedNodes
	}
	return snapshot.Sample, nil
}

// acquire takes the single-run lock and returns its release.
//
// One run at a time: a slow sweep overrunning its interval would otherwise
// stack, and each layer would compete for the same nodes on the same router.
// The manual probe shares this gate, so a "test now" click cannot double the
// real concurrency by racing the scheduler.
func (r *Runner) acquire() (func(), error) {
	r.mu.Lock()
	if r.stopping {
		r.mu.Unlock()
		return nil, errors.New("the prober is shutting down")
	}
	if r.inFlight {
		r.mu.Unlock()
		return nil, errors.New("a probe sweep is already in progress")
	}
	r.inFlight = true
	// Add happens under the same lock Stop sets `stopping` under, and Stop
	// only Waits after releasing it — so Add is never racing a Wait.
	r.inFlightWG.Add(1)
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		r.inFlight = false
		r.lastRun = time.Now()
		r.mu.Unlock()
		r.inFlightWG.Done()
	}, nil
}

// withShutdown extends a caller's context so it is also cancelled when the
// runner stops.
//
// The manual probe's context comes from an HTTP request, which knows nothing
// about the panel shutting down. Without this, Stop would cancel the scheduled
// sweep and then block waiting for a manual probe that keeps running for its
// full duration.
func (r *Runner) withShutdown(ctx context.Context) (context.Context, context.CancelFunc) {
	r.mu.RLock()
	shutdown := r.shutdown
	r.mu.RUnlock()

	derived, cancel := context.WithCancel(ctx)
	stopWatching := context.AfterFunc(shutdown, cancel)
	return derived, func() {
		stopWatching()
		cancel()
	}
}

// prepare resolves everything a sweep reads from live state.
func (r *Runner) prepare() (Prober, []string, Settings, error) {
	prober, err := r.env.Prober()
	if err != nil {
		// Nothing is recorded. The panel being unable to ask is not evidence
		// that a provider is down, and a 0% point written here would be an
		// outage the operator never had — permanently, in the chart.
		return nil, nil, Settings{}, err
	}

	tags, err := r.env.OutboundTags()
	if err != nil {
		return nil, nil, Settings{}, fmt.Errorf("failed to read outbound tags: %w", err)
	}

	return prober, tags, r.env.Settings().Normalize(), nil
}

// LatestNodes returns the per-node detail of the most recent run for one
// subscription, or nil when it has not been probed since the panel started.
func (r *Runner) LatestNodes(subID string) *Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snapshot, ok := r.snapshots[subID]
	if !ok {
		return nil
	}
	// Copied out: the caller marshals this on an HTTP goroutine while the next
	// sweep may already be replacing it.
	results := make([]NodeResult, len(snapshot.Results))
	copy(results, snapshot.Results)
	return &Snapshot{At: snapshot.At, Sample: snapshot.Sample, Results: results}
}

// Status is what the UI reads to explain itself when there is no data yet.
type Status struct {
	Running   bool      `json:"running"`
	InFlight  bool      `json:"in_flight"`
	LastRun   time.Time `json:"last_run,omitempty"`
	LastError string    `json:"last_error,omitempty"`
	Interval  string    `json:"interval"`
}

// Status reports the scheduler's state.
func (r *Runner) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Status{
		Running:   r.running,
		InFlight:  r.inFlight,
		LastRun:   r.lastRun,
		LastError: r.lastError,
		Interval:  r.env.Settings().Normalize().Interval.String(),
	}
}

func (r *Runner) setLastError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err == nil {
		r.lastError = ""
		return
	}
	r.lastError = err.Error()
}
