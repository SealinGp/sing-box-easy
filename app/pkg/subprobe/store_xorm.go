package subprobe

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subprobe/repo"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// Point is one plotted observation — a Sample with the time it was taken.
// Series returns these already bucketed; the chart plots them as-is.
type Point struct {
	At time.Time `json:"at"`
	Sample
}

// StoreXORM persists probe samples.
type StoreXORM struct {
	e *xorm.Engine
}

// NewStoreXORM creates a new XORM-backed probe sample store.
func NewStoreXORM() *StoreXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}
	return &StoreXORM{e: e}
}

// Init ensures the subscription_probe_samples table exists.
func (s *StoreXORM) Init() error {
	if err := s.e.Sync2(new(repo.ProbeSample)); err != nil {
		logger.Error("Failed to sync subscription_probe_samples table", zap.Error(err))
		return err
	}
	logger.Info("Subscription probe store initialized with XORM")
	return nil
}

// Insert records one run's aggregate for one subscription.
func (s *StoreXORM) Insert(subID string, at time.Time, sample Sample) error {
	row := &repo.ProbeSample{
		SubID:     subID,
		At:        at.Unix(),
		Total:     sample.Total,
		Reachable: sample.Reachable,
		AvgMs:     sample.AvgMs,
		MinMs:     sample.MinMs,
		MaxMs:     sample.MaxMs,
	}
	if _, err := s.e.Insert(row); err != nil {
		return fmt.Errorf("failed to insert probe sample for %q: %w", subID, err)
	}
	return nil
}

// Series returns one subscription's samples in [from, to), oldest first.
//
// A non-zero `bucket` collapses the rows into fixed-width time buckets. That is
// not cosmetic: at the 1-minute floor a 30-day window is 43k rows, and shipping
// those to a chart that is ~600px wide would spend megabytes to draw pixels
// that overlap. The caller picks a bucket from the requested range so the wire
// cost is bounded by the range, not by how densely the operator sampled it.
//
// Bucket arithmetic is deliberately NOT a mean of means:
//
//   - availability is sum(reachable)/sum(total), so a 100-node run outweighs a
//     4-node one, which is what "how much of this feed worked" means;
//   - latency is weighted by the number of nodes each row averaged over, so the
//     result is the true mean over nodes rather than over runs;
//   - a run where nothing answered contributes no latency at all. Folding its
//     zero in would drag the average toward zero and draw a dead hour as the
//     FASTEST one on the chart.
func (s *StoreXORM) Series(subID string, from, to time.Time, bucket time.Duration) ([]Point, error) {
	if bucket <= 0 {
		return s.rawSeries(subID, from, to)
	}

	width := int64(bucket / time.Second)
	if width <= 0 {
		return s.rawSeries(subID, from, to)
	}

	// SUM(avg_ms * reachable) / SUM(reachable) is the node-weighted mean. The
	// NULLIF guards the all-dead bucket, where SUM(reachable) is 0 — SQLite
	// would otherwise return NULL for the whole expression anyway, but naming
	// the case makes the intent readable and the COALESCE explicit.
	const query = `
SELECT (at / ?) * ? AS bucket_at,
       SUM(total)                                                    AS total,
       SUM(reachable)                                                AS reachable,
       CAST(COALESCE(SUM(avg_ms * reachable) / NULLIF(SUM(reachable), 0), 0) AS INTEGER) AS avg_ms,
       COALESCE(MIN(NULLIF(min_ms, 0)), 0)                           AS min_ms,
       MAX(max_ms)                                                   AS max_ms
FROM subscription_probe_samples
WHERE sub_id = ? AND at >= ? AND at < ?
GROUP BY bucket_at
ORDER BY bucket_at ASC`

	rows, err := s.e.QueryInterface(query, width, width, subID, from.Unix(), to.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to read probe series for %q: %w", subID, err)
	}

	points := make([]Point, 0, len(rows))
	for _, row := range rows {
		points = append(points, Point{
			At: time.Unix(asInt64(row["bucket_at"]), 0),
			Sample: Sample{
				Total:     int(asInt64(row["total"])),
				Reachable: int(asInt64(row["reachable"])),
				AvgMs:     int(asInt64(row["avg_ms"])),
				MinMs:     int(asInt64(row["min_ms"])),
				MaxMs:     int(asInt64(row["max_ms"])),
			},
		})
	}
	return points, nil
}

// rawSeries returns every stored row in the window, unbucketed.
func (s *StoreXORM) rawSeries(subID string, from, to time.Time) ([]Point, error) {
	var rows []repo.ProbeSample
	err := s.e.Where("sub_id = ? AND at >= ? AND at < ?", subID, from.Unix(), to.Unix()).
		Asc("at").Find(&rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read probe series for %q: %w", subID, err)
	}

	points := make([]Point, 0, len(rows))
	for _, r := range rows {
		points = append(points, Point{
			At: time.Unix(r.At, 0),
			Sample: Sample{
				Total:     r.Total,
				Reachable: r.Reachable,
				AvgMs:     r.AvgMs,
				MinMs:     r.MinMs,
				MaxMs:     r.MaxMs,
			},
		})
	}
	return points, nil
}

// Latest returns the newest sample for every subscription that has one, keyed
// by subscription id. One query rather than one per subscription: the list page
// and the Overview card both want the whole set at once.
func (s *StoreXORM) Latest() (map[string]Point, error) {
	// The inner MAX(at) per sub_id is joined back to pick that row. Ties (two
	// runs in the same second, which the 1-minute interval floor makes
	// impossible in practice) collapse harmlessly to one row.
	const query = `
SELECT p.sub_id, p.at, p.total, p.reachable, p.avg_ms, p.min_ms, p.max_ms
FROM subscription_probe_samples p
JOIN (SELECT sub_id, MAX(at) AS at FROM subscription_probe_samples GROUP BY sub_id) m
  ON m.sub_id = p.sub_id AND m.at = p.at`

	rows, err := s.e.QueryInterface(query)
	if err != nil {
		return nil, fmt.Errorf("failed to read latest probe samples: %w", err)
	}

	out := make(map[string]Point, len(rows))
	for _, row := range rows {
		subID, _ := row["sub_id"].(string)
		if subID == "" {
			// QueryInterface returns []byte for text columns on some drivers.
			if raw, ok := row["sub_id"].([]byte); ok {
				subID = string(raw)
			}
		}
		if subID == "" {
			continue
		}
		out[subID] = Point{
			At: time.Unix(asInt64(row["at"]), 0),
			Sample: Sample{
				Total:     int(asInt64(row["total"])),
				Reachable: int(asInt64(row["reachable"])),
				AvgMs:     int(asInt64(row["avg_ms"])),
				MinMs:     int(asInt64(row["min_ms"])),
				MaxMs:     int(asInt64(row["max_ms"])),
			},
		}
	}
	return out, nil
}

// Trim enforces the two retention bounds for ONE subscription and returns how
// many rows it removed. Both are optional (a non-positive value disables that
// bound), and both are applied — whichever bites first wins.
//
// Per-subscription rather than global on purpose: a global "keep N rows" cap
// would let a subscription with many nodes and a short interval evict the
// history of a quiet one, and the chart the operator opened would be empty for
// reasons nothing on screen could explain.
func (s *StoreXORM) Trim(subID string, maxAge time.Duration, keepN int) (int64, error) {
	var deleted int64

	if maxAge > 0 {
		cutoff := time.Now().Add(-maxAge).Unix()
		n, err := s.e.Where("sub_id = ? AND at < ?", subID, cutoff).Delete(new(repo.ProbeSample))
		if err != nil {
			return deleted, fmt.Errorf("failed to trim probe samples by age for %q: %w", subID, err)
		}
		deleted += n
	}

	if keepN > 0 {
		// Find the timestamp of the oldest row that must survive, then delete
		// everything strictly older. Using `at` (not `id`) as the boundary
		// keeps this correct if rows are ever backfilled out of id order.
		var boundary []repo.ProbeSample
		err := s.e.Cols("at").Where("sub_id = ?", subID).Desc("at").Limit(1, keepN-1).Find(&boundary)
		if err != nil {
			return deleted, fmt.Errorf("failed to find trim boundary for %q: %w", subID, err)
		}
		if len(boundary) > 0 {
			n, err := s.e.Where("sub_id = ? AND at < ?", subID, boundary[0].At).Delete(new(repo.ProbeSample))
			if err != nil {
				return deleted, fmt.Errorf("failed to trim probe samples by count for %q: %w", subID, err)
			}
			deleted += n
		}
	}

	return deleted, nil
}

// DeleteSubscription removes all history for one subscription. Called when the
// subscription itself is deleted, so a re-added feed does not inherit numbers
// measured against a different provider.
func (s *StoreXORM) DeleteSubscription(subID string) error {
	if _, err := s.e.Where("sub_id = ?", subID).Delete(new(repo.ProbeSample)); err != nil {
		return fmt.Errorf("failed to delete probe samples for %q: %w", subID, err)
	}
	return nil
}

// CountAll returns the total number of stored samples — shown in Settings next
// to the retention knobs, so the disk cost of the current configuration is a
// measured figure rather than an estimate the operator has to trust.
func (s *StoreXORM) CountAll() (int64, error) {
	n, err := s.e.Count(new(repo.ProbeSample))
	if err != nil {
		return 0, fmt.Errorf("failed to count probe samples: %w", err)
	}
	return n, nil
}

// asInt64 narrows whatever the driver handed back for a numeric column.
// QueryInterface is untyped, and modernc/sqlite returns int64 for aggregates
// but []byte for some expressions depending on the query shape.
func asInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case []byte:
		return parseInt64(string(n))
	case string:
		return parseInt64(n)
	default:
		return 0
	}
}

func parseInt64(s string) int64 {
	var out int64
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			// Stop at a decimal point or any trailing junk: an average that
			// arrived as "400.0" is still 400.
			break
		}
		out = out*10 + int64(c-'0')
	}
	if neg {
		return -out
	}
	return out
}
