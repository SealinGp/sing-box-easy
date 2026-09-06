package subprobe

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
)

// fakeEnv is a scriptable Environment.
type fakeEnv struct {
	targets  []Target
	tags     []string
	settings Settings
	prober   Prober
	probeErr error
	tagsErr  error
}

func (f *fakeEnv) Targets() ([]Target, error) { return f.targets, nil }
func (f *fakeEnv) OutboundTags() ([]string, error) {
	if f.tagsErr != nil {
		return nil, f.tagsErr
	}
	return f.tags, nil
}
func (f *fakeEnv) Prober() (Prober, error) {
	if f.probeErr != nil {
		return nil, f.probeErr
	}
	return f.prober, nil
}
func (f *fakeEnv) Settings() Settings { return f.settings }

// fakeProber answers from a table: tag -> (delay, error).
type fakeProber struct {
	mu       sync.Mutex
	delays   map[string]int
	errs     map[string]error
	calls    int32
	urls     []string
	maxSeen  int32
	inFlight int32
}

func (p *fakeProber) Delay(ctx context.Context, tag, testURL string, timeout time.Duration) (int, error) {
	n := atomic.AddInt32(&p.inFlight, 1)
	for {
		peak := atomic.LoadInt32(&p.maxSeen)
		if n <= peak || atomic.CompareAndSwapInt32(&p.maxSeen, peak, n) {
			break
		}
	}
	// Hold briefly so overlapping calls are observable.
	time.Sleep(5 * time.Millisecond)
	atomic.AddInt32(&p.inFlight, -1)
	atomic.AddInt32(&p.calls, 1)

	p.mu.Lock()
	p.urls = append(p.urls, testURL)
	p.mu.Unlock()

	if err, ok := p.errs[tag]; ok {
		return 0, err
	}
	return p.delays[tag], nil
}

func defaultSettings() Settings {
	return Settings{
		Interval:    10 * time.Minute,
		Timeout:     5 * time.Second,
		Concurrency: 4,
		MaxAge:      7 * 24 * time.Hour,
		MaxPoints:   2016,
	}
}

// TestRunOnceRecordsPerSubscriptionAggregate is the feature's headline
// behaviour: nodes are grouped by the subscription that owns their tag, and one
// aggregate row is written per subscription.
func TestRunOnceRecordsPerSubscriptionAggregate(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: "https://example.test/204"}},
		settings: defaultSettings(),
		tags: []string{
			"HK 01 aaaa | sub_a",
			"US 02 bbbb | sub_a",
			"JP 03 cccc | sub_a",
			// Not owned by any subscription — a hand-written outbound and a
			// rules-managed group. Probing these would measure something the
			// provider is not responsible for.
			"direct",
			"🤖 AI",
			// Owned by a DIFFERENT subscription that is not being probed.
			"SG 01 dddd | sub_b",
		},
		prober: &fakeProber{
			delays: map[string]int{"HK 01 aaaa | sub_a": 300, "US 02 bbbb | sub_a": 500},
			errs:   map[string]error{"JP 03 cccc | sub_a": clashapi.ErrDelayFailed},
		},
	}

	runner := NewRunner(store, env)
	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	latest, err := store.Latest()
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("stored %d subscriptions, want 1 (only enabled targets)", len(latest))
	}

	// The example from the feature request, exactly.
	got := latest["sub_a"]
	if got.Total != 3 || got.Reachable != 2 || got.AvgMs != 400 {
		t.Errorf("sample = %+v, want 2/3 reachable at 400ms", got.Sample)
	}

	// The configured per-subscription URL must be the one actually tested.
	fp := env.prober.(*fakeProber)
	for _, u := range fp.urls {
		if u != "https://example.test/204" {
			t.Errorf("probed %q, want the subscription's configured URL", u)
		}
	}
}

// TestRunOnceMarksUnknownTagsSkipped — a node in the config but not in the
// running sing-box cannot be tested, and counting it as down would blame the
// provider for an unapplied config.
func TestRunOnceMarksUnknownTagsSkipped(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		tags:     []string{"HK 01 aaaa | sub_a", "GONE 02 bbbb | sub_a"},
		prober: &fakeProber{
			delays: map[string]int{"HK 01 aaaa | sub_a": 120},
			errs:   map[string]error{"GONE 02 bbbb | sub_a": clashapi.ErrProxyNotFound},
		},
	}

	runner := NewRunner(store, env)
	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	latest, _ := store.Latest()
	got := latest["sub_a"]
	if got.Total != 1 || got.Reachable != 1 {
		t.Errorf("sample = %+v, want 1/1 (the untestable node excluded)", got.Sample)
	}

	// It still has to be REPORTED, or the count silently shrinks with nothing
	// on screen explaining why.
	snapshot := runner.LatestNodes("sub_a")
	if snapshot == nil {
		t.Fatal("no node snapshot recorded")
	}
	if snapshot.Sample.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", snapshot.Sample.Skipped)
	}
}

// TestRunOnceSkipsWhenSingBoxUnavailable — the panel being unable to ask is not
// evidence that the provider is down. Recording 0% here would draw an outage
// that never happened, every time sing-box is restarted or clash_api is off.
func TestRunOnceSkipsWhenSingBoxUnavailable(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		probeErr: clashapi.ErrDisabled,
	}

	runner := NewRunner(store, env)
	err := runner.RunOnce(context.Background())
	if !errors.Is(err, clashapi.ErrDisabled) {
		t.Fatalf("err = %v, want ErrDisabled", err)
	}

	latest, _ := store.Latest()
	if len(latest) != 0 {
		t.Fatalf("recorded %d samples while sing-box was unavailable, want 0", len(latest))
	}
}

// TestRunOnceAbortsOnUnauthorized — a wrong secret fails EVERY node. Recording
// that as a total outage would be the panel's own misconfiguration written into
// the provider's history, where it stays after the secret is fixed.
func TestRunOnceAbortsOnUnauthorized(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		tags:     []string{"HK 01 aaaa | sub_a", "US 02 bbbb | sub_a"},
		prober: &fakeProber{errs: map[string]error{
			"HK 01 aaaa | sub_a": clashapi.ErrUnauthorized,
			"US 02 bbbb | sub_a": clashapi.ErrUnauthorized,
		}},
	}

	runner := NewRunner(store, env)
	if err := runner.RunOnce(context.Background()); !errors.Is(err, clashapi.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	latest, _ := store.Latest()
	if len(latest) != 0 {
		t.Fatalf("recorded %d samples on an auth failure, want 0", len(latest))
	}
}

// TestRunOnceSkipsSubscriptionWithNoNodes — a subscription whose nodes have not
// been imported yet has nothing to measure, and a 0% point would misreport it.
func TestRunOnceSkipsSubscriptionWithNoNodes(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_empty", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		tags:     []string{"direct", "HK 01 aaaa | sub_other"},
		prober:   &fakeProber{},
	}

	runner := NewRunner(store, env)
	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	latest, _ := store.Latest()
	if len(latest) != 0 {
		t.Fatalf("recorded %d samples for a subscription with no nodes, want 0", len(latest))
	}
}

// TestRunOnceBoundsConcurrency — this runs on home routers, and the production
// config has 233 subscription nodes. Firing them all at once is how a probe
// takes the router down with it.
func TestRunOnceBoundsConcurrency(t *testing.T) {
	store := newStore(t)
	settings := defaultSettings()
	settings.Concurrency = 3

	tags := make([]string, 0, 20)
	delays := make(map[string]int, 20)
	for i := 0; i < 20; i++ {
		tag := fmt.Sprintf("node %02d | sub_a", i)
		tags = append(tags, tag)
		delays[tag] = 100
	}

	prober := &fakeProber{delays: delays}
	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: settings,
		tags:     tags,
		prober:   prober,
	}

	if err := NewRunner(store, env).RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := atomic.LoadInt32(&prober.calls); got != 20 {
		t.Errorf("calls = %d, want 20", got)
	}
	if peak := atomic.LoadInt32(&prober.maxSeen); peak > 3 {
		t.Errorf("peak concurrency = %d, want <= 3", peak)
	}
}

// TestRunOnceTrimsAfterWriting — the retention bound has to be enforced on the
// write path. A cron-only sweep lets a misconfigured interval fill a router's
// overlay between sweeps.
//
// The cap used here is MinPoints rather than a smaller round number, because
// Settings.Normalize clamps anything below it: a cap of "3" is not a
// configuration the runner can ever be given, so a test that assumed one would
// be testing code that cannot run.
func TestRunOnceTrimsAfterWriting(t *testing.T) {
	store := newStore(t)
	settings := defaultSettings()
	settings.MaxPoints = MinPoints

	// Fill past the cap, then let a run add one more.
	base := time.Now().Add(-24 * time.Hour)
	for i := 0; i < MinPoints+5; i++ {
		_ = store.Insert("sub_a", base.Add(time.Duration(i)*time.Minute), Sample{Total: 1, Reachable: 1})
	}

	env := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: settings,
		tags:     []string{"HK 01 aaaa | sub_a"},
		prober:   &fakeProber{delays: map[string]int{"HK 01 aaaa | sub_a": 100}},
	}

	if err := NewRunner(store, env).RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	points, _ := store.Series("sub_a", base.Add(-time.Hour), time.Now().Add(time.Hour), 0)
	if len(points) != MinPoints {
		t.Errorf("stored points = %d, want %d (cap enforced on write)", len(points), MinPoints)
	}
	// The newest row must be the one the run just wrote, not an evicted-into
	// place older one.
	if points[len(points)-1].AvgMs != 100 {
		t.Errorf("newest point avg = %d, want the run just written (100)", points[len(points)-1].AvgMs)
	}
}

// TestRunSubscriptionProbesOnlyThatOne backs the per-row "probe now" button.
func TestRunSubscriptionProbesOnlyThatOne(t *testing.T) {
	store := newStore(t)
	prober := &fakeProber{delays: map[string]int{
		"HK 01 aaaa | sub_a": 100,
		"SG 01 bbbb | sub_b": 200,
	}}
	env := &fakeEnv{
		targets: []Target{
			{SubID: "sub_a", URL: DefaultProbeURL},
			{SubID: "sub_b", URL: DefaultProbeURL},
		},
		settings: defaultSettings(),
		tags:     []string{"HK 01 aaaa | sub_a", "SG 01 bbbb | sub_b"},
		prober:   prober,
	}

	runner := NewRunner(store, env)
	sample, err := runner.RunSubscription(context.Background(), "sub_a")
	if err != nil {
		t.Fatalf("run one: %v", err)
	}
	if sample.Total != 1 || sample.AvgMs != 100 {
		t.Errorf("sample = %+v", sample)
	}

	latest, _ := store.Latest()
	if _, ok := latest["sub_b"]; ok {
		t.Error("sub_b was probed by a single-subscription run")
	}
}

// TestRunSubscriptionDistinguishesNotImportedFromNotRunning.
//
// Found by running this against a live sing-box whose config had been
// re-tagged: every node 404'd, and the runner reported "no outbounds belong to
// this subscription" — which sends the operator to refresh a subscription when
// the actual fix is to apply the config. The two states look identical from
// the outside and must not share a message.
func TestRunSubscriptionDistinguishesNotImportedFromNotRunning(t *testing.T) {
	store := newStore(t)

	// Nothing imported: no outbound carries the ownership suffix.
	notImported := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		tags:     []string{"direct", "HK 01 aaaa | sub_other"},
		prober:   &fakeProber{},
	}
	_, err := NewRunner(store, notImported).RunSubscription(context.Background(), "sub_a")
	if !errors.Is(err, ErrNoOwnedNodes) {
		t.Errorf("not-imported err = %v, want ErrNoOwnedNodes", err)
	}

	// Imported but not applied: the tags exist in the config, and the running
	// sing-box has none of them.
	notRunning := &fakeEnv{
		targets:  []Target{{SubID: "sub_a", URL: DefaultProbeURL}},
		settings: defaultSettings(),
		tags:     []string{"HK 01 aaaa | sub_a", "US 02 bbbb | sub_a"},
		prober: &fakeProber{errs: map[string]error{
			"HK 01 aaaa | sub_a": clashapi.ErrProxyNotFound,
			"US 02 bbbb | sub_a": clashapi.ErrProxyNotFound,
		}},
	}
	runner := NewRunner(store, notRunning)
	_, err = runner.RunSubscription(context.Background(), "sub_a")
	if !errors.Is(err, ErrNoNodesTestable) {
		t.Errorf("not-running err = %v, want ErrNoNodesTestable", err)
	}

	// Neither case may write a sample: a 0% point here is an outage that never
	// happened, and it would stay in the chart forever.
	latest, _ := store.Latest()
	if len(latest) != 0 {
		t.Errorf("stored %d samples, want 0", len(latest))
	}

	// The node list must still be populated for the untestable case — it is
	// exactly when someone opens it to find out what went wrong.
	snapshot := runner.LatestNodes("sub_a")
	if snapshot == nil || len(snapshot.Results) != 2 {
		t.Fatalf("snapshot = %+v, want 2 recorded results", snapshot)
	}
	if snapshot.Sample.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", snapshot.Sample.Skipped)
	}
}

// TestRunSubscriptionUnknownID surfaces a clear error rather than silently
// doing nothing — the button would otherwise spin and report success.
func TestRunSubscriptionUnknownID(t *testing.T) {
	store := newStore(t)
	env := &fakeEnv{settings: defaultSettings(), prober: &fakeProber{}}
	_, err := NewRunner(store, env).RunSubscription(context.Background(), "nope")
	if !errors.Is(err, ErrNoSuchTarget) {
		t.Fatalf("err = %v, want ErrNoSuchTarget", err)
	}
}
