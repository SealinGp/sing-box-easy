package autoselect

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"testing"
	"time"
)

func samples(now time.Time, count, passed, delay int) []nodehistory.Point {
	result := make([]nodehistory.Point, count)
	for i := range result {
		ok := i >= count-passed
		result[i] = nodehistory.Point{Time: now.Add(-time.Duration(count-i) * time.Second), Success: ok, Delay: delay}
		if !ok {
			result[i].Delay = 0
		}
	}
	return result
}
func TestDecide(t *testing.T) {
	now := time.Now()
	q := settings.DefaultNodeQuality()
	leaf := func(name string) clashapi.Proxy { return clashapi.Proxy{Name: name, Type: "VLESS"} }
	group := clashapi.Proxy{Name: "g", Type: "Selector", Now: "current", All: []string{"a", "b", "current", "nested", "direct", "gone"}}
	list := []clashapi.Proxy{leaf("a"), leaf("b"), leaf("current"), {Name: "nested", Type: "URLTest", All: []string{"a"}}, {Name: "direct", Type: "Direct"}}
	for _, tt := range []struct {
		name         string
		history      map[string][]nodehistory.Point
		want, reason string
	}{
		{"97 inclusive", map[string][]nodehistory.Point{"a": samples(now, 100, 97, 500)}, "a", "switched"},
		{"below watermark fallback", map[string][]nodehistory.Point{"a": samples(now, 100, 96, 100)}, "a", "fallback_switched"},
		{"all slow fallback", map[string][]nodehistory.Point{"a": samples(now, 100, 100, 501)}, "a", "fallback_switched"},
		{"one point band", map[string][]nodehistory.Point{"a": samples(now, 100, 71, 200), "b": samples(now, 100, 72, 400)}, "a", "fallback_switched"},
		{"outside band", map[string][]nodehistory.Point{"a": samples(now, 100, 70, 100), "b": samples(now, 100, 72, 400)}, "b", "fallback_switched"},
		{"all slow favor availability", map[string][]nodehistory.Point{"a": samples(now, 100, 95, 600), "b": samples(now, 100, 99, 800)}, "b", "fallback_switched"},
		{"retain equally ranked fallback", map[string][]nodehistory.Point{"a": samples(now, 100, 100, 600), "current": samples(now, 100, 100, 600)}, "current", "fallback_retained"},
		{"incomplete", map[string][]nodehistory.Point{"a": samples(now, 99, 99, 100)}, "", "no_candidate"},
		{"stale", map[string][]nodehistory.Point{"a": samples(now.Add(-time.Hour), 100, 100, 100)}, "", "no_candidate"},
		{"future", map[string][]nodehistory.Point{"a": samples(now.Add(time.Hour), 100, 100, 100)}, "", "no_candidate"},
		{"quality first", map[string][]nodehistory.Point{"a": samples(now, 100, 97, 100), "b": samples(now, 100, 100, 400)}, "b", "switched"},
		{"p95 tie breaker", map[string][]nodehistory.Point{"a": samples(now, 100, 100, 400), "b": samples(now, 100, 100, 100)}, "b", "switched"},
		{"stable tie", map[string][]nodehistory.Point{"a": samples(now, 100, 100, 100), "b": samples(now, 100, 100, 100)}, "a", "switched"},
		{"hold incumbent", map[string][]nodehistory.Point{"current": samples(now, 100, 97, 450), "a": samples(now, 100, 100, 100)}, "current", "retained"},
		{"exclude groups direct and removed", map[string][]nodehistory.Point{"nested": samples(now, 100, 100, 100), "direct": samples(now, 100, 100, 100), "gone": samples(now, 100, 100, 100)}, "", "no_candidate"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := Decide(group, list, tt.history, q, now, 15*time.Minute)
			if got != tt.want || reason != tt.reason {
				t.Fatalf("%s %s; want %s %s", got, reason, tt.want, tt.reason)
			}
		})
	}
	points := samples(now, 100, 99, 100)
	points[99].Success = false
	got, _ := Decide(group, list, map[string][]nodehistory.Point{"a": points}, q, now, 15*time.Minute)
	if got != "" {
		t.Fatal("latest failure must veto otherwise qualified node")
	}
	// Ignore old samples outside the configured window.
	points = append(samples(now.Add(-time.Hour), 50, 0, 0), samples(now, 100, 97, 100)...)
	got, _ = Decide(group, list, map[string][]nodehistory.Point{"a": points}, q, now, 15*time.Minute)
	if got != "a" {
		t.Fatal("did not use last configured samples")
	}
}
