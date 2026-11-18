package nodehistory

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func TestMembershipPrunesRemovedNodesAndRejectsLateResults(t *testing.T) {
	s, _ := openStore(t, filepath.Join(t.TempDir(), "nodes.db"), func() (time.Duration, int) { return 24 * time.Hour, 10 })
	now := time.Now()
	values := []Measurement{
		{Tag: "keep | sub_a", SubID: "sub_a", At: now, Delay: 100, Success: true},
		{Tag: "gone | sub_a", SubID: "sub_a", At: now, Delay: 200, Success: true},
		{Tag: "other | sub_b", SubID: "sub_b", At: now, Delay: 300, Success: true},
	}
	if err := s.RecordBatch(values); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncSubscription("sub_a", []string{values[0].Tag}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordBatch(values[1:2]); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Series(values[1].Tag); len(got) != 0 {
		t.Fatal("late result resurrected a removed node")
	}
	if got, _ := s.Series(values[0].Tag); len(got) != 1 {
		t.Fatal("unchanged node history lost")
	}
	if got, _ := s.Series(values[2].Tag); len(got) != 1 {
		t.Fatal("another subscription history lost")
	}
	if err := s.SyncSubscription("sub_a", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordBatch(values[:2]); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Count(); n != 1 {
		t.Fatal("deleted subscription history recreated")
	}
	if err := s.SyncSubscription("sub_a", []string{values[1].Tag}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordBatch(values[1:2]); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Series(values[1].Tag); len(got) != 1 {
		t.Fatal("re-added node did not begin fresh history")
	}
}

func TestPolicyReadsTheSameSQLiteConnectionBeforeWriteTransaction(t *testing.T) {
	s, e := openStore(t, filepath.Join(t.TempDir(), "nodes.db"), func() (time.Duration, int) { return 24 * time.Hour, 10 })
	s.bounds = func() (time.Duration, int) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		session := e.Context(ctx)
		defer session.Close()
		if _, err := session.Count(new(Target)); err != nil {
			t.Errorf("policy read inside transaction: %v", err)
		}
		return 24 * time.Hour, 10
	}
	if err := s.RecordBatch([]Measurement{{Tag: "node", At: time.Now(), Delay: 1, Success: true}}); err != nil {
		t.Fatal(err)
	}
	if err := s.TrimAll(); err != nil {
		t.Fatal(err)
	}
}

func openStore(t *testing.T, path string, bounds func() (time.Duration, int)) (*Store, *xorm.Engine) {
	t.Helper()
	e, err := xorm.NewEngine("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	e.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = e.Close() })
	s := NewStore(e, bounds)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	return s, e
}

func TestDurableHistoryAndBoundedRecentQueries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nodes.db")
	policy := func() (time.Duration, int) { return 7 * 24 * time.Hour, 10 }
	s, e := openStore(t, path, policy)
	now := time.Now()
	values := []Measurement{
		{Tag: "🇰🇷 Korea / 01 | sub_a", SubID: "sub_a", TestURL: "https://a.test/204", At: now, Delay: 120, Success: true, Source: "subscription"},
		{Tag: "🇰🇷 Korea / 01 | sub_a", SubID: "sub_a", TestURL: "https://a.test/204", At: now, Delay: 0, Success: false, Source: "manual"},
		{Tag: "🇰🇷 Korea / 01 | sub_a", SubID: "sub_a", TestURL: "https://b.test/204", At: now, Delay: 0, Success: true, Source: "manual-group"},
		{Tag: "other", At: now, Delay: 90, Success: true},
	}
	if err := s.RecordBatch(values); err != nil {
		t.Fatal(err)
	}
	recent, err := s.Recent([]string{values[0].Tag, "other", "missing"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := recent[values[0].Tag]; len(got) != 2 || got[0].Source != "manual" || got[0].Success || !got[1].Success || got[1].TestURL != "https://b.test/204" {
		t.Fatalf("recent=%+v", got)
	}
	if len(recent["other"]) != 1 || len(recent["missing"]) != 0 {
		t.Fatalf("cross-node leakage: %+v", recent)
	}
	if n, err := e.Count(new(Target)); err != nil || n != 3 {
		t.Fatalf("normalized targets=%d err=%v", n, err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := openStore(t, path, policy)
	points, err := reopened.Series(values[0].Tag)
	if err != nil || len(points) != 3 {
		t.Fatalf("history lost after reopen: %+v %v", points, err)
	}
}

func TestRetentionAndAtomicBatch(t *testing.T) {
	limit := 3
	age := 24 * time.Hour
	s, _ := openStore(t, filepath.Join(t.TempDir(), "nodes.db"), func() (time.Duration, int) { return age, limit })
	now := time.Now()
	var values []Measurement
	for i := range 6 {
		values = append(values, Measurement{Tag: "node", At: now.Add(time.Duration(i-5) * time.Minute), Delay: i, Success: true})
	}
	if err := s.RecordBatch(values); err != nil {
		t.Fatal(err)
	}
	points, err := s.Series("node")
	if err != nil || len(points) != 3 || points[0].Delay != 3 {
		t.Fatalf("count retention: %+v %v", points, err)
	}
	if err := s.RecordBatch([]Measurement{{Tag: "new", At: now, Delay: 99}, {Tag: "", At: now}}); err == nil {
		t.Fatal("invalid batch accepted")
	}
	if got, _ := s.Series("new"); len(got) != 0 {
		t.Fatal("partial batch committed")
	}
	limit = 1
	if err := s.TrimAll(); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Series("node"); len(got) != 1 || got[0].Delay != 5 {
		t.Fatalf("tightened retention: %+v", got)
	}
	if err := s.RecordBatch([]Measurement{{Tag: "old", At: now.Add(-48 * time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Count(); n != 1 {
		t.Fatalf("expired row not trimmed: %d", n)
	}
}
