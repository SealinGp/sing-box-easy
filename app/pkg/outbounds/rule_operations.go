package outbounds

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
)

// nodeRulesErr maps domain errors to the response envelope codes.

// GetNodeRules returns the full ruleset (filters + groups).
func (h *Service) GetNodeRules(ctx context.Context) (any, error) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil, err
	}
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		return nil, err
	}
	return map[string]any{"filters": filters, "groups": groups}, nil
}

// ---- Filters ----

type FilterRequest struct {
	Name          string              `json:"name"`
	Matchers      []noderules.Matcher `json:"matchers"`
	Excludes      []noderules.Matcher `json:"excludes"`
	OutboundType  string              `json:"outbound_type"`
	Priority      int                 `json:"priority"`
	TestURL       string              `json:"test_url"`
	TestInterval  string              `json:"test_interval"`
	TestTolerance int                 `json:"test_tolerance"`
}

// toFilter maps a request body to a domain Filter (id assigned separately).
func (r FilterRequest) toFilter(id string) *noderules.Filter {
	return &noderules.Filter{
		ID:            id,
		Name:          r.Name,
		Matchers:      r.Matchers,
		Excludes:      r.Excludes,
		OutboundType:  r.OutboundType,
		Priority:      r.Priority,
		TestURL:       r.TestURL,
		TestInterval:  r.TestInterval,
		TestTolerance: r.TestTolerance,
	}
}

func (h *Service) GetFilters(ctx context.Context) (any, error) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil, err
	}
	return map[string]any{"filters": filters}, nil
}

func (h *Service) CreateFilter(ctx context.Context, req FilterRequest) (any, error) {

	created, err := h.nodeRulesManager.CreateFilter(req.toFilter(""))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (h *Service) UpdateFilter(ctx context.Context, id string, req FilterRequest) (any, error) {

	updated, err := h.nodeRulesManager.UpdateFilter(req.toFilter(id))
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (h *Service) DeleteFilter(ctx context.Context, id string) (any, error) {

	if err := h.nodeRulesManager.DeleteFilter(id); err != nil {
		return nil, err
	}
	return map[string]any{"message": "filter deleted", "id": id}, nil
}

// ---- Groups ----

type GroupRequest struct {
	Name      string   `json:"name"`
	FilterIDs []string `json:"filter_ids"`
	// ExtraTags: outbounds the Group names directly (e.g. `direct`), which no
	// Filter can collect on its own.
	ExtraTags []string `json:"extra_tags"`
	Priority  int      `json:"priority"`
}

func (h *Service) GetGroups(ctx context.Context) (any, error) {
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		return nil, err
	}
	return map[string]any{"groups": groups}, nil
}

func (h *Service) CreateGroup(ctx context.Context, req GroupRequest) (any, error) {

	created, err := h.nodeRulesManager.CreateGroup(&noderules.Group{
		Name:      req.Name,
		FilterIDs: req.FilterIDs,
		ExtraTags: req.ExtraTags,
		Priority:  req.Priority,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (h *Service) UpdateGroup(ctx context.Context, id string, req GroupRequest) (any, error) {

	updated, err := h.nodeRulesManager.UpdateGroup(&noderules.Group{
		ID:        id,
		Name:      req.Name,
		FilterIDs: req.FilterIDs,
		ExtraTags: req.ExtraTags,
		Priority:  req.Priority,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (h *Service) DeleteGroup(ctx context.Context, id string) (any, error) {

	if err := h.nodeRulesManager.DeleteGroup(id); err != nil {
		return nil, err
	}
	return map[string]any{"message": "group deleted", "id": id}, nil
}

// ---- Catalog / Templates ----

func (h *Service) GetNodeRuleKeywords(ctx context.Context) (any, error) {
	return map[string]any{"keywords": noderules.Catalog()}, nil
}

func (h *Service) GetNodeRuleTemplates(ctx context.Context) (any, error) {
	return map[string]any{"templates": noderules.Templates()}, nil
}

// ApplyNodeRuleTemplate creates a Filter from a built-in template.
func (h *Service) ApplyNodeRuleTemplate(ctx context.Context, id string) (any, error) {

	tpl, ok := noderules.TemplateByID(id)
	if !ok {
		return nil, fault.New(fault.Missing, "template not found: "+id)
	}
	created, err := h.nodeRulesManager.CreateFilter(&noderules.Filter{
		Name:         tpl.Name,
		Matchers:     tpl.Matchers,
		OutboundType: tpl.OutboundType,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ---- Apply / Preview ----

// previewFilter is the per-Filter summary returned by preview/apply.
type previewFilter struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	OutboundType string   `json:"outbound_type"`
	IsFallback   bool     `json:"is_fallback"`
	MemberCount  int      `json:"member_count"`
	Members      []string `json:"members"`
}

// buildPreview runs the matcher over the given node pool and returns a
// frontend-friendly per-Filter breakdown plus the unmatched tags.
func (h *Service) buildPreview(pool noderules.NodePool) ([]previewFilter, []string, error) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil, nil, err
	}
	membership, others := noderules.AssignFilters(pool, filters)
	out := make([]previewFilter, 0, len(filters))
	for _, f := range filters {
		members := membership[f.ID]
		out = append(out, previewFilter{
			ID:           f.ID,
			Name:         f.Name,
			OutboundType: f.OutboundType,
			IsFallback:   f.IsFallback,
			MemberCount:  len(members),
			Members:      members,
		})
	}
	return out, others, nil
}

// PreviewNodeRules is a dry-run: it reports how current endpoints would be
// assigned WITHOUT writing the config.
func (h *Service) PreviewNodeRules(ctx context.Context) (any, error) {
	cfg, err := h.configManager.GetOutboundsConfig()
	if err != nil {
		return nil, fault.New(fault.Configuration, err.Error())
	}
	pool := noderules.NodePool{
		Endpoints: config.EndpointTags(cfg.Outbounds),
		OptIn:     config.OptInTags(cfg.Outbounds),
	}
	preview, others, err := h.buildPreview(pool)
	if err != nil {
		return nil, err
	}
	// `optional` is the pool the UI may offer for explicit inclusion/exclusion
	// (the `direct` outbounds). They are absent from `unmatched` by design, so
	// without this the picker could not see them at all.
	return map[string]any{
		"endpoints": len(pool.Endpoints),
		"filters":   preview,
		"unmatched": others,
		"optional":  pool.OptIn,
	}, nil
}

// ApplyNodeRules rebuilds the Filter/Group outbounds in the live config from the
// current rules (no subscription fetch).
func (h *Service) ApplyNodeRules(ctx context.Context) (any, error) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil, err
	}
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		return nil, err
	}

	var (
		emittedFilters int
		emittedGroups  int
		endpoints      int
		unmatched      int
	)
	err = h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
		pool := noderules.NodePool{
			Endpoints: config.EndpointTags(cfg.Outbounds),
			OptIn:     config.OptInTags(cfg.Outbounds),
		}
		filterSpecs, groupSpecs, _, others := noderules.BuildSpecs(filters, groups, pool)
		cfg.Outbounds = config.BuildGroupOutbounds(cfg.Outbounds, filterSpecs, groupSpecs)
		endpoints = len(pool.Endpoints)
		unmatched = len(others)
		emittedFilters = len(filterSpecs)
		emittedGroups = len(groupSpecs)
		return nil
	})
	if err != nil {
		return nil, fault.New(fault.Configuration, "failed to apply node rules: "+err.Error())
	}

	return map[string]any{
		"message":   "node rules applied",
		"endpoints": endpoints,
		"filters":   emittedFilters,
		"groups":    emittedGroups,
		"unmatched": unmatched,
	}, nil
}
