// Package subprobe measures how good a subscription actually is, over time.
//
// A subscription's page already says when it was last fetched and how much
// quota is left. Neither answers the question an operator asks before renewing:
// do the nodes in this feed WORK, and how fast are they? That is a property
// only visible over time — one bad afternoon is weather, three bad weeks is the
// provider — so the answer has to be sampled and kept, not computed on demand.
//
// The measurement itself is delegated to sing-box (see prober.go): its Clash
// API exposes a per-outbound URL test that dials through that exact outbound,
// with that outbound's real TLS/transport settings. Re-implementing the dial
// here would mean re-implementing six protocols to get a number that would then
// describe a connection nobody makes.
//
// Storage is deliberately small: ONE aggregate row per subscription per run,
// bounded by both an age and a count (see store_xorm.go). This runs on home
// routers; a metrics feature that fills the overlay is worse than no metrics.
package subprobe

// NodeResult is one node's outcome in one probe run.
//
// Three states, not two. A node that could not be TESTED (Skipped — it is in
// the subscription but not in the running sing-box config, so the Clash API has
// no outbound to dial through) is different from a node that was tested and
// failed, and counting the first as unavailable would blame the provider for
// the operator not having applied the config yet.
type NodeResult struct {
	Tag     string `json:"tag"`
	DelayMs int    `json:"delay_ms,omitempty"`
	// Error is empty when the node answered. It is the reason otherwise, kept
	// verbatim so the drill-down can distinguish a timeout from a refusal.
	Error   string `json:"error,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// Reachable reports whether this node answered the URL test.
func (r NodeResult) Reachable() bool {
	return !r.Skipped && r.Error == ""
}

// Sample is the aggregate of one run, and the row that gets persisted.
type Sample struct {
	Total     int `json:"total"`
	Reachable int `json:"reachable"`
	AvgMs     int `json:"avg_ms"`
	MinMs     int `json:"min_ms"`
	MaxMs     int `json:"max_ms"`
	// Skipped nodes are reported but excluded from Total, so the UI can say
	// "3 nodes could not be tested" instead of silently shrinking the
	// denominator with no explanation.
	Skipped int `json:"skipped,omitempty"`
}

// Availability is the fraction of tested nodes that answered, in [0,1].
func (s Sample) Availability() float64 {
	if s.Total <= 0 {
		return 0
	}
	return float64(s.Reachable) / float64(s.Total)
}

// Aggregate folds one run's per-node results into the row to store.
//
// It returns ok=false when there was nothing to measure. That case must NOT
// become a zero-availability sample: a subscription whose nodes have not been
// imported yet, or one that is entirely absent from the running config, would
// otherwise be drawn as a provider that is 100% down.
//
// The returned Sample still carries Skipped in that case, so a caller can say
// HOW MANY nodes were unmeasurable. Zeroing it made the panel report "no
// outbounds belong to this subscription" for a config that had plenty — just
// none of them applied.
func Aggregate(results []NodeResult) (Sample, bool) {
	var sample Sample
	sum := 0
	// Explicit rather than using MinMs == 0 as "not seen yet": sing-box computes
	// the delay as time.Since(start)/time.Millisecond, so a sub-millisecond
	// result IS 0, and the sentinel would then treat a genuine minimum as unset
	// and let every later, larger value overwrite it.
	seenReachable := false

	for _, r := range results {
		if r.Skipped {
			sample.Skipped++
			continue
		}
		sample.Total++
		if r.Error != "" {
			continue
		}
		sample.Reachable++
		sum += r.DelayMs
		if !seenReachable || r.DelayMs < sample.MinMs {
			sample.MinMs = r.DelayMs
		}
		seenReachable = true
		if r.DelayMs > sample.MaxMs {
			sample.MaxMs = r.DelayMs
		}
	}

	if sample.Total == 0 {
		return Sample{Skipped: sample.Skipped}, false
	}
	if sample.Reachable > 0 {
		// Round rather than truncate: with two nodes at 100ms and 101ms, a
		// truncated mean reads as "100ms" and hides the slower half.
		sample.AvgMs = (sum + sample.Reachable/2) / sample.Reachable
	}
	return sample, true
}
