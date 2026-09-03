package trafficflow

import (
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
)

// Differ derives per-connection rates between consecutive snapshots.
//
// This is the same diff zashboard does in the browser
// (assembly/connections/clash.ts), with one correction: the rate is divided by
// the wall-clock seconds between samples, not assumed to be per tick. A poll
// that arrives late — the panel host busy, the router swapping — would
// otherwise report two seconds of bytes as one second of speed.
type Differ struct {
	previous   map[string]sample
	previousAt time.Time
}

type sample struct {
	up, down int64
}

// NewDiffer starts with no baseline: the first Step marks every connection
// Fresh.
func NewDiffer() *Differ {
	return &Differ{previous: map[string]sample{}}
}

// Step folds in one snapshot and returns the live view plus how many
// connections from the previous sample are gone.
//
// A counter that decreased is clamped to zero. It means sing-box restarted and
// reused an id, or the counter wrapped; a negative rate would draw a ribbon
// flowing backwards, which is a lie in a direction nobody would think to doubt.
func (d *Differ) Step(at time.Time, snapshot *clashapi.Snapshot) ([]Live, int) {
	elapsed := at.Sub(d.previousAt).Seconds()
	hasBaseline := !d.previousAt.IsZero() && elapsed > 0

	current := make(map[string]sample, len(snapshot.Connections))
	live := make([]Live, 0, len(snapshot.Connections))

	for _, conn := range snapshot.Connections {
		current[conn.ID] = sample{up: conn.Upload, down: conn.Download}

		before, seen := d.previous[conn.ID]
		if !seen || !hasBaseline {
			live = append(live, Live{Connection: conn, Fresh: true})
			continue
		}
		live = append(live, Live{
			Connection: conn,
			DownRate:   rate(conn.Download-before.down, elapsed),
			UpRate:     rate(conn.Upload-before.up, elapsed),
		})
	}

	closed := 0
	for id := range d.previous {
		if _, still := current[id]; !still {
			closed++
		}
	}

	// Replace, never mutate: the old map is what a concurrent reader of the
	// previous Step's result would still be looking at.
	d.previous = current
	d.previousAt = at
	return live, closed
}

func rate(delta int64, seconds float64) float64 {
	if delta <= 0 {
		return 0
	}
	return float64(delta) / seconds
}
