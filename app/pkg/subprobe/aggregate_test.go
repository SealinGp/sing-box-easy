package subprobe

import "testing"

// TestAggregate covers the arithmetic the whole feature rests on: availability
// is reachable/total, and latency is averaged over the REACHABLE nodes only.
func TestAggregate(t *testing.T) {
	tests := []struct {
		name    string
		results []NodeResult
		want    Sample
		wantOK  bool
	}{
		{
			// The example from the feature request: 300ms, 500ms, unreachable
			// => 2/3 available, 400ms average.
			name: "mixed reachable and unreachable",
			results: []NodeResult{
				{Tag: "a", DelayMs: 300},
				{Tag: "b", DelayMs: 500},
				{Tag: "c", Error: "timeout"},
			},
			want:   Sample{Total: 3, Reachable: 2, AvgMs: 400, MinMs: 300, MaxMs: 500},
			wantOK: true,
		},
		{
			// A wholly dead subscription is a real, recordable observation:
			// 0% available. Latency must stay 0 rather than inventing a number
			// out of an empty set.
			name: "all unreachable",
			results: []NodeResult{
				{Tag: "a", Error: "timeout"},
				{Tag: "b", Error: "connection refused"},
			},
			want:   Sample{Total: 2, Reachable: 0, AvgMs: 0, MinMs: 0, MaxMs: 0},
			wantOK: true,
		},
		{
			name:    "all reachable",
			results: []NodeResult{{Tag: "a", DelayMs: 120}},
			want:    Sample{Total: 1, Reachable: 1, AvgMs: 120, MinMs: 120, MaxMs: 120},
			wantOK:  true,
		},
		{
			// No nodes at all is NOT a 0% sample — there was nothing to
			// measure. Recording one would draw a subscription whose nodes
			// have not been imported yet as a failing provider.
			name:    "no nodes yields no sample",
			results: nil,
			wantOK:  false,
		},
		{
			// Skipped nodes (present in the feed, absent from the running
			// sing-box) must not count against availability: the panel could
			// not test them, which is not evidence that they are down.
			name: "skipped nodes are excluded from totals",
			results: []NodeResult{
				{Tag: "a", DelayMs: 200},
				{Tag: "b", Skipped: true},
			},
			want:   Sample{Total: 1, Reachable: 1, AvgMs: 200, MinMs: 200, MaxMs: 200, Skipped: 1},
			wantOK: true,
		},
		{
			// Averages are integers; the store holds milliseconds. Rounding,
			// not truncation, so 100/101 does not read as 100.
			name: "average rounds to nearest",
			results: []NodeResult{
				{Tag: "a", DelayMs: 100},
				{Tag: "b", DelayMs: 101},
			},
			want:   Sample{Total: 2, Reachable: 2, AvgMs: 101, MinMs: 100, MaxMs: 101},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Aggregate(tt.results)
			if ok != tt.wantOK {
				t.Fatalf("Aggregate ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got != tt.want {
				t.Errorf("Aggregate = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestAggregateReportsSkippedWhenNothingMeasurable — the count survives even
// though there is no sample to store, so the panel can say "34 nodes are not
// in the running sing-box" instead of "this subscription has no nodes".
func TestAggregateReportsSkippedWhenNothingMeasurable(t *testing.T) {
	got, ok := Aggregate([]NodeResult{
		{Tag: "a", Skipped: true},
		{Tag: "b", Skipped: true},
	})
	if ok {
		t.Fatal("Aggregate ok = true, want false (nothing was measured)")
	}
	if got.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", got.Skipped)
	}
	if got.Total != 0 || got.Reachable != 0 {
		t.Errorf("counts = %+v, want zeroed", got)
	}
}

// TestAggregateHandlesZeroMillisecondDelay — sing-box computes its delay as
// time.Since(start)/time.Millisecond, so a sub-millisecond result is literally
// 0. An earlier version used MinMs == 0 to mean "no minimum recorded yet",
// which made a genuine 0ms reading look unset and let every later, larger
// value overwrite it.
func TestAggregateHandlesZeroMillisecondDelay(t *testing.T) {
	got, ok := Aggregate([]NodeResult{
		{Tag: "fast", DelayMs: 0},
		{Tag: "slow", DelayMs: 500},
	})
	if !ok {
		t.Fatal("Aggregate ok = false, want true")
	}
	if got.MinMs != 0 {
		t.Errorf("MinMs = %d, want 0 (a real measurement, not an unset sentinel)", got.MinMs)
	}
	if got.MaxMs != 500 {
		t.Errorf("MaxMs = %d, want 500", got.MaxMs)
	}
	if got.Reachable != 2 {
		t.Errorf("Reachable = %d, want 2", got.Reachable)
	}
}

// TestSampleAvailability pins the ratio helper the API and chart both read.
func TestSampleAvailability(t *testing.T) {
	if got := (Sample{Total: 3, Reachable: 2}).Availability(); got != 2.0/3.0 {
		t.Errorf("Availability = %v, want 2/3", got)
	}
	// A zero total must not divide by zero.
	if got := (Sample{}).Availability(); got != 0 {
		t.Errorf("Availability of empty = %v, want 0", got)
	}
}
