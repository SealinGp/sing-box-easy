package outbounds

import (
	"context"
	"errors"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
	"go.uber.org/zap"
)

type RulesProvider interface {
	ListFilters() ([]*noderules.Filter, error)
	ListGroups() ([]*noderules.Group, error)
}
type Reconciler struct {
	configManager *config.Manager
	nodeRules     RulesProvider
}
type Changes struct {
	Delete map[string]struct{}
	Add    []config.Outbound
	Update map[string]config.Outbound
}

var errUnchanged = errors.New("outbounds unchanged")

func NewReconciler(cm *config.Manager, rules RulesProvider) *Reconciler {
	return &Reconciler{cm, rules}
}
func collectTags(nodes []config.Outbound) []string {
	tags := make([]string, 0, len(nodes))
	for _, n := range nodes {
		tags = append(tags, n.Tag)
	}
	return tags
}

// applyChanges applies the calculated changes to the configuration.
//
// In addition to the outbound list itself, this also rewrites every
// selector/urltest group outbound so it no longer references tags that were
// deleted, and picks up the new tag for any outbound that was renamed by an
// update (the server endpoint survives but the human-facing tag changed).
// Without this pass, `sing-box check` would still pass but selector groups
// would silently keep dangling tags pointing at gone nodes.
func (au *Reconciler) Apply(ctx context.Context, subID string, build func(*config.SingBoxConfig) Changes) error {
	err := au.configManager.UpdateOutboundsConfig(ctx, func(c *config.SingBoxConfig) error {
		changes := build(c)
		toDelete, toAdd, toUpdate := changes.Delete, changes.Add, changes.Update
		if len(toDelete)+len(toAdd)+len(toUpdate) == 0 {
			return errUnchanged
		}

		// Create new outbounds slice
		newOutbounds := make([]config.Outbound, 0, len(c.Outbounds)+len(toAdd))

		// Tags collected while filtering — needed to scrub stale references
		// from selector/urltest groups after the rebuild.
		deletedTags := make(map[string]struct{})
		renameMap := make(map[string]string)

		// emitted guards against duplicate tags ever reaching the config (which
		// would fail `sing-box check`): the final outbound list has at most one
		// entry per tag. Keyed by the tag actually written to newOutbounds.
		emitted := make(map[string]bool)

		// First pass: keep non-deleted outbounds and apply updates. Identity is
		// the outbound tag (matching diffNodes), not server:port.
		for _, outbound := range c.Outbounds {
			if _, ok := toDelete[outbound.Tag]; ok {
				deletedTags[outbound.Tag] = struct{}{}
				continue // Skip deleted
			}

			if updatedOutbound, ok := toUpdate[outbound.Tag]; ok {
				if outbound.Tag != updatedOutbound.Tag {
					renameMap[outbound.Tag] = updatedOutbound.Tag
				}
				// Multiple existing outbounds can resolve to the same updated
				// node (e.g. pre-existing duplicates or legacy renames); emit it
				// only once so we never write a duplicate tag.
				if emitted[updatedOutbound.Tag] {
					continue
				}
				emitted[updatedOutbound.Tag] = true
				newOutbounds = append(newOutbounds, updatedOutbound)
				continue
			}

			// Keep existing outbound, dropping any stray duplicate-tag copy.
			if emitted[outbound.Tag] {
				logger.Warn("dropping duplicate outbound tag during subscription update",
					zap.String("subscription", subID), zap.String("tag", outbound.Tag))
				continue
			}
			emitted[outbound.Tag] = true
			newOutbounds = append(newOutbounds, outbound)
		}

		// Add new outbounds (same duplicate guard).
		for _, outbound := range toAdd {
			if emitted[outbound.Tag] {
				logger.Warn("dropping duplicate outbound tag during subscription update",
					zap.String("subscription", subID), zap.String("tag", outbound.Tag))
				continue
			}
			emitted[outbound.Tag] = true
			newOutbounds = append(newOutbounds, outbound)
		}

		// Strip references to deleted tags and rewrite renamed tags inside
		// selector/urltest outbounds. When the Outbound Node Rules engine is
		// active it owns node placement, so we do NOT append new nodes here
		// (addTags=nil); the rebuild below assigns them. Without rules, fall back
		// to the legacy "append into non-group collections" behavior.
		addTags := []string(nil)
		if au.nodeRules == nil {
			addTags = collectTags(toAdd)
		}
		newOutbounds = config.PruneGroupReferences(newOutbounds, deletedTags, renameMap, addTags)

		// Rules-driven rebuild: reassign every endpoint to its matching Filters
		// and regenerate Filter/Group outbounds from the current rule set. This
		// is a full, deterministic rebuild from (endpoints + rules), so it
		// handles adds/deletes/renames and multiple subscriptions uniformly.
		rebuilt, rerr := au.Rebuild(newOutbounds, subID)
		if rerr != nil {
			// A rules failure must not abandon the subscription update; log and
			// keep the pruned outbounds as-is.
			logger.Warn("node-rules rebuild skipped", zap.String("subscription", subID), zap.Error(rerr))
		} else {
			newOutbounds = rebuilt
		}

		// Update the configuration
		c.Outbounds = newOutbounds

		logger.Info("Applied subscription changes",
			zap.String("subscription", subID),
			zap.Int("deleted", len(toDelete)),
			zap.Int("added", len(toAdd)),
			zap.Int("updated", len(toUpdate)),
			zap.Int("group_refs_renamed", len(renameMap)))

		return nil
	})
	if errors.Is(err, errUnchanged) {
		return nil
	}
	return err
}

// Rebuild regenerates the rule-managed Filter/Group outbounds from the
// current rule set and the endpoints present in `outbounds`. Returns the new
// outbound list. When no rules provider is configured it returns the input
// unchanged (the legacy path already handled additions).
func (au *Reconciler) Rebuild(outbounds []config.Outbound, subID string) ([]config.Outbound, error) {
	if au.nodeRules == nil {
		return outbounds, nil
	}
	filters, err := au.nodeRules.ListFilters()
	if err != nil {
		return nil, fmt.Errorf("failed to list filters: %w", err)
	}
	groups, err := au.nodeRules.ListGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	pool := noderules.NodePool{
		Endpoints: config.EndpointTags(outbounds),
		OptIn:     config.OptInTags(outbounds),
	}
	filterSpecs, groupSpecs, _, others := noderules.BuildSpecs(filters, groups, pool)
	rebuilt := config.BuildGroupOutbounds(outbounds, filterSpecs, groupSpecs)

	logger.Info("Rebuilt node-rules groups",
		zap.String("subscription", subID),
		zap.Int("endpoints", len(pool.Endpoints)),
		zap.Int("filters", len(filterSpecs)),
		zap.Int("groups", len(groupSpecs)),
		zap.Int("unmatched", len(others)))
	return rebuilt, nil
}
