package model

import "time"

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

type ProbePoint struct {
	At time.Time `json:"at"`
	Sample
}
