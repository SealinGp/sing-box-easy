package outbounds

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/nodetag"
	"go.uber.org/zap"
)

// firstExistingTag returns the first candidate tag already present in the
// config, if any.
func firstExistingTag(existingTags map[string]bool, candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if existingTags[candidate] {
			return candidate, true
		}
	}
	return "", false
}

// addOutbounds adds multiple outbounds, skipping duplicates based on existing tags.
//
// If every input outbound is already present (or the input slice is empty),
// the function returns without touching disk — there is no point running the
// write-then-validate cycle on a config that has not changed, and doing so
// would surface unrelated pre-existing validation failures.
func addOutbounds(ctx context.Context, m *config.Manager, outbounds []config.Outbound) (addedTags []string, skippedTags []string, err error) {
	if len(outbounds) == 0 {
		return nil, nil, nil
	}

	// Pre-flight: compute the diff against the current on-disk config so we
	// can decide whether a save is necessary before we touch anything.
	current, err := m.GetOutboundsConfig()
	if err != nil {
		return nil, nil, err
	}
	existingTags := make(map[string]bool, len(current.Outbounds))
	for _, existing := range current.Outbounds {
		existingTags[existing.Tag] = true
	}

	toAdd := make([]config.Outbound, 0, len(outbounds))
	for _, outbound := range outbounds {
		// The first candidate is the tag to mint; the rest are older shapes the
		// same node may already be stored under, so re-adding stays a skip rather
		// than producing a second copy under a new tag.
		candidates := nodetag.OutboundTagCandidates(outbound.Tag, outbound)
		outbound.Tag = candidates[0]

		if existing, found := firstExistingTag(existingTags, candidates); found {
			skippedTags = append(skippedTags, existing)
			continue
		}
		toAdd = append(toAdd, outbound)
		addedTags = append(addedTags, outbound.Tag)
	}

	if len(toAdd) == 0 {
		logger.Warn("all outbounds already exist; skipping save",
			zap.Int("input", len(outbounds)),
			zap.Int("skipped", len(skippedTags)))
		return addedTags, skippedTags, nil
	}

	err = m.UpdateOutboundsConfig(ctx, func(cfg *config.SingBoxConfig) error {
		cfg.Outbounds = append(cfg.Outbounds, toAdd...)
		return nil
	})
	return
}
