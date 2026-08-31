package noderules

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules/repo"
	"go.uber.org/zap"
	"xorm.io/xorm"
)

// ErrNotFound is returned (wrapped) when a filter/group ID has no row.
var ErrNotFound = errors.New("node rule not found")

// ErrFallbackProtected is returned when an operation would delete or
// structurally break the mandatory fallback Filter.
var ErrFallbackProtected = errors.New("the fallback filter cannot be deleted")

// ErrDuplicateName is returned when a filter/group name collides.
var ErrDuplicateName = errors.New("name already exists")

// ErrInvalidInput is returned (wrapped) when a request field fails validation
// (empty name, out-of-range tolerance, unparseable interval, ...). Distinct from
// ErrDuplicateName so the API can map it to a 4xx "bad request" rather than a
// "conflict".
var ErrInvalidInput = errors.New("invalid input")

// ManagerXORM persists Filters and Groups using XORM.
type ManagerXORM struct {
	e *xorm.Engine
}

// NewManagerXORM creates a new XORM-backed node-rules manager.
func NewManagerXORM() *ManagerXORM {
	e, err := database.GetEngine()
	if err != nil {
		logger.Fatal("Failed to get database engine", zap.Error(err))
	}
	return &ManagerXORM{e: e}
}

// Init syncs the schema and seeds the mandatory fallback Filter.
func (m *ManagerXORM) Init() error {
	if err := m.e.Sync2(new(repo.FilterRule), new(repo.GroupRule)); err != nil {
		return fmt.Errorf("failed to sync node-rules tables: %w", err)
	}
	if err := m.seedFallback(); err != nil {
		return err
	}
	logger.Info("Node-rules manager initialized with XORM")
	return nil
}

// seedFallback ensures the protected "Other Nodes" Filter exists exactly once.
func (m *ManagerXORM) seedFallback() error {
	has, err := m.e.ID(FallbackFilterID).Exist(new(repo.FilterRule))
	if err != nil {
		return fmt.Errorf("failed to check fallback filter: %w", err)
	}
	if has {
		return nil
	}
	row := &repo.FilterRule{
		ID:           FallbackFilterID,
		Name:         FallbackFilterName,
		Matchers:     "[]",
		OutboundType: OutboundTypeURLTest,
		Priority:     FallbackPriority,
		IsFallback:   true,
	}
	if _, err := m.e.Insert(row); err != nil {
		return fmt.Errorf("failed to seed fallback filter: %w", err)
	}
	logger.Info("Seeded mandatory fallback filter", zap.String("id", FallbackFilterID))
	return nil
}

// ---- Filters ----

// ListFilters returns all Filters ordered by priority asc, then name.
func (m *ManagerXORM) ListFilters() ([]*Filter, error) {
	var rows []repo.FilterRule
	if err := m.e.Asc("priority").Asc("name").Find(&rows); err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}
	out := make([]*Filter, len(rows))
	for i := range rows {
		f, err := filterFromRow(&rows[i])
		if err != nil {
			return nil, err
		}
		out[i] = f
	}
	return out, nil
}

// GetFilter returns a single Filter by ID.
func (m *ManagerXORM) GetFilter(id string) (*Filter, error) {
	var row repo.FilterRule
	has, err := m.e.ID(id).Get(&row)
	if err != nil {
		return nil, fmt.Errorf("failed to get filter %q: %w", id, err)
	}
	if !has {
		return nil, fmt.Errorf("filter %q: %w", id, ErrNotFound)
	}
	return filterFromRow(&row)
}

// CreateFilter inserts a new (non-fallback) Filter. The ID and IsFallback are
// assigned by the manager; callers cannot mint a second fallback.
func (m *ManagerXORM) CreateFilter(f *Filter) (*Filter, error) {
	if err := validateName(f.Name); err != nil {
		return nil, err
	}
	if err := validateFilterSettings(f); err != nil {
		return nil, err
	}
	dup, err := m.nameTaken(new(repo.FilterRule), f.Name, "")
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, fmt.Errorf("filter name %q: %w", f.Name, ErrDuplicateName)
	}
	matchers, err := marshalMatchers(f.Matchers)
	if err != nil {
		return nil, err
	}
	excludes, err := marshalMatchers(f.Excludes)
	if err != nil {
		return nil, err
	}
	row := &repo.FilterRule{
		ID:            fmt.Sprintf("filter_%d", time.Now().UnixNano()),
		Name:          f.Name,
		Matchers:      matchers,
		Excludes:      excludes,
		OutboundType:  NormalizeOutboundType(f.OutboundType),
		Priority:      f.Priority,
		IsFallback:    false,
		TestURL:       f.TestURL,
		TestInterval:  f.TestInterval,
		TestTolerance: f.TestTolerance,
	}
	if _, err := m.e.Insert(row); err != nil {
		return nil, fmt.Errorf("failed to insert filter: %w", err)
	}
	return filterFromRow(row)
}

// UpdateFilter updates an existing Filter. The fallback Filter may be renamed
// and retyped but keeps its IsFallback flag and ignores matcher edits (it is the
// catch-all, not a matched bucket).
func (m *ManagerXORM) UpdateFilter(f *Filter) (*Filter, error) {
	existing, err := m.GetFilter(f.ID)
	if err != nil {
		return nil, err
	}
	if err := validateName(f.Name); err != nil {
		return nil, err
	}
	if err := validateFilterSettings(f); err != nil {
		return nil, err
	}
	dup, err := m.nameTaken(new(repo.FilterRule), f.Name, f.ID)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, fmt.Errorf("filter name %q: %w", f.Name, ErrDuplicateName)
	}

	matchers := "[]"
	excludes := "[]"
	if !existing.IsFallback {
		matchers, err = marshalMatchers(f.Matchers)
		if err != nil {
			return nil, err
		}
		excludes, err = marshalMatchers(f.Excludes)
		if err != nil {
			return nil, err
		}
	}
	priority := f.Priority
	if existing.IsFallback {
		priority = FallbackPriority // keep fallback last regardless of input
	}

	row := &repo.FilterRule{
		Name:          f.Name,
		Matchers:      matchers,
		Excludes:      excludes,
		OutboundType:  NormalizeOutboundType(f.OutboundType),
		Priority:      priority,
		TestURL:       f.TestURL,
		TestInterval:  f.TestInterval,
		TestTolerance: f.TestTolerance,
	}
	// Cols forces writing these columns even when zero-valued.
	if _, err := m.e.ID(f.ID).Cols("name", "matchers", "excludes", "outbound_type", "priority", "test_url", "test_interval", "test_tolerance").Update(row); err != nil {
		return nil, fmt.Errorf("failed to update filter %q: %w", f.ID, err)
	}
	return m.GetFilter(f.ID)
}

// DeleteFilter removes a Filter (never the fallback) and scrubs it from every
// Group's membership.
func (m *ManagerXORM) DeleteFilter(id string) error {
	existing, err := m.GetFilter(id)
	if err != nil {
		return err
	}
	if existing.IsFallback {
		return ErrFallbackProtected
	}
	session := m.e.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	if _, err := session.ID(id).Delete(new(repo.FilterRule)); err != nil {
		_ = session.Rollback()
		return fmt.Errorf("failed to delete filter %q: %w", id, err)
	}
	// Remove the deleted ID from every group's filter list.
	var groups []repo.GroupRule
	if err := session.Find(&groups); err != nil {
		_ = session.Rollback()
		return fmt.Errorf("failed to load groups for scrub: %w", err)
	}
	for i := range groups {
		ids, err := unmarshalIDs(groups[i].FilterIDs)
		if err != nil {
			_ = session.Rollback()
			return fmt.Errorf("failed to decode filter_ids for group %q: %w", groups[i].ID, err)
		}
		pruned := removeString(ids, id)
		if len(pruned) == len(ids) {
			continue
		}
		encoded, _ := marshalIDs(pruned)
		if _, err := session.ID(groups[i].ID).Cols("filter_ids").Update(&repo.GroupRule{FilterIDs: encoded}); err != nil {
			_ = session.Rollback()
			return fmt.Errorf("failed to scrub group %q: %w", groups[i].ID, err)
		}
	}
	return session.Commit()
}

// ---- Groups ----

// ListGroups returns all Groups ordered by priority asc, then name.
func (m *ManagerXORM) ListGroups() ([]*Group, error) {
	var rows []repo.GroupRule
	if err := m.e.Asc("priority").Asc("name").Find(&rows); err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	out := make([]*Group, len(rows))
	for i := range rows {
		g, err := groupFromRow(&rows[i])
		if err != nil {
			return nil, err
		}
		out[i] = g
	}
	return out, nil
}

// GetGroup returns a single Group by ID.
func (m *ManagerXORM) GetGroup(id string) (*Group, error) {
	var row repo.GroupRule
	has, err := m.e.ID(id).Get(&row)
	if err != nil {
		return nil, fmt.Errorf("failed to get group %q: %w", id, err)
	}
	if !has {
		return nil, fmt.Errorf("group %q: %w", id, ErrNotFound)
	}
	return groupFromRow(&row)
}

// CreateGroup inserts a new Group.
func (m *ManagerXORM) CreateGroup(g *Group) (*Group, error) {
	if err := validateName(g.Name); err != nil {
		return nil, err
	}
	dup, err := m.nameTaken(new(repo.GroupRule), g.Name, "")
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, fmt.Errorf("group name %q: %w", g.Name, ErrDuplicateName)
	}
	ids, err := marshalIDs(g.FilterIDs)
	if err != nil {
		return nil, err
	}
	extras, err := marshalIDs(g.ExtraTags)
	if err != nil {
		return nil, err
	}
	row := &repo.GroupRule{
		ID:        fmt.Sprintf("group_%d", time.Now().UnixNano()),
		Name:      g.Name,
		FilterIDs: ids,
		ExtraTags: extras,
		Priority:  g.Priority,
	}
	if _, err := m.e.Insert(row); err != nil {
		return nil, fmt.Errorf("failed to insert group: %w", err)
	}
	return groupFromRow(row)
}

// UpdateGroup updates an existing Group.
func (m *ManagerXORM) UpdateGroup(g *Group) (*Group, error) {
	if _, err := m.GetGroup(g.ID); err != nil {
		return nil, err
	}
	if err := validateName(g.Name); err != nil {
		return nil, err
	}
	dup, err := m.nameTaken(new(repo.GroupRule), g.Name, g.ID)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, fmt.Errorf("group name %q: %w", g.Name, ErrDuplicateName)
	}
	ids, err := marshalIDs(g.FilterIDs)
	if err != nil {
		return nil, err
	}
	extras, err := marshalIDs(g.ExtraTags)
	if err != nil {
		return nil, err
	}
	row := &repo.GroupRule{Name: g.Name, FilterIDs: ids, ExtraTags: extras, Priority: g.Priority}
	if _, err := m.e.ID(g.ID).Cols("name", "filter_ids", "extra_tags", "priority").Update(row); err != nil {
		return nil, fmt.Errorf("failed to update group %q: %w", g.ID, err)
	}
	return m.GetGroup(g.ID)
}

// DeleteGroup removes a Group.
func (m *ManagerXORM) DeleteGroup(id string) error {
	if _, err := m.GetGroup(id); err != nil {
		return err
	}
	if _, err := m.e.ID(id).Delete(new(repo.GroupRule)); err != nil {
		return fmt.Errorf("failed to delete group %q: %w", id, err)
	}
	return nil
}

// ---- helpers ----

func (m *ManagerXORM) nameTaken(bean interface{}, name, excludeID string) (bool, error) {
	session := m.e.Where("name = ?", name)
	if excludeID != "" {
		session = session.And("id != ?", excludeID)
	}
	has, err := session.Exist(bean)
	if err != nil {
		return false, fmt.Errorf("failed to check name uniqueness: %w", err)
	}
	return has, nil
}

// validateName enforces a non-empty name for both filters and groups.
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required: %w", ErrInvalidInput)
	}
	return nil
}

// validateFilterSettings checks the urltest health-check fields of a Filter.
// Both are only meaningful for urltest filters but are validated unconditionally
// so a bad value is never silently persisted.
func validateFilterSettings(f *Filter) error {
	if f.TestTolerance < 0 || f.TestTolerance > math.MaxUint16 {
		return fmt.Errorf("test_tolerance %d out of range [0,%d]: %w", f.TestTolerance, math.MaxUint16, ErrInvalidInput)
	}
	if f.TestInterval != "" {
		if _, err := time.ParseDuration(f.TestInterval); err != nil {
			return fmt.Errorf("invalid test_interval %q: %w", f.TestInterval, ErrInvalidInput)
		}
	}
	return nil
}

func filterFromRow(r *repo.FilterRule) (*Filter, error) {
	matchers, err := unmarshalMatchers(r.Matchers)
	if err != nil {
		return nil, fmt.Errorf("filter %q has invalid matchers: %w", r.ID, err)
	}
	excludes, err := unmarshalMatchers(r.Excludes)
	if err != nil {
		return nil, fmt.Errorf("filter %q has invalid excludes: %w", r.ID, err)
	}
	return &Filter{
		ID:            r.ID,
		Name:          r.Name,
		Matchers:      matchers,
		Excludes:      excludes,
		OutboundType:  NormalizeOutboundType(r.OutboundType),
		Priority:      r.Priority,
		IsFallback:    r.IsFallback,
		TestURL:       r.TestURL,
		TestInterval:  r.TestInterval,
		TestTolerance: r.TestTolerance,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}, nil
}

func groupFromRow(r *repo.GroupRule) (*Group, error) {
	ids, err := unmarshalIDs(r.FilterIDs)
	if err != nil {
		return nil, fmt.Errorf("group %q has invalid filter_ids: %w", r.ID, err)
	}
	extras, err := unmarshalIDs(r.ExtraTags)
	if err != nil {
		return nil, fmt.Errorf("group %q has invalid extra_tags: %w", r.ID, err)
	}
	return &Group{
		ID:        r.ID,
		Name:      r.Name,
		FilterIDs: ids,
		ExtraTags: extras,
		Priority:  r.Priority,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func marshalMatchers(ms []Matcher) (string, error) {
	if ms == nil {
		ms = []Matcher{}
	}
	b, err := json.Marshal(ms)
	if err != nil {
		return "", fmt.Errorf("failed to encode matchers: %w", err)
	}
	return string(b), nil
}

func unmarshalMatchers(s string) ([]Matcher, error) {
	if s == "" {
		return nil, nil
	}
	var ms []Matcher
	if err := json.Unmarshal([]byte(s), &ms); err != nil {
		return nil, err
	}
	return ms, nil
}

func marshalIDs(ids []string) (string, error) {
	if ids == nil {
		ids = []string{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", fmt.Errorf("failed to encode tag list: %w", err)
	}
	return string(b), nil
}

func unmarshalIDs(s string) ([]string, error) {
	if s == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(s), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func removeString(list []string, target string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s != target {
			out = append(out, s)
		}
	}
	return out
}
