// Package autoselect chooses Selector members from durable quality observations.
package autoselect

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

type candidate struct {
	tag                    string
	passed, successes, p95 int
}

// Decide holds a qualified incumbent. It never selects a nested group or an
// unmeasured/stale node. Below the watermark, it compares measured alternatives.
func Decide(group clashapi.Proxy, list []clashapi.Proxy, history map[string][]nodehistory.Point, rules settings.NodeQuality, now time.Time, maxAge time.Duration) (string, string) {
	known := map[string]clashapi.Proxy{}
	for _, p := range list {
		known[p.Name] = p
	}
	var candidates []candidate
	bestPassed := 0
	for _, tag := range group.All {
		p, exists := known[tag]
		if !exists || len(p.All) > 0 {
			continue
		}
		switch strings.ToLower(p.Type) {
		case "", "selector", "urltest", "fallback", "direct", "reject", "block", "dns":
			continue
		}
		points := history[tag]
		if len(points) < rules.Samples {
			continue
		}
		points = points[len(points)-rules.Samples:]
		last := points[len(points)-1]
		if !last.Success || last.Delay <= 0 || now.Sub(last.Time) > maxAge || last.Time.After(now.Add(time.Minute)) {
			continue
		}
		passed := 0
		var delays []int
		for _, point := range points {
			if point.Success && point.Delay > 0 {
				delays = append(delays, point.Delay)
				if point.Delay <= rules.MaxDelayMs {
					passed++
				}
			}
		}
		bestPassed = max(bestPassed, passed)
		if tag == group.Now && passed*100 >= rules.Samples*rules.GoodPercent {
			return tag, "retained"
		}
		sort.Ints(delays)
		candidates = append(candidates, candidate{tag: tag, passed: passed, successes: len(delays), p95: delays[(len(delays)*95+99)/100-1]})
	}
	if len(candidates) == 0 {
		return "", "no_candidate"
	}
	fallback := bestPassed*100 < rules.Samples*rules.GoodPercent
	if fallback {
		// Anchor the band to the best rate; pairwise tolerances are not transitive.
		candidates = slices.DeleteFunc(candidates, func(c candidate) bool {
			return (bestPassed-c.passed)*100 > rules.Samples || (bestPassed > 0 && c.passed == 0)
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if !fallback && a.passed != b.passed {
			return a.passed > b.passed
		}
		if bestPassed == 0 && a.successes != b.successes {
			return a.successes > b.successes
		}
		if a.p95 != b.p95 {
			return a.p95 < b.p95
		}
		if fallback && (a.tag == group.Now || b.tag == group.Now) {
			return a.tag == group.Now && b.tag != group.Now
		}
		return a.tag < b.tag
	})
	best := candidates[0].tag
	if fallback {
		if best == group.Now {
			return best, "fallback_retained"
		}
		return best, "fallback_switched"
	}
	return best, "switched"
}
