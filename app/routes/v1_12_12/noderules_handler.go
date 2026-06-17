package v1_13_0

import (
	"context"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules"
	"github.com/cloudwego/hertz/pkg/app"
)

// nodeRulesErr maps domain errors to the response envelope codes.
func nodeRulesErr(ctx context.Context, c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, noderules.ErrNotFound):
		respErr(ctx, c, CodeNotFound, err.Error())
	case errors.Is(err, noderules.ErrInvalidInput):
		respErr(ctx, c, CodeValidationError, err.Error())
	case errors.Is(err, noderules.ErrDuplicateName):
		respErr(ctx, c, CodeConflict, err.Error())
	case errors.Is(err, noderules.ErrFallbackProtected):
		respErr(ctx, c, CodeForbidden, err.Error())
	default:
		respErr(ctx, c, CodeInternalError, err.Error())
	}
}

// GetNodeRules returns the full ruleset (filters + groups).
func (h *Handler) GetNodeRules(ctx context.Context, c *app.RequestContext) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"filters": filters, "groups": groups})
}

// ---- Filters ----

type filterRequest struct {
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
func (r filterRequest) toFilter(id string) *noderules.Filter {
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

func (h *Handler) GetFilters(ctx context.Context, c *app.RequestContext) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"filters": filters})
}

func (h *Handler) CreateFilter(ctx context.Context, c *app.RequestContext) {
	var req filterRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	created, err := h.nodeRulesManager.CreateFilter(req.toFilter(""))
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, created)
}

func (h *Handler) UpdateFilter(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	var req filterRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	updated, err := h.nodeRulesManager.UpdateFilter(req.toFilter(id))
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, updated)
}

func (h *Handler) DeleteFilter(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if err := h.nodeRulesManager.DeleteFilter(id); err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "filter deleted", "id": id})
}

// ---- Groups ----

type groupRequest struct {
	Name      string   `json:"name"`
	FilterIDs []string `json:"filter_ids"`
	Priority  int      `json:"priority"`
}

func (h *Handler) GetGroups(ctx context.Context, c *app.RequestContext) {
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"groups": groups})
}

func (h *Handler) CreateGroup(ctx context.Context, c *app.RequestContext) {
	var req groupRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	created, err := h.nodeRulesManager.CreateGroup(&noderules.Group{
		Name:      req.Name,
		FilterIDs: req.FilterIDs,
		Priority:  req.Priority,
	})
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, created)
}

func (h *Handler) UpdateGroup(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	var req groupRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	updated, err := h.nodeRulesManager.UpdateGroup(&noderules.Group{
		ID:        id,
		Name:      req.Name,
		FilterIDs: req.FilterIDs,
		Priority:  req.Priority,
	})
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, updated)
}

func (h *Handler) DeleteGroup(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if err := h.nodeRulesManager.DeleteGroup(id); err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "group deleted", "id": id})
}

// ---- Catalog / Templates ----

func (h *Handler) GetNodeRuleKeywords(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, map[string]any{"keywords": noderules.Catalog()})
}

func (h *Handler) GetNodeRuleTemplates(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, map[string]any{"templates": noderules.Templates()})
}

// ApplyNodeRuleTemplate creates a Filter from a built-in template.
func (h *Handler) ApplyNodeRuleTemplate(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	tpl, ok := noderules.TemplateByID(id)
	if !ok {
		respErr(ctx, c, CodeNotFound, "template not found: "+id)
		return
	}
	created, err := h.nodeRulesManager.CreateFilter(&noderules.Filter{
		Name:         tpl.Name,
		Matchers:     tpl.Matchers,
		OutboundType: tpl.OutboundType,
	})
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, created)
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

// buildPreview runs the matcher over the given endpoint tags and returns a
// frontend-friendly per-Filter breakdown plus the unmatched tags.
func (h *Handler) buildPreview(endpointTags []string) ([]previewFilter, []string, error) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil, nil, err
	}
	membership, others := noderules.AssignFilters(endpointTags, filters)
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
func (h *Handler) PreviewNodeRules(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}
	endpointTags := config.EndpointTags(cfg.Outbounds)
	preview, others, err := h.buildPreview(endpointTags)
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{
		"endpoints": len(endpointTags),
		"filters":   preview,
		"unmatched": others,
	})
}

// ApplyNodeRules rebuilds the Filter/Group outbounds in the live config from the
// current rules (no subscription fetch).
func (h *Handler) ApplyNodeRules(ctx context.Context, c *app.RequestContext) {
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		nodeRulesErr(ctx, c, err)
		return
	}

	var (
		emittedFilters int
		emittedGroups  int
		endpoints      int
		unmatched      int
	)
	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		endpointTags := config.EndpointTags(cfg.Outbounds)
		filterSpecs, groupSpecs, _, others := noderules.BuildSpecs(filters, groups, endpointTags)
		cfg.Outbounds = config.BuildGroupOutbounds(cfg.Outbounds, filterSpecs, groupSpecs)
		endpoints = len(endpointTags)
		unmatched = len(others)
		emittedFilters = len(filterSpecs)
		emittedGroups = len(groupSpecs)
		return nil
	})
	if err != nil {
		respErr(ctx, c, CodeConfigError, "failed to apply node rules: "+err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message":   "node rules applied",
		"endpoints": endpoints,
		"filters":   emittedFilters,
		"groups":    emittedGroups,
		"unmatched": unmatched,
	})
}
