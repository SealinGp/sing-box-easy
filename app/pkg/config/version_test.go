package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// fakeStore is an in-memory, thread-safe VersionStore for testing the Manager's
// version behavior without a database.
type fakeStore struct {
	mu     sync.Mutex
	rows   map[int64][]byte
	order  []int64 // insertion order (oldest first)
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{rows: map[int64][]byte{}, nextID: 1}
}

func (s *fakeStore) Save(content []byte) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	cp := make([]byte, len(content))
	copy(cp, content)
	s.rows[id] = cp
	s.order = append(s.order, id)
	return id, nil
}

func (s *fakeStore) List() ([]VersionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]VersionInfo, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- { // newest first
		id := s.order[i]
		out = append(out, VersionInfo{ID: id, Size: len(s.rows[id])})
	}
	return out, nil
}

func (s *fakeStore) Get(id int64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.rows[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return c, nil
}

func (s *fakeStore) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[id]; !ok {
		return ErrVersionNotFound
	}
	delete(s.rows, id)
	for i, v := range s.order {
		if v == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return nil
}

func (s *fakeStore) Prune(keep int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.order) > keep {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.rows, oldest)
	}
	return nil
}

// newTestManager builds a Manager pointed at a temp config file with a fake
// store attached. sing-box is not invoked (we test version bookkeeping only).
func newTestManager(t *testing.T, store VersionStore, keep int) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	m := NewManager(cfgPath, "sing-box", "")
	m.SetVersionStore(store)
	m.SetKeepVersions(keep)
	return m, cfgPath
}

func TestSnapshotAndPrune(t *testing.T) {
	store := newFakeStore()
	m, cfgPath := newTestManager(t, store, 3)

	// Simulate 5 saves: each writes the live config then snapshots it.
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(cfgPath, []byte(`{"v":`+string(rune('0'+i))+`}`), 0600); err != nil {
			t.Fatal(err)
		}
		m.snapshotCurrent()
	}

	versions, err := m.ListVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 3 {
		t.Fatalf("expected 3 retained versions, got %d", len(versions))
	}
	// Newest first: ids should be 5,4,3 (1 and 2 pruned).
	if versions[0].ID != 5 || versions[2].ID != 3 {
		t.Fatalf("unexpected version ordering: %+v", versions)
	}
}

func TestDeleteVersion(t *testing.T) {
	store := newFakeStore()
	m, cfgPath := newTestManager(t, store, 10)

	// Create 3 versions.
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(cfgPath, []byte(`{"v":`+string(rune('0'+i))+`}`), 0600); err != nil {
			t.Fatal(err)
		}
		m.snapshotCurrent()
	}

	// Delete the middle one (id 2); only 1 and 3 should remain.
	if err := m.DeleteVersion(2); err != nil {
		t.Fatalf("DeleteVersion(2) failed: %v", err)
	}
	versions, err := m.ListVersions()
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions after delete, got %d", len(versions))
	}
	for _, v := range versions {
		if v.ID == 2 {
			t.Fatalf("version 2 should have been deleted, still present: %+v", versions)
		}
	}

	// Deleting a non-existent id reports ErrVersionNotFound.
	if err := m.DeleteVersion(999); !errors.Is(err, ErrVersionNotFound) {
		t.Fatalf("expected ErrVersionNotFound for missing id, got %v", err)
	}
}

func TestSetKeepVersionsClamp(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	m.SetKeepVersions(0)
	if got := m.keepVersions.Load(); got != int64(minKeepVersions) {
		t.Errorf("keep below min not clamped: got %d", got)
	}
	m.SetKeepVersions(99999)
	if got := m.keepVersions.Load(); got != int64(maxKeepVersions) {
		t.Errorf("keep above max not clamped: got %d", got)
	}
}

func TestRollbackToVersionRestoresAndSnapshotsCurrent(t *testing.T) {
	store := newFakeStore()
	m, cfgPath := newTestManager(t, store, 10)

	// RollbackToVersion parses with the strict sing-box registry, so fixtures
	// must be valid configs.
	const v1 = `{"log":{"level":"info"}}`
	const cur = `{"log":{"level":"debug"}}`

	// v1 saved to history, then the live config becomes "current".
	if err := os.WriteFile(cfgPath, []byte(v1), 0600); err != nil {
		t.Fatal(err)
	}
	m.snapshotCurrent() // id=1 holds v1
	if err := os.WriteFile(cfgPath, []byte(cur), 0600); err != nil {
		t.Fatal(err)
	}

	// Roll back to id=1: current should be snapshotted, live becomes v1.
	if err := m.RollbackToVersion(1); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	live, _ := os.ReadFile(cfgPath)
	if string(live) != v1 {
		t.Fatalf("live config not restored, got %q", string(live))
	}

	// The pre-rollback current must now be in history (rollback is reversible).
	versions, _ := m.ListVersions()
	found := false
	for _, v := range versions {
		c, _ := store.Get(v.ID)
		if string(c) == cur {
			found = true
		}
	}
	if !found {
		t.Fatal("pre-rollback config was not snapshotted; rollback is not reversible")
	}
}

// TestKeepVersionsConcurrent exercises the path that motivated making
// keepVersions atomic: a settings update (SetKeepVersions) running concurrently
// with config saves (snapshotCurrent reads keepVersions). Run with -race.
func TestKeepVersionsConcurrent(t *testing.T) {
	m, cfgPath := newTestManager(t, newFakeStore(), 10)
	if err := os.WriteFile(cfgPath, []byte(`{"log":{"level":"info"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.SetKeepVersions(1 + i%100)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.snapshotCurrent()
		}
	}()
	wg.Wait()
}

func TestVersionFeaturesNoStore(t *testing.T) {
	m, _ := newTestManager(t, nil, 10)
	m.SetVersionStore(nil)

	if vs, err := m.ListVersions(); err != nil || len(vs) != 0 {
		t.Errorf("ListVersions without store should be empty/no-error, got %v %v", vs, err)
	}
	if err := m.Rollback(); err == nil {
		t.Error("Rollback without store should error")
	}
	if _, err := m.GetVersion(1); err == nil {
		t.Error("GetVersion without store should error")
	}
}
