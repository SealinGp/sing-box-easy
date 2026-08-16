package v1_13_0

import (
	"context"
	"fmt"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// GetOutbounds returns all outbound configurations
func (h *Handler) GetOutbounds(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	response := map[string]any{
		"outbounds": cfg.Outbounds,
		// Tags the node-rules engine owns and rebuilds in place. An edit made
		// to one of these through the outbounds form is discarded on the next
		// rule apply — for a Group the rebuilt selector carries only its
		// members, so url/interval/tolerance go too. The UI warns rather than
		// letting the operator spend time on an edit that will not survive.
		"managed_tags": h.managedOutboundTags(),
	}
	respOK(ctx, c, response)
}

// managedOutboundTags lists the outbound tags owned by the node-rules engine.
//
// Degrades to nil on any error: the warning it drives is a courtesy, and
// failing the whole outbound list because the rules tables were unreadable
// would be a bad trade.
func (h *Handler) managedOutboundTags() []string {
	if h.nodeRulesManager == nil {
		return nil
	}
	filters, err := h.nodeRulesManager.ListFilters()
	if err != nil {
		return nil
	}
	groups, err := h.nodeRulesManager.ListGroups()
	if err != nil {
		return nil
	}

	// Names alone decide ownership, so the endpoint tags the matcher would
	// assign are irrelevant here — passing none keeps this cheap.
	filterSpecs, groupSpecs, _, _ := noderules.BuildSpecs(filters, groups, nil)
	return config.ManagedOutboundTags(filterSpecs, groupSpecs)
}

// GetOutboundByTag returns a specific outbound by tag
func (h *Handler) GetOutboundByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	for _, outbound := range cfg.Outbounds {
		if outbound.Tag == tag {
			respOK(ctx, c, outbound)
			return
		}
	}

	respErr(ctx, c, CodeNotFound, "outbound not found")
}

// AddOutbound adds a new outbound
func (h *Handler) AddOutbound(ctx context.Context, c *app.RequestContext) {
	// Use sing-box JSON deserialization to properly parse outbound config
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid outbound configuration: "+err.Error())
		return
	}

	// Validate required fields
	if outbound.Tag == "" {
		respErr(ctx, c, CodeBadRequest, "tag is required")
		return
	}

	// Generate unique tag to avoid conflicts
	originalTag := outbound.Tag
	outbound.Tag = config.GenerateUniqueTag(originalTag, outbound)

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		// Check if tag already exists
		for _, existing := range cfg.Outbounds {
			if existing.Tag == outbound.Tag {
				return fmt.Errorf("outbound with tag '%s' already exists", outbound.Tag)
			}
		}

		cfg.Outbounds = append(cfg.Outbounds, outbound)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "outbound added successfully",
		"tag":     outbound.Tag,
	})
}

// AddOutboundsBatch adds multiple outbounds at once
func (h *Handler) AddOutboundsBatch(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Outbounds []config.Outbound `json:"outbounds"`
	}

	var req Request
	data := c.Request.Body()
	jsonCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(jsonCtx, data, &req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.Outbounds) == 0 {
		respErr(ctx, c, CodeBadRequest, "outbounds array is required and cannot be empty")
		return
	}

	tagMap := make(map[string]bool)
	for i, outbound := range req.Outbounds {
		if outbound.Tag == "" {
			respErr(ctx, c, CodeBadRequest, fmt.Sprintf("outbound at index %d: tag is required", i))
			return
		}

		if tagMap[outbound.Tag] {
			respErr(ctx, c, CodeBadRequest, fmt.Sprintf("duplicate tag '%s' in request", outbound.Tag))
			return
		}
		tagMap[outbound.Tag] = true
	}

	addedTags, skippedTags, err := h.configManager.UpdateOutbounds(req.Outbounds)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	response := map[string]any{
		"message":     "outbounds batch add completed",
		"added_count": len(addedTags),
		"added_tags":  addedTags,
	}

	if len(skippedTags) > 0 {
		response["skipped_count"] = len(skippedTags)
		response["skipped_tags"] = skippedTags
		response["message"] = fmt.Sprintf("added %d outbounds, skipped %d existing outbounds", len(addedTags), len(skippedTags))
	}

	respOK(ctx, c, response)
}

// UpdateOutbound updates an existing outbound
func (h *Handler) UpdateOutbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	// Use sing-box JSON deserialization to properly parse outbound config
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid outbound configuration: "+err.Error())
		return
	}

	// Ensure the tag matches
	outbound.Tag = tag

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		found := false
		for i, existing := range cfg.Outbounds {
			if existing.Tag == tag {
				cfg.Outbounds[i] = outbound
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("outbound not found")
		}

		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "outbound updated successfully",
		"tag":     tag,
	})
}

// DeleteOutbound deletes an outbound
func (h *Handler) DeleteOutbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag") //tag or index
	idx, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		idx = -1
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		newOutbounds := make([]config.Outbound, 0, len(cfg.Outbounds))
		deletedTags := make(map[string]struct{})
		found := false

		for i, outbound := range cfg.Outbounds {
			if idx > -1 && int(idx) == i {
				found = true
				deletedTags[outbound.Tag] = struct{}{}
				continue
			}
			if outbound.Tag == tag {
				found = true
				deletedTags[outbound.Tag] = struct{}{}
				continue
			}

			newOutbounds = append(newOutbounds, outbound)
		}

		if !found {
			return fmt.Errorf("outbound not found")
		}

		// Strip the deleted tag from selector/urltest group references so the
		// config doesn't silently keep dangling pointers.
		newOutbounds = config.PruneGroupReferences(newOutbounds, deletedTags, nil, nil)

		cfg.Outbounds = newOutbounds
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "outbound deleted successfully",
		"tag":     tag,
	})
}

// DeleteOutboundsBatch deletes multiple outbounds at once
func (h *Handler) DeleteOutboundsBatch(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Tags []string `json:"tags"`
	}

	var req Request
	data := c.Request.Body()
	jsonCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(jsonCtx, data, &req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(req.Tags) == 0 {
		respErr(ctx, c, CodeBadRequest, "tags array is required and cannot be empty")
		return
	}

	// Track results across the UpdateConfig closure so the response can
	// reflect what was actually deleted vs. what didn't exist.
	var (
		deletedTags  []string
		notFoundTags []string
	)

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		// Build a set of existing tags once so the not-found check is O(n+m)
		// rather than the previous O(n*m).
		existing := make(map[string]bool, len(cfg.Outbounds))
		for _, outbound := range cfg.Outbounds {
			existing[outbound.Tag] = true
		}

		tagSet := make(map[string]bool, len(req.Tags))
		for _, tag := range req.Tags {
			tagSet[tag] = true
			if !existing[tag] {
				notFoundTags = append(notFoundTags, tag)
			}
		}

		newOutbounds := make([]config.Outbound, 0, len(cfg.Outbounds))
		deletedSet := make(map[string]struct{}, len(req.Tags))
		for _, outbound := range cfg.Outbounds {
			if tagSet[outbound.Tag] {
				deletedTags = append(deletedTags, outbound.Tag)
				deletedSet[outbound.Tag] = struct{}{}
				continue
			}
			newOutbounds = append(newOutbounds, outbound)
		}

		// Strip every deleted tag from selector/urltest group references so the
		// resulting config doesn't silently keep dangling pointers.
		newOutbounds = config.PruneGroupReferences(newOutbounds, deletedSet, nil, nil)

		cfg.Outbounds = newOutbounds
		logger.Info(fmt.Sprintf("deleted %d outbounds: %v", len(deletedTags), deletedTags))
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message":        "outbounds deleted successfully",
		"deleted_count":  len(deletedTags),
		"deleted_tags":   deletedTags,
		"not_found_tags": notFoundTags,
	})
}

// GetOutboundGroups returns all group type outbounds (selector/urltest)
func (h *Handler) GetOutboundGroups(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	groups := make([]option.Outbound, 0)
	for _, outbound := range cfg.Outbounds {
		if outbound.Type == "selector" || outbound.Type == "urltest" {
			groups = append(groups, outbound)
		}
	}

	respOK(ctx, c, map[string]any{"groups": groups})
}

// UpdateOutboundMembers updates the members of a group outbound
func (h *Handler) UpdateOutboundMembers(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	type Request struct {
		Outbounds []string `json:"outbounds"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		found := false
		for i, outbound := range cfg.Outbounds {
			if outbound.Tag == tag {
				if outbound.Type != "selector" && outbound.Type != "urltest" {
					return fmt.Errorf("outbound '%s' is not a group type (selector/urltest)", tag)
				}

				// Update the Outbounds field in the Options
				switch outbound.Type {
				case "selector":
					if opts, ok := outbound.Options.(*option.SelectorOutboundOptions); ok {
						opts.Outbounds = req.Outbounds
						cfg.Outbounds[i].Options = opts
					} else {
						return fmt.Errorf("invalid selector outbound options")
					}
				case "urltest":
					if opts, ok := outbound.Options.(*option.URLTestOutboundOptions); ok {
						opts.Outbounds = req.Outbounds
						cfg.Outbounds[i].Options = opts
					} else {
						return fmt.Errorf("invalid urltest outbound options")
					}
				}

				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("outbound not found")
		}

		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "outbound members updated successfully",
		"tag":     tag,
	})
}
