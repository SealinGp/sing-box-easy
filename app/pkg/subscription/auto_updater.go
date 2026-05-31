package subscription

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// AutoUpdater handles automatic subscription updates
type AutoUpdater struct {
	subscriptionManager SubscriptionManager
	sublinkManager      *sublink.SubLink
	configManager       *config.Manager
	cron                *cron.Cron
	mutex               sync.RWMutex
	isRunning           bool
	lastCheckTime       time.Time
	updateStats         map[string]*UpdateStats
}

// UpdateStats tracks statistics for each subscription
type UpdateStats struct {
	LastUpdateAttempt time.Time
	LastSuccess       time.Time
	SuccessCount      int
	FailureCount      int
	LastError         string
}

// UpdateResult is the structured outcome of a single subscription refresh,
// shared by the cron-driven path and the manual-trigger HTTP route.
type UpdateResult struct {
	AddedTags   []string `json:"added_tags"`   // outbound tags that were newly inserted
	UpdatedTags []string `json:"updated_tags"` // outbound tags that were replaced in-place
	DeletedKeys []string `json:"deleted_keys"` // server keys removed (no longer present in subscription)
}

// Counts returns convenient totals for logging/response payloads.
func (r *UpdateResult) Counts() (added, updated, deleted int) {
	if r == nil {
		return 0, 0, 0
	}
	return len(r.AddedTags), len(r.UpdatedTags), len(r.DeletedKeys)
}

// NewAutoUpdater creates a new auto-updater instance
func NewAutoUpdater(configManager *config.Manager, subscriptionManager SubscriptionManager, sublinkManager *sublink.SubLink) *AutoUpdater {
	return &AutoUpdater{
		subscriptionManager: subscriptionManager,
		sublinkManager:      sublinkManager,
		configManager:       configManager,
		updateStats:         make(map[string]*UpdateStats),
	}
}

// Start starts the cron job for automatic updates
func (au *AutoUpdater) Start(cronExpression string) error {
	au.mutex.Lock()
	defer au.mutex.Unlock()

	if au.isRunning {
		return fmt.Errorf("auto-updater is already running")
	}

	// Default to every 5 minutes if no expression provided
	if cronExpression == "" {
		cronExpression = "*/5 * * * *"
	}

	au.cron = cron.New()
	_, err := au.cron.AddFunc(cronExpression, au.CheckSubscriptions)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	au.cron.Start()
	au.isRunning = true
	logger.Info("Auto-updater started", zap.String("cron", cronExpression))

	// Run an initial check
	go au.CheckSubscriptions()

	return nil
}

// Stop stops the cron job
func (au *AutoUpdater) Stop() {
	au.mutex.Lock()
	defer au.mutex.Unlock()

	if au.cron != nil {
		au.cron.Stop()
		au.isRunning = false
		logger.Info("Auto-updater stopped")
	}
}

// IsRunning returns whether the auto-updater is running
func (au *AutoUpdater) IsRunning() bool {
	au.mutex.RLock()
	defer au.mutex.RUnlock()
	return au.isRunning
}

// CheckSubscriptions checks all subscriptions and updates those that need it
func (au *AutoUpdater) CheckSubscriptions() {
	au.mutex.Lock()
	au.lastCheckTime = time.Now()
	au.mutex.Unlock()

	logger.Info("Starting subscription check")

	// List all subscriptions
	subscriptions, err := au.subscriptionManager.List()
	logger.Info("subscriptions list", zap.Any("subscriptions", subscriptions))
	if err != nil {
		logger.Error("Failed to list subscriptions", zap.Error(err))
		return
	}

	// Process each subscription sequentially to avoid config conflicts
	updatedCount := 0
	failedCount := 0

	for _, sub := range subscriptions {
		// Skip disabled or non-auto-update subscriptions
		if !sub.Enabled || !sub.AutoUpdate {
			continue
		}

		// Check if it's time to update
		if !au.shouldUpdate(sub) {
			logger.Debug("it's not time to update")
			continue
		}

		logger.Info("Updating subscription", zap.String("id", sub.ID), zap.String("name", sub.Name))

		// Update subscription sequentially to ensure config integrity.
		// UpdateSubscription records success/failure stats internally so both
		// the cron path and the manual route path stay consistent.
		result, err := au.UpdateSubscription(sub)
		if err != nil {
			failedCount++
			logger.Error("Failed to update subscription",
				zap.String("id", sub.ID),
				zap.String("name", sub.Name),
				zap.Error(err))
			continue
		}
		updatedCount++
		added, updated, deleted := result.Counts()
		logger.Info("Successfully updated subscription",
			zap.String("id", sub.ID),
			zap.String("name", sub.Name),
			zap.Int("added", added),
			zap.Int("updated", updated),
			zap.Int("deleted", deleted))
	}

	logger.Info("Subscription check completed",
		zap.Int("updated", updatedCount),
		zap.Int("failed", failedCount))
}

// shouldUpdate checks if a subscription needs updating
func (au *AutoUpdater) shouldUpdate(sub *Subscription) bool {
	// Parse update interval
	intervalDuration := parseDuration(sub.UpdateInterval)
	if intervalDuration <= 0 {
		intervalDuration = 24 * time.Hour // Default to 24 hours
	}

	// Check if enough time has passed since last update
	return time.Since(sub.LastUpdate) >= intervalDuration
}

// RefreshByID is the canonical entry point for refreshing a single subscription
// regardless of trigger source (HTTP route, cron, or other callers).
// It fetches the subscription record, runs the same diff/apply path as the cron
// loop, and records success/failure stats. The manual route handler should call
// this instead of touching configManager.UpdateOutbounds directly.
func (au *AutoUpdater) RefreshByID(id string) (*UpdateResult, error) {
	sub, err := au.subscriptionManager.Get(id)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}
	return au.UpdateSubscription(sub)
}

// UpdateSubscription updates a single subscription: fetch → diff → apply.
// Records success/failure stats internally so the cron loop and the manual
// route both produce identical bookkeeping.
func (au *AutoUpdater) UpdateSubscription(sub *Subscription) (result *UpdateResult, err error) {
	defer func() {
		if err != nil {
			au.recordFailure(sub.ID, err)
		} else {
			au.recordSuccess(sub.ID)
		}
	}()

	// Step 1: Fetch new nodes from subscription URL
	lines := []string{sub.URL}
	newNodes, err := au.sublinkManager.ListNodes(lines)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nodes: %w", err)
	}

	if len(newNodes) == 0 {
		return nil, fmt.Errorf("no nodes returned from subscription")
	}

	// Step 2: Get current configuration
	cfg, err := au.configManager.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Step 3: Compare and diff nodes
	toDelete, toAdd, toUpdate := au.diffNodes(cfg, newNodes)

	// Build the result up-front from the diff so callers (route handler,
	// cron logger) get the same shape regardless of whether any change was
	// applied this round.
	result = &UpdateResult{
		AddedTags:   collectTags(toAdd),
		UpdatedTags: collectTagsFromMap(toUpdate),
		DeletedKeys: collectKeys(toDelete),
	}

	// Step 4: Apply changes (skip the write if the diff is empty)
	if len(toDelete) > 0 || len(toAdd) > 0 || len(toUpdate) > 0 {
		if applyErr := au.applyChanges(cfg, toDelete, toAdd, toUpdate, sub.ID); applyErr != nil {
			err = fmt.Errorf("failed to apply changes: %w", applyErr)
			return nil, err
		}
	}

	// Step 5: Update subscription's last update time. A failure here doesn't
	// roll back the config — we just log it.
	if uerr := au.subscriptionManager.UpdateLastUpdate(sub.ID); uerr != nil {
		logger.Warn("Failed to update last update time", zap.String("id", sub.ID), zap.Error(uerr))
	}

	return result, nil
}

// collectTags returns the Tag of every outbound in the slice.
func collectTags(outbounds []config.Outbound) []string {
	if len(outbounds) == 0 {
		return nil
	}
	tags := make([]string, 0, len(outbounds))
	for _, o := range outbounds {
		tags = append(tags, o.Tag)
	}
	return tags
}

// collectTagsFromMap returns the Tag of every outbound in the map.
func collectTagsFromMap(m map[string]config.Outbound) []string {
	if len(m) == 0 {
		return nil
	}
	tags := make([]string, 0, len(m))
	for _, o := range m {
		tags = append(tags, o.Tag)
	}
	return tags
}

// collectKeys returns the keys of a set-style map (struct{} values).
func collectKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// diffNodes compares current outbounds with the freshly fetched nodes and
// returns the changes, keyed by the existing outbound's unique tag.
//
// Identity is the unique tag ("name server:port"), NOT the bare server:port.
// Many providers put dozens of distinct nodes behind a single relay/CDN
// endpoint (same host:port, different display names), so server:port is not a
// unique node identity — keying by it collapses those nodes together and makes
// a single update overwrite every co-located node with the same tag (a
// duplicate-tag explosion that fails `sing-box check`). The unique tag is what
// sing-box already requires to be unique, so it is the correct key.
//
//   - toDelete: tags of this subscription's outbounds no longer in the feed.
//   - toUpdate: existing tag -> replacement outbound (the value's Tag may differ
//     from the key only for the legacy-rename case below, which is applied as a
//     group-reference rename in applyChanges).
//   - toAdd:    feed nodes with no matching existing outbound.
func (au *AutoUpdater) diffNodes(cfg *config.SingBoxConfig, newNodes []*node.SubNode) (toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound) {
	toDelete = make(map[string]struct{})
	toUpdate = make(map[string]config.Outbound)

	// Index the feed by unique tag. subServers is the set of hosts in the feed,
	// used to decide which existing outbounds belong to this subscription.
	newByTag := make(map[string]config.Outbound)
	subServers := make(map[string]struct{})
	for _, n := range newNodes {
		outbound := config.Outbound{
			Tag:     n.Tag,
			Type:    n.Type,
			Options: n.Options,
		}
		if svr := config.GetOutboundServer(outbound); svr != "" {
			subServers[svr] = struct{}{}
		}
		if config.GetOutboundServerKey(outbound) == "" {
			continue
		}
		uniqueTag := config.GenerateUniqueTag(n.Tag, outbound)
		outbound.Tag = uniqueTag
		// Two feed nodes with the same name AND endpoint are genuinely
		// indistinguishable; collapsing them here is correct (one survives).
		newByTag[uniqueTag] = outbound
	}

	// consumed tracks which feed nodes (by unique tag) were matched to an
	// existing outbound, so the leftovers become additions.
	consumed := make(map[string]bool)

	for _, outbound := range cfg.Outbounds {
		// Only consider outbounds belonging to this subscription (host in feed).
		if _, ok := subServers[config.GetOutboundServer(outbound)]; !ok {
			continue
		}

		// Exact identity match on the unique tag.
		if newOutbound, ok := newByTag[outbound.Tag]; ok && !consumed[outbound.Tag] {
			consumed[outbound.Tag] = true
			if !outboundsDeepEqual(outbound, newOutbound) {
				toUpdate[outbound.Tag] = newOutbound
			}
			continue
		}

		// Legacy match: older imports stored the bare display name without the
		// " server:port" suffix. Recognize "<tag> <server:port>" so such a node
		// is renamed in place (preserving its selector-group memberships)
		// instead of being deleted and re-added under a new tag.
		if sk := config.GetOutboundServerKey(outbound); sk != "" {
			candidate := outbound.Tag + " " + sk
			if newOutbound, ok := newByTag[candidate]; ok && !consumed[candidate] {
				consumed[candidate] = true
				toUpdate[outbound.Tag] = newOutbound
				continue
			}
		}

		// Belongs to this subscription but absent from the feed → delete.
		toDelete[outbound.Tag] = struct{}{}
	}

	for tag, outbound := range newByTag {
		if !consumed[tag] {
			toAdd = append(toAdd, outbound)
		}
	}

	return toDelete, toAdd, toUpdate
}

// applyChanges applies the calculated changes to the configuration.
//
// In addition to the outbound list itself, this also rewrites every
// selector/urltest group outbound so it no longer references tags that were
// deleted, and picks up the new tag for any outbound that was renamed by an
// update (the server endpoint survives but the human-facing tag changed).
// Without this pass, `sing-box check` would still pass but selector groups
// would silently keep dangling tags pointing at gone nodes.
//
// NOTE on concurrency: the diff (toDelete/toAdd/toUpdate) is computed against
// a snapshot read by an earlier GetConfig call, while UpdateConfig re-reads the
// config from disk. The deletedTags/renameMap built below are derived from the
// *fresh* snapshot inside the closure, so the group-reference rewrite is always
// internally consistent. The remaining TOCTOU window between the outer diff and
// the inner write is a pre-existing concern in this codebase (Manager has no
// lock); fixing it would require pushing diffNodes into the closure too.
func (au *AutoUpdater) applyChanges(cfg *config.SingBoxConfig, toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound, subID string) error {
	return au.configManager.UpdateConfig(func(c *config.SingBoxConfig) error {
		// Create new outbounds slice
		newOutbounds := make([]config.Outbound, 0, len(c.Outbounds)-len(toDelete)+len(toAdd))

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
		// selector/urltest group outbounds.
		newOutbounds = config.PruneGroupReferences(newOutbounds, deletedTags, renameMap)

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
}

// outboundsDeepEqual performs deep comparison of two outbounds via JSON.
// The same options struct may be a typed sing-box option type on one side and
// a map on the other (e.g. existing outbound from typed registry vs. new node
// from parser); marshalling both to JSON normalizes them before compare.
func outboundsDeepEqual(a, b config.Outbound) bool {
	if a.Type != b.Type {
		return false
	}
	aJSON, err1 := json.Marshal(a.Options)
	bJSON, err2 := json.Marshal(b.Options)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// parseDuration parses duration string like "24h", "7d", "2w", "1mo".
// Falls back to 24h on any parse failure.
func parseDuration(s string) time.Duration {
	const defaultInterval = 24 * time.Hour

	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return defaultInterval
	}

	// Try standard duration parsing first (handles ns, us, ms, s, m, h)
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	// Extract leading numeric prefix and trailing unit suffix
	splitIdx := -1
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			splitIdx = i
			break
		}
	}
	if splitIdx <= 0 {
		return defaultInterval
	}

	n, err := strconv.Atoi(s[:splitIdx])
	if err != nil || n <= 0 {
		return defaultInterval
	}
	unit := s[splitIdx:]

	switch unit {
	case "d", "day", "days":
		return time.Duration(n) * 24 * time.Hour
	case "w", "week", "weeks":
		return time.Duration(n) * 24 * 7 * time.Hour
	case "mo", "month", "months":
		return time.Duration(n) * 24 * 30 * time.Hour
	default:
		return defaultInterval
	}
}

// recordSuccess records a successful update
func (au *AutoUpdater) recordSuccess(subID string) {
	au.mutex.Lock()
	defer au.mutex.Unlock()

	if au.updateStats[subID] == nil {
		au.updateStats[subID] = &UpdateStats{}
	}

	stats := au.updateStats[subID]
	stats.LastUpdateAttempt = time.Now()
	stats.LastSuccess = time.Now()
	stats.SuccessCount++
	stats.LastError = ""
}

// recordFailure records a failed update
func (au *AutoUpdater) recordFailure(subID string, err error) {
	au.mutex.Lock()
	defer au.mutex.Unlock()

	if au.updateStats[subID] == nil {
		au.updateStats[subID] = &UpdateStats{}
	}

	stats := au.updateStats[subID]
	stats.LastUpdateAttempt = time.Now()
	stats.FailureCount++
	stats.LastError = err.Error()
}

// GetStats returns update statistics
func (au *AutoUpdater) GetStats() map[string]*UpdateStats {
	au.mutex.RLock()
	defer au.mutex.RUnlock()

	// Return a copy to avoid race conditions
	statsCopy := make(map[string]*UpdateStats)
	for k, v := range au.updateStats {
		statsCopy[k] = &UpdateStats{
			LastUpdateAttempt: v.LastUpdateAttempt,
			LastSuccess:       v.LastSuccess,
			SuccessCount:      v.SuccessCount,
			FailureCount:      v.FailureCount,
			LastError:         v.LastError,
		}
	}
	return statsCopy
}

// GetLastCheckTime returns the last time subscriptions were checked
func (au *AutoUpdater) GetLastCheckTime() time.Time {
	au.mutex.RLock()
	defer au.mutex.RUnlock()
	return au.lastCheckTime
}

// TriggerCheck manually triggers a subscription check
func (au *AutoUpdater) TriggerCheck() {
	go au.CheckSubscriptions()
}
