package subprobe

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

// TestMain initializes the logger and a single process-wide SQLite database
// (database.Init is guarded by sync.Once).
func TestMain(m *testing.M) {
	logger.InitDefault()
	dir, err := os.MkdirTemp("", "subprobe_test")
	if err != nil {
		panic(err)
	}
	if err := database.Init(filepath.Join(dir, "test.db")); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = database.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func newStore(t *testing.T) *StoreXORM {
	t.Helper()
	s := NewStoreXORM()
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM subscription_probe_samples"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

// TestInsertAndSeries round-trips samples and returns them oldest-first, which
// is the order a chart consumes.
func TestInsertAndSeries(t *testing.T) {
	s := newStore(t)
	base := time.Unix(1_700_000_000, 0)

	if err := s.Insert("subA", base, Sample{Total: 3, Reachable: 2, AvgMs: 400, MinMs: 300, MaxMs: 500}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := s.Insert("subA", base.Add(time.Hour), Sample{Total: 3, Reachable: 3, AvgMs: 200, MinMs: 100, MaxMs: 300}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// A second subscription must never leak into the first one's series.
	if err := s.Insert("subB", base, Sample{Total: 1, Reachable: 0}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	points, err := s.Series("subA", base.Add(-time.Minute), base.Add(2*time.Hour), 0)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2 (subB must not leak)", len(points))
	}
	if !points[0].At.Before(points[1].At) {
		t.Errorf("points are not oldest-first: %v then %v", points[0].At, points[1].At)
	}
	if points[0].AvgMs != 400 || points[1].Reachable != 3 {
		t.Errorf("round-trip mismatch: %+v", points)
	}
}

// TestSeriesWindow excludes samples outside the requested range — the range
// picker is the only thing keeping a 30-day store off a 1-hour chart.
func TestSeriesWindow(t *testing.T) {
	s := newStore(t)
	base := time.Unix(1_700_000_000, 0)
	_ = s.Insert("subA", base.Add(-48*time.Hour), Sample{Total: 1, Reachable: 1, AvgMs: 999})
	_ = s.Insert("subA", base, Sample{Total: 1, Reachable: 1, AvgMs: 100})

	points, err := s.Series("subA", base.Add(-time.Hour), base.Add(time.Hour), 0)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(points) != 1 || points[0].AvgMs != 100 {
		t.Fatalf("window not applied: %+v", points)
	}
}

// TestSeriesBucketing is what keeps a month of history off the wire: many rows
// collapse into one point per bucket.
//
// The arithmetic matters as much as the count. Availability over a bucket is
// sum(reachable)/sum(total) — NOT the mean of each row's percentage, which
// would weight a 1-node run the same as a 100-node one. Latency is likewise
// weighted by how many nodes each row averaged over.
func TestSeriesBucketing(t *testing.T) {
	s := newStore(t)
	base := time.Unix(1_700_000_000, 0).Truncate(time.Hour)

	// Two runs inside one hour, one run in the next.
	_ = s.Insert("subA", base.Add(5*time.Minute), Sample{Total: 10, Reachable: 10, AvgMs: 100, MinMs: 50, MaxMs: 150})
	_ = s.Insert("subA", base.Add(35*time.Minute), Sample{Total: 10, Reachable: 0, AvgMs: 0})
	_ = s.Insert("subA", base.Add(65*time.Minute), Sample{Total: 4, Reachable: 2, AvgMs: 300, MinMs: 300, MaxMs: 300})

	points, err := s.Series("subA", base, base.Add(2*time.Hour), time.Hour)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2 buckets", len(points))
	}

	// Bucket 1: 20 tested, 10 reachable => 50%. Latency averages only the ten
	// nodes that answered, so it stays 100ms — the all-dead run contributes no
	// latency at all rather than dragging the mean to 50.
	if points[0].Total != 20 || points[0].Reachable != 10 {
		t.Errorf("bucket 1 counts = %d/%d, want 10/20", points[0].Reachable, points[0].Total)
	}
	if points[0].AvgMs != 100 {
		t.Errorf("bucket 1 avg = %d, want 100 (weighted by reachable nodes)", points[0].AvgMs)
	}
	if points[0].MinMs != 50 || points[0].MaxMs != 150 {
		t.Errorf("bucket 1 min/max = %d/%d, want 50/150", points[0].MinMs, points[0].MaxMs)
	}
	if points[1].Total != 4 || points[1].AvgMs != 300 {
		t.Errorf("bucket 2 = %+v", points[1])
	}
}

// TestLatest is what the subscription list and Overview card read: one row,
// the newest, per subscription.
func TestLatest(t *testing.T) {
	s := newStore(t)
	base := time.Unix(1_700_000_000, 0)
	_ = s.Insert("subA", base, Sample{Total: 3, Reachable: 1, AvgMs: 900})
	_ = s.Insert("subA", base.Add(time.Hour), Sample{Total: 3, Reachable: 3, AvgMs: 120})
	_ = s.Insert("subB", base, Sample{Total: 2, Reachable: 2, AvgMs: 50})

	latest, err := s.Latest()
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("len(latest) = %d, want 2", len(latest))
	}
	if latest["subA"].AvgMs != 120 {
		t.Errorf("subA latest avg = %d, want 120 (newest row wins)", latest["subA"].AvgMs)
	}
	if latest["subB"].Reachable != 2 {
		t.Errorf("subB latest = %+v", latest["subB"])
	}
}

// TestTrimByAge enforces the retention window.
func TestTrimByAge(t *testing.T) {
	s := newStore(t)
	now := time.Now()
	_ = s.Insert("subA", now.Add(-10*24*time.Hour), Sample{Total: 1, Reachable: 1})
	_ = s.Insert("subA", now.Add(-1*time.Hour), Sample{Total: 1, Reachable: 1})

	deleted, err := s.Trim("subA", 7*24*time.Hour, 0)
	if err != nil {
		t.Fatalf("trim: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	points, _ := s.Series("subA", now.Add(-30*24*time.Hour), now.Add(time.Hour), 0)
	if len(points) != 1 {
		t.Errorf("remaining = %d, want 1", len(points))
	}
}

// TestTrimByCount enforces the point cap — the bound that actually protects a
// small disk, since an operator who shortens the interval multiplies the rows
// inside any retention window.
func TestTrimByCount(t *testing.T) {
	s := newStore(t)
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 10; i++ {
		_ = s.Insert("subA", base.Add(time.Duration(i)*time.Minute), Sample{Total: 1, Reachable: 1, AvgMs: i})
	}
	// Another subscription's rows must not count toward this one's cap.
	for i := 0; i < 5; i++ {
		_ = s.Insert("subB", base.Add(time.Duration(i)*time.Minute), Sample{Total: 1, Reachable: 1})
	}

	deleted, err := s.Trim("subA", 0, 4)
	if err != nil {
		t.Fatalf("trim: %v", err)
	}
	if deleted != 6 {
		t.Errorf("deleted = %d, want 6", deleted)
	}

	points, _ := s.Series("subA", base.Add(-time.Hour), time.Now().Add(time.Hour), 0)
	if len(points) != 4 {
		t.Fatalf("remaining = %d, want 4", len(points))
	}
	// The NEWEST four must survive, not an arbitrary four.
	if points[0].AvgMs != 6 || points[3].AvgMs != 9 {
		t.Errorf("kept the wrong rows: %+v", points)
	}

	if bPoints, _ := s.Series("subB", base.Add(-time.Hour), time.Now().Add(time.Hour), 0); len(bPoints) != 5 {
		t.Errorf("subB rows = %d, want 5 (untouched)", len(bPoints))
	}
}

// TestDeleteSubscription removes a deleted subscription's history, so a feed
// that was removed and re-added under a new id does not inherit old numbers.
func TestDeleteSubscription(t *testing.T) {
	s := newStore(t)
	_ = s.Insert("subA", time.Now(), Sample{Total: 1, Reachable: 1})
	if err := s.DeleteSubscription("subA"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	latest, _ := s.Latest()
	if _, ok := latest["subA"]; ok {
		t.Error("subA still present after delete")
	}
}
