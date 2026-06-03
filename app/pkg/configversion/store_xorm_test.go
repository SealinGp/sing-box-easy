package configversion

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
	dir, err := os.MkdirTemp("", "configversion_test")
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
	if _, err := e.Exec("DELETE FROM config_versions"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

// TestDeleteBatch removes multiple ids in one call and ignores missing ones.
func TestDeleteBatch(t *testing.T) {
	s := newStore(t)
	id1, _ := s.Save([]byte(`{"a":1}`))
	id2, _ := s.Save([]byte(`{"b":2}`))
	id3, _ := s.Save([]byte(`{"c":3}`))

	deleted, err := s.DeleteBatch([]int64{id1, id3, 99999})
	if err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2 (missing id ignored)", deleted)
	}
	rows, _ := s.List()
	if len(rows) != 1 || rows[0].ID != id2 {
		t.Errorf("remaining = %v, want only id %d", rows, id2)
	}

	// Empty input is a no-op.
	if n, err := s.DeleteBatch(nil); err != nil || n != 0 {
		t.Errorf("empty batch = (%d,%v), want (0,nil)", n, err)
	}
}

// TestDeleteOlderThan removes only rows aged beyond the window.
func TestDeleteOlderThan(t *testing.T) {
	s := newStore(t)
	oldID, _ := s.Save([]byte(`{"old":1}`))
	newID, _ := s.Save([]byte(`{"new":1}`))

	// Age the first row to 70 days ago. Raw SQL with xorm's datetime layout so
	// the lexicographic comparison in DeleteOlderThan matches; the 10-day margin
	// (70d vs the 60d window) absorbs any timezone nuance.
	e, _ := database.GetEngine()
	agedStr := time.Now().Add(-70 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := e.Exec("UPDATE config_versions SET created_at = ? WHERE id = ?", agedStr, oldID); err != nil {
		t.Fatalf("age row: %v", err)
	}

	// Non-positive maxAge is a guard no-op.
	if n, err := s.DeleteOlderThan(0); err != nil || n != 0 {
		t.Errorf("DeleteOlderThan(0) = (%d,%v), want (0,nil)", n, err)
	}

	deleted, err := s.DeleteOlderThan(60 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("delete older than: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1 (only the 70-day-old row)", deleted)
	}
	rows, _ := s.List()
	if len(rows) != 1 || rows[0].ID != newID {
		t.Errorf("remaining = %v, want only the recent id %d", rows, newID)
	}
}
