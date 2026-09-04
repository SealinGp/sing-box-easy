package trafficflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
)

func conn(id, inbound, source, host, rule string, chains []string, up, down int64) clashapi.Connection {
	return clashapi.Connection{
		ID:       id,
		Upload:   up,
		Download: down,
		Chains:   chains,
		Rule:     rule,
		Metadata: clashapi.ConnectionMetadata{
			Network:       "tcp",
			Type:          inbound,
			SourceIP:      source,
			DestinationIP: "1.2.3.4",
			Host:          host,
		},
	}
}

/* ── RuleIndex ──────────────────────────────────────────────────────────── */

func TestRuleIndexRecoversPositionFromRulesEndpoint(t *testing.T) {
	// `/rules` is config order; a connection's `rule` is "payload => proxy".
	index := BuildRuleIndex([]clashapi.Rule{
		{Payload: "inbound=dns-in", Proxy: "hijack-dns"},
		{Payload: "", Proxy: "sniff"},
		{Payload: "rule_set=sea-rulesets-ai", Proxy: "route(🤖 AI)"},
	})

	got, ok := index.Lookup("rule_set=sea-rulesets-ai => route(🤖 AI)")
	if !ok || got != 2 {
		t.Fatalf("Lookup = %d, %v; want 2, true", got, ok)
	}
	if _, ok := index.Lookup("domain=nowhere => route(x)"); ok {
		t.Fatal("unknown rule string must not resolve")
	}
}

// Two byte-identical rules: sing-box matches the first, so the first index is
// the true one. The second is dead config — the expected-flow diagram already
// says so.
func TestRuleIndexFirstDuplicateWins(t *testing.T) {
	index := BuildRuleIndex([]clashapi.Rule{
		{Payload: "domain=a.com", Proxy: "route(x)"},
		{Payload: "domain=a.com", Proxy: "route(x)"},
	})
	if got, _ := index.Lookup("domain=a.com => route(x)"); got != 0 {
		t.Fatalf("Lookup = %d, want 0", got)
	}
}

/* ── Differ ─────────────────────────────────────────────────────────────── */

func TestDifferDerivesRatesFromElapsedTime(t *testing.T) {
	d := NewDiffer()
	t0 := time.Unix(1_700_000_000, 0)

	first, closed := d.Step(t0, &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "10.0.0.2", "x.com", "final", []string{"direct"}, 100, 1000),
	}})
	if closed != 0 || len(first) != 1 || !first[0].Fresh || first[0].DownRate != 0 {
		t.Fatalf("first sighting = %+v closed=%d; want fresh with zero rate", first, closed)
	}

	// Two seconds later, 4000 more bytes down: 2000 B/s — per SECOND, not per
	// tick, so a slow poll does not read as a fast link.
	second, _ := d.Step(t0.Add(2*time.Second), &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "10.0.0.2", "x.com", "final", []string{"direct"}, 300, 5000),
	}})
	if second[0].Fresh || second[0].DownRate != 2000 || second[0].UpRate != 100 {
		t.Fatalf("second = %+v; want down 2000 B/s up 100 B/s", second[0])
	}
}

func TestDifferCountsClosedConnections(t *testing.T) {
	d := NewDiffer()
	t0 := time.Unix(1_700_000_000, 0)
	d.Step(t0, &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "s", "h", "final", []string{"direct"}, 0, 0),
		conn("b", "tun/tun-in", "s", "h", "final", []string{"direct"}, 0, 0),
	}})
	_, closed := d.Step(t0.Add(time.Second), &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("b", "tun/tun-in", "s", "h", "final", []string{"direct"}, 0, 0),
	}})
	if closed != 1 {
		t.Fatalf("closed = %d, want 1", closed)
	}
}

// A counter that goes DOWN means sing-box restarted and handed out a fresh
// connection under a reused id, or the number wrapped. Either way a negative
// rate is nonsense; clamp rather than draw a ribbon flowing backwards.
func TestDifferClampsNegativeDeltas(t *testing.T) {
	d := NewDiffer()
	t0 := time.Unix(1_700_000_000, 0)
	d.Step(t0, &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "s", "h", "final", []string{"direct"}, 0, 9000),
	}})
	out, _ := d.Step(t0.Add(time.Second), &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "tun/tun-in", "s", "h", "final", []string{"direct"}, 0, 100),
	}})
	if out[0].DownRate != 0 {
		t.Fatalf("rate = %v, want 0", out[0].DownRate)
	}
}

/* ── Aggregate ──────────────────────────────────────────────────────────── */

func liveSet() ([]Live, *RuleIndex) {
	index := BuildRuleIndex([]clashapi.Rule{
		{Payload: "inbound=dns-in", Proxy: "hijack-dns"},
		{Payload: "rule_set=sea-rulesets-ai", Proxy: "route(🤖 AI)"},
		{Payload: "rule_set=geosite-google", Proxy: "route(Google)"},
	})
	ai := "rule_set=sea-rulesets-ai => route(🤖 AI)"
	google := "rule_set=geosite-google => route(Google)"
	mk := func(id, inbound, src, host, rule string, chains []string, down float64) Live {
		return Live{
			Connection: conn(id, inbound, src, host, rule, chains, 0, 0),
			DownRate:   down,
			UpRate:     down / 10,
		}
	}
	return []Live{
		mk("1", "tun/tun-in", "192.168.9.20", "api.openai.com", ai, []string{"新加坡03", "🤖 AI"}, 3000),
		mk("2", "tun/tun-in", "192.168.9.21", "claude.ai", ai, []string{"新加坡01", "🤖 AI"}, 1000),
		mk("3", "mixed/mixed-in", "192.168.9.20", "www.google.com", google, []string{"台湾01", "🌐 Google", "Google"}, 500),
		mk("4", "tun/tun-in", "192.168.9.22", "1.1.1.1", "final", []string{"➡️ 直连"}, 200),
		// A rule string sing-box no longer has: config edited since it started.
		mk("5", "tun/tun-in", "192.168.9.22", "old.example", "domain=old.example => route(Claude)", []string{"relay", "Claude"}, 50),
	}, index
}

func TestAggregateBuildsEdgesFromRuleExitAndInbound(t *testing.T) {
	live, index := liveSet()
	frame := Aggregate(time.Unix(1_700_000_000, 0), live, index, Filter{})

	if frame.Totals.Connections != 5 || frame.Totals.Down != 4750 {
		t.Fatalf("totals = %+v", frame.Totals)
	}

	// Rules are sorted by download rate, so the AI rule leads.
	if len(frame.Rules) != 4 {
		t.Fatalf("rules = %+v", frame.Rules)
	}
	ai := frame.Rules[0]
	if ai.Kind != "rule" || ai.Index != 1 || ai.Exit != "🤖 AI" || ai.Down != 4000 || ai.Connections != 2 {
		t.Fatalf("ai rule = %+v", ai)
	}
	// Top hosts per rule, highest first — what someone hovering the ribbon wants.
	if len(ai.Hosts) != 2 || ai.Hosts[0].Host != "api.openai.com" {
		t.Fatalf("ai hosts = %+v", ai.Hosts)
	}

	var final, unmatched *RuleFlow
	for i := range frame.Rules {
		switch frame.Rules[i].Kind {
		case "final":
			final = &frame.Rules[i]
		case "unmatched":
			unmatched = &frame.Rules[i]
		}
	}
	if final == nil || final.Exit != "➡️ 直连" || final.Index != -1 {
		t.Fatalf("final = %+v", final)
	}
	if unmatched == nil || unmatched.Rule != "domain=old.example => route(Claude)" || unmatched.Exit != "Claude" {
		t.Fatalf("unmatched = %+v", unmatched)
	}
	if frame.Unmatched != 1 {
		t.Fatalf("frame.Unmatched = %d", frame.Unmatched)
	}

	// Exits: keyed by the LAST chain element, with the leaf as `via`.
	if frame.Exits[0].Tag != "🤖 AI" || frame.Exits[0].Connections != 2 || len(frame.Exits[0].Via) != 2 {
		t.Fatalf("exit = %+v", frame.Exits[0])
	}
	if frame.Exits[0].Via[0].Tag != "新加坡03" {
		t.Fatalf("via order = %+v", frame.Exits[0].Via)
	}

	// Inbounds: tag parsed from "type/tag".
	tags := map[string]int{}
	for _, in := range frame.Inbounds {
		tags[in.Tag] = in.Connections
	}
	if tags["tun-in"] != 4 || tags["mixed-in"] != 1 {
		t.Fatalf("inbounds = %+v", frame.Inbounds)
	}
}

func TestAggregateFilterBySourceIPAndHost(t *testing.T) {
	live, index := liveSet()

	bySource := Aggregate(time.Now(), live, index, Filter{SourceIP: "192.168.9.20"})
	if bySource.Totals.Connections != 2 || !bySource.Filtered {
		t.Fatalf("by source = %+v", bySource.Totals)
	}

	// Host is a case-insensitive substring: "google" finds www.google.com.
	byHost := Aggregate(time.Now(), live, index, Filter{Host: "GOOGLE"})
	if byHost.Totals.Connections != 1 || byHost.Rules[0].Exit != "Google" {
		t.Fatalf("by host = %+v", byHost.Rules)
	}

	// A host filter also matches the destination IP, for connections that
	// never had a domain.
	byIP := Aggregate(time.Now(), live, index, Filter{Host: "1.1.1.1"})
	if byIP.Totals.Connections != 1 {
		t.Fatalf("by ip = %+v", byIP.Totals)
	}
	// The unfiltered total is still reported, so "2 of 5" can be shown.
	if bySource.Totals.All != 5 {
		t.Fatalf("All = %d, want 5", bySource.Totals.All)
	}
}

// The source picker's list is built before the filter, so choosing one device
// still leaves the others selectable. It is ordered by address numerically —
// .20 before .100 — because a list that reshuffled by rate every second could
// not be clicked reliably.
func TestAggregateSourcesArePreFilterAndOrderedByAddress(t *testing.T) {
	live, index := liveSet()
	live = append(live, Live{
		Connection: conn("6", "tun/tun-in", "192.168.9.100", "a.example", "final", []string{"direct"}, 0, 0),
	})

	full := Aggregate(time.Now(), live, index, Filter{})
	got := make([]string, 0, len(full.Sources))
	for _, src := range full.Sources {
		got = append(got, src.IP)
	}
	want := []string{"192.168.9.20", "192.168.9.21", "192.168.9.22", "192.168.9.100"}
	if len(got) != len(want) {
		t.Fatalf("sources = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sources = %v, want %v", got, want)
		}
	}

	if full.Sources[0].Connections != 2 || full.Sources[0].Down != 3500 {
		t.Fatalf("busiest source = %+v", full.Sources[0])
	}

	// Narrowed to one device, every device is still on offer.
	narrowed := Aggregate(time.Now(), live, index, Filter{SourceIP: "192.168.9.20"})
	if len(narrowed.Sources) != len(want) {
		t.Fatalf("filtered sources = %+v; the picker must not collapse", narrowed.Sources)
	}
	if narrowed.Totals.Connections != 2 {
		t.Fatalf("filter stopped working: %+v", narrowed.Totals)
	}
}

func TestAggregateInboundTagFallsBackToType(t *testing.T) {
	live := []Live{{Connection: conn("1", "tun", "s", "h", "final", []string{"direct"}, 0, 0)}}
	frame := Aggregate(time.Now(), live, BuildRuleIndex(nil), Filter{})
	if frame.Inbounds[0].Tag != "tun" {
		t.Fatalf("tag = %q", frame.Inbounds[0].Tag)
	}
}

func TestAggregateSurvivesEmptyChains(t *testing.T) {
	live := []Live{{Connection: conn("1", "tun/tun-in", "s", "h", "final", nil, 0, 0)}}
	frame := Aggregate(time.Now(), live, BuildRuleIndex(nil), Filter{})
	if len(frame.Exits) != 0 || frame.Totals.Connections != 1 {
		t.Fatalf("frame = %+v", frame)
	}
}

/* ── Run ────────────────────────────────────────────────────────────────── */

type fakeSource struct {
	rules     []clashapi.Rule
	snapshots []*clashapi.Snapshot
	calls     int
	rulesGets int
	err       error
}

func (f *fakeSource) Connections(context.Context) (*clashapi.Snapshot, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.calls >= len(f.snapshots) {
		return f.snapshots[len(f.snapshots)-1], nil
	}
	s := f.snapshots[f.calls]
	f.calls++
	return s, nil
}

func (f *fakeSource) Rules(context.Context) ([]clashapi.Rule, error) {
	f.rulesGets++
	return f.rules, nil
}

func TestRunEmitsAFramePerTickAndStopsOnCancel(t *testing.T) {
	src := &fakeSource{
		rules: []clashapi.Rule{{Payload: "rule_set=x", Proxy: "route(P)"}},
		snapshots: []*clashapi.Snapshot{
			{Connections: []clashapi.Connection{conn("a", "tun/tun-in", "s", "h", "rule_set=x => route(P)", []string{"n", "P"}, 0, 0)}},
			{Connections: []clashapi.Connection{conn("a", "tun/tun-in", "s", "h", "rule_set=x => route(P)", []string{"n", "P"}, 0, 5000)}},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	var frames []*Frame
	err := Run(ctx, src, Options{Interval: 5 * time.Millisecond}, func(f *Frame) error {
		frames = append(frames, f)
		if len(frames) == 2 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(frames) < 2 {
		t.Fatalf("frames = %d, want >= 2", len(frames))
	}
	// The first frame is emitted immediately — before the first tick — so the
	// diagram lights up on connect rather than one interval later.
	if frames[0].Rules[0].Index != 0 {
		t.Fatalf("first frame rule = %+v", frames[0].Rules[0])
	}
	if frames[1].Rules[0].Down <= 0 {
		t.Fatalf("second frame should carry a rate, got %+v", frames[1].Rules[0])
	}
}

func TestRunStopsWhenTheEmitterFails(t *testing.T) {
	src := &fakeSource{snapshots: []*clashapi.Snapshot{{}}}
	wantErr := errors.New("client went away")
	err := Run(context.Background(), src, Options{Interval: time.Millisecond}, func(*Frame) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want emitter error", err)
	}
}

func TestRunReportsSnapshotFailure(t *testing.T) {
	src := &fakeSource{err: errors.New("unreachable")}
	err := Run(context.Background(), src, Options{Interval: time.Millisecond}, func(*Frame) error { return nil })
	if err == nil || err.Error() != "unreachable" {
		t.Fatalf("err = %v", err)
	}
}

// An unmatched rule string means the running rule list may have changed under
// us — sing-box was reloaded — so the index is re-read, but only once the
// cooldown since the LAST read has passed. Re-reading immediately would return
// the list that was just fetched: a connection opened before a reload keeps
// its old rule string for its whole life, and no number of re-reads maps it.
func TestRunRefreshesRulesOnUnmatchedOnlyAfterCooldown(t *testing.T) {
	run := func(cooldown time.Duration) int {
		src := &fakeSource{
			rules: []clashapi.Rule{{Payload: "old", Proxy: "route(P)"}},
			snapshots: []*clashapi.Snapshot{
				{Connections: []clashapi.Connection{conn("a", "tun/tun-in", "s", "h", "new => route(P)", []string{"P"}, 0, 0)}},
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		n := 0
		_ = Run(ctx, src, Options{Interval: time.Millisecond, RulesRefreshCooldown: cooldown}, func(*Frame) error {
			n++
			if n == 6 {
				cancel()
			}
			return nil
		})
		return src.rulesGets
	}

	if got := run(time.Hour); got != 1 {
		t.Fatalf("with a long cooldown rules were fetched %d times, want 1", got)
	}
	if got := run(time.Nanosecond); got < 2 {
		t.Fatalf("with no effective cooldown rules were fetched %d times, want >= 2", got)
	}
}
