package noderules

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

// TestMain initializes the logger and a single process-wide SQLite database
// (database.Init is guarded by sync.Once, so it can only run once per process).
func TestMain(m *testing.M) {
	logger.InitDefault()
	dir, err := os.MkdirTemp("", "noderules_test")
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

// newTestManager returns an initialized manager over a freshly-truncated schema
// so each test starts from a clean slate (only the seeded fallback present).
func newTestManager(t *testing.T) *ManagerXORM {
	t.Helper()
	m := NewManagerXORM()
	// First Init ensures the tables exist.
	if err := m.Init(); err != nil {
		t.Fatalf("manager init: %v", err)
	}
	// Truncate so the test starts clean, then re-seed the fallback.
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("get engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM filter_rules"); err != nil {
		t.Fatalf("truncate filters: %v", err)
	}
	if _, err := e.Exec("DELETE FROM group_rules"); err != nil {
		t.Fatalf("truncate groups: %v", err)
	}
	if err := m.Init(); err != nil {
		t.Fatalf("manager re-init (reseed): %v", err)
	}
	return m
}

// TestManager_SeedsFallbackOnce verifies Init seeds exactly one protected
// fallback Filter and is idempotent.
func TestManager_SeedsFallbackOnce(t *testing.T) {
	m := newTestManager(t)

	if err := m.Init(); err != nil { // second Init must not duplicate
		t.Fatalf("second init: %v", err)
	}
	filters, err := m.ListFilters()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	count := 0
	for _, f := range filters {
		if f.IsFallback {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("fallback count = %d, want exactly 1", count)
	}
}

// TestManager_FallbackCannotBeDeleted verifies the protection.
func TestManager_FallbackCannotBeDeleted(t *testing.T) {
	m := newTestManager(t)
	err := m.DeleteFilter(FallbackFilterID)
	if !errors.Is(err, ErrFallbackProtected) {
		t.Fatalf("delete fallback err = %v, want ErrFallbackProtected", err)
	}
}

// TestManager_FilterCRUDAndDuplicateName covers create/update/list + the unique
// name constraint surfaced as ErrDuplicateName.
func TestManager_FilterCRUDAndDuplicateName(t *testing.T) {
	m := newTestManager(t)

	asia, err := m.CreateFilter(&Filter{Name: "Asia", OutboundType: "urltest", Priority: 10, Matchers: []Matcher{{Type: MatcherCode, Value: "HK"}}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if asia.ID == "" || asia.IsFallback {
		t.Fatalf("unexpected created filter: %+v", asia)
	}

	// Duplicate name rejected.
	if _, err := m.CreateFilter(&Filter{Name: "Asia"}); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("dup create err = %v, want ErrDuplicateName", err)
	}

	// Update matchers + name.
	asia.Name = "Asia-Pacific"
	asia.Matchers = append(asia.Matchers, Matcher{Type: MatcherCode, Value: "JP"})
	updated, err := m.UpdateFilter(asia)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Asia-Pacific" || len(updated.Matchers) != 2 {
		t.Fatalf("update not persisted: %+v", updated)
	}

	// Delete works for a normal filter.
	if err := m.DeleteFilter(asia.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := m.GetFilter(asia.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete err = %v, want ErrNotFound", err)
	}
}

// TestManager_DeleteFilterScrubsGroups verifies deleting a Filter removes it
// from any Group's membership.
func TestManager_DeleteFilterScrubsGroups(t *testing.T) {
	m := newTestManager(t)

	asia, _ := m.CreateFilter(&Filter{Name: "Asia", OutboundType: "urltest"})
	us, _ := m.CreateFilter(&Filter{Name: "US", OutboundType: "urltest"})
	grp, err := m.CreateGroup(&Group{Name: "All", FilterIDs: []string{asia.ID, us.ID}})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if err := m.DeleteFilter(asia.ID); err != nil {
		t.Fatalf("delete filter: %v", err)
	}
	got, err := m.GetGroup(grp.ID)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if len(got.FilterIDs) != 1 || got.FilterIDs[0] != us.ID {
		t.Fatalf("group membership after scrub = %v, want [%s]", got.FilterIDs, us.ID)
	}
}
