package autoselect

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"testing"
	"time"
)

type fakeSettings struct {
	raw  string
	fail bool
}

func (s *fakeSettings) Get(string) (string, error) {
	if s.raw == "" {
		return "", settings.ErrSettingNotFound
	}
	return s.raw, nil
}
func (s *fakeSettings) Set(_, v string) error {
	if s.fail {
		return errors.New("disk full")
	}
	s.raw = v
	return nil
}
func (s *fakeSettings) GetNodeQuality() settings.NodeQuality { return settings.DefaultNodeQuality() }
func (s *fakeSettings) GetProbeInterval() time.Duration      { return 10 * time.Minute }

type fakeHistory map[string][]nodehistory.Point

func (h fakeHistory) Recent([]string, int) (map[string][]nodehistory.Point, error) { return h, nil }

type fakeRuntime struct {
	calls, switched int
	fail, changed   bool
	now             string
}

func (r *fakeRuntime) Proxies(context.Context) ([]clashapi.Proxy, error) {
	r.calls++
	current := r.now
	if r.changed && r.calls%2 == 1 {
		current = "someone-else"
	}
	return []clashapi.Proxy{{Name: "g", Type: "Selector", Now: current, All: []string{"a"}}, {Name: "a", Type: "VLESS"}, {Name: "urltest", Type: "URLTest", All: []string{"a"}}}, nil
}
func (r *fakeRuntime) SelectProxy(_ context.Context, _, name string) error {
	if r.fail {
		return errors.New("controller refused")
	}
	r.switched++
	r.now = name
	return nil
}
func managerFixture(t *testing.T) (*Manager, *fakeSettings, *fakeRuntime, time.Time) {
	t.Helper()
	now := time.Now()
	s := &fakeSettings{}
	r := &fakeRuntime{}
	m := New(s, fakeHistory{"a": samples(now, 100, 100, 100)}, func() (Runtime, error) { return r, nil })
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	return m, s, r, now
}
func TestPolicyAndScheduler(t *testing.T) {
	m, s, r, now := managerFixture(t)
	ctx := context.Background()
	if _, err := m.Configure(ctx, "urltest", Policy{true, 300}); err == nil {
		t.Fatal("URLTest must not be controlled")
	}
	if _, err := m.Configure(ctx, "g", Policy{true, 59}); err == nil {
		t.Fatal("interval accepted")
	}
	if _, err := m.Configure(ctx, "g", Policy{true, 300}); err != nil {
		t.Fatal(err)
	}
	if s.raw == "" {
		t.Fatal("not persisted")
	}
	m.Tick(ctx, now)
	if r.switched != 1 {
		t.Fatal("did not select")
	}
	state := m.Statuses()["g"]
	if state.Reason != "switched" || state.LastSwitched.IsZero() || !state.NextCheck.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("%+v", state)
	}
	calls := r.calls
	m.Tick(ctx, now.Add(time.Minute))
	if r.calls != calls {
		t.Fatal("interval ignored")
	}
	m.Tick(ctx, now.Add(5*time.Minute))
	if r.switched != 1 || m.Statuses()["g"].Reason != "retained" {
		t.Fatal("qualified incumbent not retained")
	}
	restored := New(s, m.history, m.connect)
	if err := restored.Init(); err != nil {
		t.Fatal(err)
	}
	if !restored.Statuses()["g"].Enabled {
		t.Fatal("policy not restored")
	}
	if err := m.Pause("g"); err != nil {
		t.Fatal(err)
	}
	m.Tick(ctx, now.Add(time.Hour))
	if r.switched != 1 || m.Statuses()["g"].Enabled {
		t.Fatal("manual pause ignored")
	}
}
func TestFailedPolicyWriteDoesNotActivate(t *testing.T) {
	m, s, _, _ := managerFixture(t)
	s.fail = true
	if _, err := m.Configure(context.Background(), "g", Policy{true, 300}); err == nil {
		t.Fatal("expected disk failure")
	}
	if len(m.Statuses()) != 0 {
		t.Fatal("failed write activated policy")
	}
}
func TestFailedSwitchNotReportedAsSuccess(t *testing.T) {
	m, _, r, now := managerFixture(t)
	_, _ = m.Configure(context.Background(), "g", Policy{true, 300})
	r.fail = true
	m.Tick(context.Background(), now)
	state := m.Statuses()["g"]
	if state.Reason != "error" || state.Error == "" || !state.LastSwitched.IsZero() || state.Selected != "" {
		t.Fatalf("%+v", state)
	}
}
func TestGroupChangesAreNotOverwritten(t *testing.T) {
	m, _, r, now := managerFixture(t)
	_, _ = m.Configure(context.Background(), "g", Policy{true, 300})
	r.changed = true
	m.Tick(context.Background(), now)
	if r.switched != 0 || m.Statuses()["g"].Reason != "error" {
		t.Fatal("overwrote external selection")
	}
}
func TestDisableWorksOffline(t *testing.T) {
	m, _, _, _ := managerFixture(t)
	m.connect = func() (Runtime, error) { return nil, errors.New("offline") }
	if _, err := m.Configure(context.Background(), "g", Policy{false, 300}); err != nil {
		t.Fatal(err)
	}
}
func TestNoCandidateKeepsCurrent(t *testing.T) {
	m, _, r, now := managerFixture(t)
	m.history = fakeHistory{}
	_, _ = m.Configure(context.Background(), "g", Policy{true, 300})
	m.Tick(context.Background(), now)
	if r.switched != 0 || m.Statuses()["g"].Reason != "no_candidate" {
		t.Fatal("unsafe fallback")
	}
}

type blockingRuntime struct{ entered chan struct{} }

func (r blockingRuntime) Proxies(ctx context.Context) ([]clashapi.Proxy, error) {
	close(r.entered)
	<-ctx.Done()
	return nil, ctx.Err()
}
func (r blockingRuntime) SelectProxy(context.Context, string, string) error { return nil }
func TestStatusReadsDoNotWaitForRuntime(t *testing.T) {
	m, _, _, now := managerFixture(t)
	_, _ = m.Configure(context.Background(), "g", Policy{true, 300})
	blocked := blockingRuntime{make(chan struct{})}
	m.connect = func() (Runtime, error) { return blocked, nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); m.Tick(ctx, now) }()
	<-blocked.entered
	read := make(chan struct{})
	go func() { m.Statuses(); close(read) }()
	select {
	case <-read:
	case <-time.After(time.Second):
		t.Fatal("runtime blocked proxy list status reads")
	}
	cancel()
	<-done
}
func TestInitRejectsCorruptOrInvalidStoredPolicy(t *testing.T) {
	for _, raw := range []string{"broken", `{"g":{"enabled":true,"interval_seconds":0}}`} {
		m := New(&fakeSettings{raw: raw}, fakeHistory{}, nil)
		if m.Init() == nil {
			t.Fatal("invalid policy activated")
		}
	}
}

func TestFallbackSwitchAndRetain(t *testing.T) {
	m, _, r, now := managerFixture(t)
	m.history = fakeHistory{"a": samples(now, 100, 100, 800)}
	ctx := context.Background()
	if _, err := m.Configure(ctx, "g", Policy{true, 300}); err != nil {
		t.Fatal(err)
	}
	m.Tick(ctx, now)
	first := m.Statuses()["g"]
	if r.now != "a" || first.Reason != "fallback_switched" || !first.LastSwitched.Equal(now) {
		t.Fatal(first)
	}
	m.Tick(ctx, now.Add(5*time.Minute))
	retained := m.Statuses()["g"]
	if r.switched != 1 || retained.Reason != "fallback_retained" || !retained.LastSwitched.Equal(now) {
		t.Fatal(retained)
	}
}
