package outbounds

import (
	"context"
	wirejson "encoding/json"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func (h *Service) GetOutbounds(ctx context.Context) (any, error) {
	cfg, err := h.configManager.GetOutboundsConfig()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
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
	return response, nil
}

func (h *Service) managedOutboundTags() []string {
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
	filterSpecs, groupSpecs, _, _ := noderules.BuildSpecs(filters, groups, noderules.NodePool{})
	return config.ManagedOutboundTags(filterSpecs, groupSpecs)
}

func (h *Service) GetOutboundByTag(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag

	cfg, err := h.configManager.GetOutboundsConfig()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	for _, outbound := range cfg.Outbounds {
		if outbound.Tag == tag {
			return outbound, nil
		}
	}

	return nil, fault.New(fault.Missing, "outbound not found")
}

func (h *Service) AddOutbound(ctx context.Context, body []byte) (any, error) {
	// Use sing-box JSON deserialization to properly parse outbound config
	var err error

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		return nil, fault.New(fault.Input, "invalid outbound configuration: "+err.Error())
	}

	// Validate required fields
	if outbound.Tag == "" {
		return nil, fault.New(fault.Input, "tag is required")
	}

	// Mint the tag the way every node-minting path does: the name plus a short
	// fingerprint of the endpoint. The remaining candidates are the shapes an
	// older build produced, so "already exists" still recognizes a node added
	// before the format changed.
	candidates := config.OutboundTagCandidates(outbound.Tag, outbound)
	outbound.Tag = candidates[0]

	err = h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
		// Check if the tag already exists, under any shape this panel has minted.
		taken := make(map[string]bool, len(cfg.Outbounds))
		for _, existing := range cfg.Outbounds {
			taken[existing.Tag] = true
		}
		for _, candidate := range candidates {
			if taken[candidate] {
				return fmt.Errorf("outbound with tag '%s' already exists", candidate)
			}
		}

		cfg.Outbounds = append(cfg.Outbounds, outbound)
		return nil
	})

	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "outbound added successfully",
		"tag":     outbound.Tag,
	}, nil
}

func (h *Service) AddOutboundsBatch(ctx context.Context, body []byte) (any, error) {
	type Request struct {
		Outbounds []config.Outbound `json:"outbounds"`
	}

	var req Request
	data := body
	jsonCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(jsonCtx, data, &req); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}

	if len(req.Outbounds) == 0 {
		return nil, fault.New(fault.Input, "outbounds array is required and cannot be empty")
	}

	tagMap := make(map[string]bool)
	for i, outbound := range req.Outbounds {
		if outbound.Tag == "" {
			return nil, fault.New(fault.Input, fmt.Sprintf("outbound at index %d: tag is required", i))
		}

		if tagMap[outbound.Tag] {
			return nil, fault.New(fault.Input, fmt.Sprintf("duplicate tag '%s' in request", outbound.Tag))
		}
		tagMap[outbound.Tag] = true
	}

	addedTags, skippedTags, err := h.configManager.UpdateOutbounds(req.Outbounds)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
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

	return response, nil
}

func (h *Service) UpdateOutbound(ctx context.Context, body []byte, pathTag string) (any, error) {
	tag := pathTag

	// Use sing-box JSON deserialization to properly parse outbound config
	var err error

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		return nil, fault.New(fault.Input, "invalid outbound configuration: "+err.Error())
	}

	// Ensure the tag matches
	outbound.Tag = tag

	err = h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "outbound updated successfully",
		"tag":     tag,
	}, nil
}

func (h *Service) DeleteOutbound(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag //tag or index
	idx, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		idx = -1
	}

	err = h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "outbound deleted successfully",
		"tag":     tag,
	}, nil
}

func (h *Service) DeleteOutboundsBatch(ctx context.Context, body []byte) (any, error) {
	type Request struct {
		Tags []string `json:"tags"`
	}

	var req Request
	data := body
	jsonCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(jsonCtx, data, &req); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}

	if len(req.Tags) == 0 {
		return nil, fault.New(fault.Input, "tags array is required and cannot be empty")
	}

	// Track results across the UpdateConfig closure so the response can
	// reflect what was actually deleted vs. what didn't exist.
	var (
		deletedTags  []string
		notFoundTags []string
	)

	err := h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message":        "outbounds deleted successfully",
		"deleted_count":  len(deletedTags),
		"deleted_tags":   deletedTags,
		"not_found_tags": notFoundTags,
	}, nil
}

func (h *Service) GetOutboundGroups(ctx context.Context) (any, error) {
	cfg, err := h.configManager.GetOutboundsConfig()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	groups := make([]option.Outbound, 0)
	for _, outbound := range cfg.Outbounds {
		if outbound.Type == "selector" || outbound.Type == "urltest" {
			groups = append(groups, outbound)
		}
	}

	return map[string]any{"groups": groups}, nil
}

func (h *Service) UpdateOutboundMembers(ctx context.Context, body []byte, pathTag string) (any, error) {
	tag := pathTag

	type Request struct {
		Outbounds []string `json:"outbounds"`
	}

	var req Request
	if err := wirejson.Unmarshal(body, &req); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}

	err := h.configManager.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "outbound members updated successfully",
		"tag":     tag,
	}, nil
}
