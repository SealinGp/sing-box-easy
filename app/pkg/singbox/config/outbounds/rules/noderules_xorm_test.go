package rules

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
	m := NewManagerXORM(testEngine())
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

// TestManager_GroupExtraTagsRoundTrip pins that a Group's directly-named
// outbounds (the `direct` bypass case) survive create, read and update. They
// live in their own column, so a missed Cols() on update would silently drop
// them on the next edit.
func TestManager_GroupExtraTagsRoundTrip(t *testing.T) {
	m := newTestManager(t)

	asia, _ := m.CreateFilter(&Filter{Name: "Asia", OutboundType: "urltest"})
	created, err := m.CreateGroup(&Group{
		Name:      "All",
		FilterIDs: []string{asia.ID},
		ExtraTags: []string{"direct"},
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if len(created.ExtraTags) != 1 || created.ExtraTags[0] != "direct" {
		t.Fatalf("created extra_tags = %v, want [direct]", created.ExtraTags)
	}

	updated, err := m.UpdateGroup(&Group{
		ID:        created.ID,
		Name:      "All",
		FilterIDs: []string{asia.ID},
		ExtraTags: []string{"direct", "bypass-cn"},
	})
	if err != nil {
		t.Fatalf("update group: %v", err)
	}
	if len(updated.ExtraTags) != 2 || updated.ExtraTags[1] != "bypass-cn" {
		t.Fatalf("updated extra_tags = %v, want [direct bypass-cn]", updated.ExtraTags)
	}

	// Clearing must persist as empty, not fall back to the stored value.
	cleared, err := m.UpdateGroup(&Group{ID: created.ID, Name: "All", FilterIDs: []string{asia.ID}})
	if err != nil {
		t.Fatalf("clear extra tags: %v", err)
	}
	if len(cleared.ExtraTags) != 0 {
		t.Fatalf("cleared extra_tags = %v, want empty", cleared.ExtraTags)
	}
}

// The seeded fallback is never saved through a form, so it stores no health
// check values. It must still REPORT the ones its urltest group runs with —
// the same ones BuildSpecs writes into config.json — not three blanks.
func TestManager_FallbackReportsEffectiveURLTestSettings(t *testing.T) {
	m := newTestManager(t)
	fallback, err := m.GetFilter(FallbackFilterID)
	if err != nil {
		t.Fatalf("get fallback: %v", err)
	}
	if fallback.TestURL != DefaultURLTestURL || fallback.TestInterval != DefaultURLTestInterval || fallback.TestTolerance != DefaultURLTestTolerance {
		t.Fatalf("fallback url-test settings = (%q, %q, %d), want the defaults (%q, %q, %d)",
			fallback.TestURL, fallback.TestInterval, fallback.TestTolerance,
			DefaultURLTestURL, DefaultURLTestInterval, DefaultURLTestTolerance)
	}

	// What is reported is what is built: the two must not be able to disagree.
	url, interval, tolerance := fallback.URLTestSettings()
	if url != fallback.TestURL || interval != fallback.TestInterval || tolerance != fallback.TestTolerance {
		t.Errorf("reported settings differ from the ones the config is built with")
	}

	listed, err := m.ListFilters()
	if err != nil {
		t.Fatalf("list filters: %v", err)
	}
	for _, filter := range listed {
		if filter.IsFallback && filter.TestURL == "" {
			t.Error("the list endpoint still serves the fallback with a blank test_url")
		}
	}
}

// A filter's own choices win over the defaults, and a selector — which has no
// health check — is not handed settings it would never use.
func TestManager_URLTestDefaultsDoNotOverrideOrLeakIntoSelectors(t *testing.T) {
	m := newTestManager(t)

	custom, err := m.CreateFilter(&Filter{
		Name: "custom", OutboundType: OutboundTypeURLTest, Priority: 10,
		Matchers: []Matcher{{Type: "keyword", Value: "hk"}},
		TestURL:  "https://example.com/204", TestInterval: "30s", TestTolerance: 50,
	})
	if err != nil {
		t.Fatalf("create urltest filter: %v", err)
	}
	if custom.TestURL != "https://example.com/204" || custom.TestInterval != "30s" || custom.TestTolerance != 50 {
		t.Errorf("explicit settings were overridden: (%q, %q, %d)", custom.TestURL, custom.TestInterval, custom.TestTolerance)
	}

	selector, err := m.CreateFilter(&Filter{
		Name: "manual", OutboundType: OutboundTypeSelector, Priority: 20,
		Matchers: []Matcher{{Type: "keyword", Value: "jp"}},
	})
	if err != nil {
		t.Fatalf("create selector filter: %v", err)
	}
	if selector.TestURL != "" || selector.TestInterval != "" || selector.TestTolerance != 0 {
		t.Errorf("a selector was given url-test settings: (%q, %q, %d)", selector.TestURL, selector.TestInterval, selector.TestTolerance)
	}
}
