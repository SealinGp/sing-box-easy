package config

import (
	"testing"
	"time"
)

// A urltest group dials every member on every tick, so the probe rate is
// members/interval. Left unbounded it saturates a router: a 272-node group on a
// 10s interval is ~27 proxy handshakes per second, forever. That starves
// everything else sing-box does at startup — including its rule-set downloads,
// which then fail and abort the whole start ("timeout: no recent network
// activity"), putting procd into a crash loop.
func TestEffectiveURLTestIntervalBoundsTheProbeRate(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		members    int
		want       time.Duration
	}{
		{
			name:       "large group on a hand-typed 10s is stretched to bound the rate",
			configured: "10s",
			members:    272,
			want:       68 * time.Second, // 272 / 4 probes-per-second
		},
		{
			name:       "small group keeps the operator's short interval",
			configured: "10s",
			members:    21,
			want:       10 * time.Second,
		},
		{
			name:       "a generous interval is never shortened",
			configured: "5m",
			members:    272,
			want:       5 * time.Minute,
		},
		{
			name:       "empty falls back to the sing-box-parity default",
			configured: "",
			members:    50,
			want:       3 * time.Minute,
		},
		{
			name:       "unparseable falls back to the default rather than being dropped",
			configured: "not-a-duration",
			members:    50,
			want:       3 * time.Minute,
		},
		{
			name:       "an absurdly short interval is floored even for a tiny group",
			configured: "100ms",
			members:    2,
			want:       10 * time.Second,
		},
		{
			name:       "an empty group cannot divide by zero",
			configured: "10s",
			members:    0,
			want:       10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveURLTestInterval(tt.configured, tt.members)
			if got != tt.want {
				t.Errorf("effectiveURLTestInterval(%q, %d) = %v, want %v",
					tt.configured, tt.members, got, tt.want)
			}
		})
	}
}

func TestBuildURLTestOptionsAppliesTheBoundedInterval(t *testing.T) {
	members := make([]string, 272)
	for i := range members {
		members[i] = "node"
	}

	opts := buildURLTestOptions(members, FilterSpec{TestInterval: "10s"})

	if time.Duration(opts.Interval) != 68*time.Second {
		t.Errorf("interval = %v, want 68s", time.Duration(opts.Interval))
	}
}

func TestBuildURLTestOptionsAlwaysSetsAnInterval(t *testing.T) {
	// Previously an unset or unparseable interval was omitted so sing-box would
	// apply its own default. That is fine in isolation, but it means the panel
	// cannot promise a bounded probe rate — so an interval is now always
	// written explicitly.
	opts := buildURLTestOptions([]string{"a", "b"}, FilterSpec{})

	if opts.Interval == 0 {
		t.Error("expected an explicit interval, got 0")
	}
}
