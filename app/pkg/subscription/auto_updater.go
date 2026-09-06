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
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// NodeRulesProvider supplies the current Outbound Node Rules (Filters + Groups)
// used to auto-organize freshly-fetched nodes on each subscription update. It is
// an interface (not the concrete manager) so the updater stays testable and the
// rules feature can be absent (nil) without breaking subscription updates.
type NodeRulesProvider interface {
	ListFilters() ([]*noderules.Filter, error)
	ListGroups() ([]*noderules.Group, error)
}

// AutoUpdater handles automatic subscription updates
type AutoUpdater struct {
	subscriptionManager SubscriptionManager
	sublinkManager      *sublink.SubLink
	configManager       *config.Manager
	nodeRules           NodeRulesProvider
	infoKeywords        InfoKeywordsProvider
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

// NewAutoUpdater creates a new auto-updater instance. nodeRules may be nil, in
// which case the legacy "append new nodes into non-group collections" behavior
// is used; when present, the Outbound Node Rules engine owns node placement.
// infoKeywords may also be nil, in which case the built-in
// DefaultInfoLabelKeywords decide which feed entries are account metadata.
func NewAutoUpdater(
	configManager *config.Manager,
	subscriptionManager SubscriptionManager,
	sublinkManager *sublink.SubLink,
	nodeRules NodeRulesProvider,
	infoKeywords InfoKeywordsProvider,
) *AutoUpdater {
	return &AutoUpdater{
		subscriptionManager: subscriptionManager,
		sublinkManager:      sublinkManager,
		configManager:       configManager,
		nodeRules:           nodeRules,
		infoKeywords:        infoKeywords,
		updateStats:         make(map[string]*UpdateStats),
	}
}

// infoLabelKeywords resolves the keyword list for this refresh. It is read per
// update (not cached) so an edit on the Subscriptions page takes effect on the
// very next refresh without a restart.
func (au *AutoUpdater) infoLabelKeywords() []string {
	if au.infoKeywords == nil {
		return DefaultInfoLabelKeywords
	}
	return EffectiveInfoKeywords(au.infoKeywords.GetSubscriptionInfoKeywords())
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

	// Delay the initial check by 30 s so the HTTP server is fully up and any
	// in-flight config edits the operator made just before restarting the
	// service are not immediately overwritten by a subscription refresh.
	go func() {
		time.Sleep(30 * time.Second)
		au.CheckSubscriptions()
	}()

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
		// Skip subscriptions that don't have auto-update enabled
		if !sub.AutoUpdate {
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

	// Step 1: Fetch new nodes (and account metadata) from the subscription URL,
	// honoring the per-subscription fetch strategy (direct / clean-DNS / proxy)
	// for censored networks.
	lines := []string{sub.URL}
	newNodes, meta, err := au.sublinkManager.ListNodesWithMetaOpts(lines, sublink.FetchOptions{
		Mode:     sub.FetchMode,
		ProxyURL: sub.ProxyURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nodes: %w", err)
	}

	if len(newNodes) == 0 {
		return nil, fmt.Errorf("no nodes returned from subscription")
	}

	// Step 1b: Collect account metadata from BOTH sources (providers use one or
	// the other): the standard `Subscription-Userinfo` HTTP header, and the
	// in-feed loopback "info nodes". Header entries come first. Splitting also
	// keeps info nodes out of the config — any existing loopback outbounds from
	// this subscription are no longer in `realNodes`, so the diff deletes them.
	realNodes, nodeInfo := partitionNodes(newNodes, au.infoLabelKeywords())
	info := parseUserinfo(meta.Userinfo)
	info = append(info, nodeInfo...)

	// Step 2: Get current configuration
	cfg, err := au.configManager.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Step 3: Compare and diff nodes
	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, realNodes)

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

	// Step 6: Persist the extracted account metadata (traffic/expiry/reset).
	// Always written (even empty) so stale info clears; a failure here is logged
	// but does not roll back the applied config.
	if ierr := au.subscriptionManager.UpdateInfo(sub.ID, info); ierr != nil {
		logger.Warn("Failed to update subscription info", zap.String("id", sub.ID), zap.Error(ierr))
	}

	// Step 7: Fill in the provider's own site, if the feed named one and the
	// field is still empty. Only when empty: an operator who corrected the link
	// (providers move domains, and a mirror often reports the old one) must not
	// have that correction undone by the next refresh.
	if site := officialURLToPersist(sub.OfficialURL, meta.SiteURL, info); site != "" {
		if serr := au.subscriptionManager.UpdateOfficialURL(sub.ID, site); serr != nil {
			logger.Warn("Failed to update subscription official url",
				zap.String("id", sub.ID), zap.Error(serr))
		}
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

// subscriptionTagSeparator joins the human-facing unique tag with its owning
// subscription's ID. It is visually distinct and unlikely to appear inside a
// provider-supplied node name, so it can be used to recognize ownership.
const subscriptionTagSeparator = " | "

// subscriptionTagSuffix returns the ownership suffix for a subscription, e.g.
// " | sub_1778669376". The ID is appended (not prepended) because it exists
// only to machine-identify the owning subscription — keeping it at the end
// leaves the human-readable "<name> <server:port>" at the front of the tag.
// Every node fetched from this subscription is tagged as
// "<name> <server:port> | <subID>".
func subscriptionTagSuffix(subID string) string {
	return subscriptionTagSeparator + subID
}

// TagBelongsToSubscription reports whether an outbound tag was minted for the
// given subscription (i.e. it carries that subscription's ID suffix). This is
// the authoritative ownership test: it holds no matter how the provider has
// since renamed the node in front of the suffix.
//
// Exported because it is the ONE rule that decides which outbounds a
// subscription is answerable for, and a second consumer now needs it: the
// quality prober (app/pkg/subprobe) groups nodes by subscription to compute
// availability. A prober carrying its own copy of the suffix rule would keep
// measuring the wrong nodes for a while after this one changed.
func TagBelongsToSubscription(tag, subID string) bool {
	return strings.HasSuffix(tag, subscriptionTagSuffix(subID))
}

// fingerprintLegacyTag rewrites a pre-fingerprint subscription tag
// ("<name> <server:port> | <subID>") into the current form
// ("<name> <fingerprint> | <subID>"), or returns "" when the tag is not in the
// legacy shape and therefore needs no migration.
//
// The endpoint is recovered from the tag itself rather than from the outbound's
// options on purpose: the tag records the endpoint the node had WHEN IT WAS
// MINTED, and matching is against tags the feed produces from those same
// strings. Reading today's options would silently mis-migrate a node whose
// provider moved it to a new host.
func fingerprintLegacyTag(tag, subID string) string {
	suffix := subscriptionTagSuffix(subID)
	if !strings.HasSuffix(tag, suffix) {
		return ""
	}
	head := strings.TrimSuffix(tag, suffix)

	space := strings.LastIndex(head, " ")
	if space == -1 {
		return ""
	}
	endpoint := head[space+1:]
	// Only "host:port" (or a bare host) is a legacy endpoint. Anything else —
	// including a tag already carrying a fingerprint — is left alone.
	if !strings.Contains(endpoint, ":") && !strings.Contains(endpoint, ".") {
		return ""
	}
	fp := config.FingerprintEndpointKey(endpoint)
	if fp == "" || fp == endpoint {
		return ""
	}
	return head[:space+1] + fp + suffix
}

// diffNodes compares current outbounds with the freshly fetched nodes and
// returns the changes, keyed by the existing outbound's tag.
//
// Ownership is determined by the subscription-ID suffix on the tag, NOT by the
// node's server host. Every node fetched from a subscription is tagged as
// "<name> <server:port> | <subID>", so the suffix unambiguously enumerates all
// of a subscription's nodes even when the provider renames them or co-locates
// many distinct nodes behind one relay/CDN endpoint. The leading
// "name server:port" keeps tags unique within the namespace (sing-box requires
// globally unique tags), so it remains the per-node identity in front of the
// suffix.
//
//   - toDelete: tags of this subscription's outbounds no longer in the feed
//     (including nodes whose name changed — the old tag is deleted and the new
//     one is added, since the name is part of the identity).
//   - toUpdate: existing tag -> replacement outbound. The value's Tag differs
//     from the key only for the legacy-migration case (an un-suffixed outbound
//     adopted into the namespace), which is applied as a group-reference rename
//     in applyChanges.
//   - toAdd:    feed nodes with no matching existing outbound.
func (au *AutoUpdater) diffNodes(cfg *config.SingBoxConfig, sub *Subscription, newNodes []*node.SubNode) (toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound) {
	toDelete = make(map[string]struct{})
	toUpdate = make(map[string]config.Outbound)

	suffix := subscriptionTagSuffix(sub.ID)

	// Index the feed by its final (suffixed) unique tag. subServers records the
	// feed's server hosts purely so legacy un-suffixed outbounds from this
	// subscription can still be recognized and migrated into the namespace.
	newByTag := make(map[string]config.Outbound, len(newNodes))
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
		// Final tag = "<name> <endpoint-fingerprint> | <subID>".
		taggedTag := config.GenerateFingerprintedTag(n.Tag, outbound) + suffix
		outbound.Tag = taggedTag
		// Two feed nodes with the same name AND endpoint are genuinely
		// indistinguishable; collapsing them here is correct (one survives).
		newByTag[taggedTag] = outbound
	}

	// consumed tracks which feed nodes (by suffixed tag) were matched to an
	// existing outbound, so the leftovers become additions.
	consumed := make(map[string]bool)

	// Pass 1: outbounds already in this subscription's namespace. The suffix is
	// the sole ownership signal here, so a node that was renamed by the provider
	// (new tag) is correctly seen as a delete of the old tag plus an add.
	for _, outbound := range cfg.Outbounds {
		if !TagBelongsToSubscription(outbound.Tag, sub.ID) {
			continue
		}
		if newOutbound, ok := newByTag[outbound.Tag]; ok && !consumed[outbound.Tag] {
			consumed[outbound.Tag] = true
			if !outboundsDeepEqual(outbound, newOutbound) {
				toUpdate[outbound.Tag] = newOutbound
			}
			continue
		}

		// Tags minted before the endpoint was hashed spell the endpoint out:
		// "<name> <server:port> | <subID>". Convert and retry, so the format
		// change lands as a RENAME rather than as delete-plus-add. The
		// difference is not cosmetic: a rename carries selector/urltest
		// membership across (see renameMap in applyChanges), while a delete
		// would drop every group reference to the node and re-add it bare.
		if migrated := fingerprintLegacyTag(outbound.Tag, sub.ID); migrated != "" {
			if newOutbound, ok := newByTag[migrated]; ok && !consumed[migrated] {
				consumed[migrated] = true
				toUpdate[outbound.Tag] = newOutbound
				continue
			}
		}

		// Carries our suffix but no longer in the feed → delete.
		toDelete[outbound.Tag] = struct{}{}
	}

	// Pass 2: migration for outbounds added before tag-suffixing existed (or
	// added manually). Recognize them by server host and re-tag them into the
	// namespace so future passes can rely solely on the suffix. Running after
	// pass 1 guarantees already-suffixed nodes win the match for a feed entry.
	for _, outbound := range cfg.Outbounds {
		if TagBelongsToSubscription(outbound.Tag, sub.ID) {
			continue
		}
		// Only outbounds whose server is present in this feed are candidates,
		// matching the historical host-based ownership heuristic.
		if _, ok := subServers[config.GetOutboundServer(outbound)]; !ok {
			continue
		}

		// The legacy tag is a bare display name, or already carries an endpoint
		// ("name server:port"). The feed index is keyed by the current form
		// ("name fingerprint | subID"), so every candidate is built in that
		// form: the bare tag, the tag plus this outbound's own fingerprint, and
		// the tag with an endpoint it already spells out converted.
		candidates := []string{outbound.Tag + suffix}
		if fp := config.FingerprintEndpointKey(config.GetOutboundServerKey(outbound)); fp != "" {
			candidates = append(candidates, outbound.Tag+" "+fp+suffix)
		}
		if migrated := fingerprintLegacyTag(outbound.Tag+suffix, sub.ID); migrated != "" {
			candidates = append(candidates, migrated)
		}
		matched := false
		for _, candidate := range candidates {
			if newOutbound, ok := newByTag[candidate]; ok && !consumed[candidate] {
				consumed[candidate] = true
				// Rename in place (legacy tag -> suffixed tag) so selector/urltest
				// group memberships are preserved via the rename map in applyChanges.
				toUpdate[outbound.Tag] = newOutbound
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Legacy node from this subscription's host set but absent from the feed → delete.
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
		rebuilt, rerr := au.rebuildNodeRules(newOutbounds, subID)
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
}

// rebuildNodeRules regenerates the rule-managed Filter/Group outbounds from the
// current rule set and the endpoints present in `outbounds`. Returns the new
// outbound list. When no rules provider is configured it returns the input
// unchanged (the legacy path already handled additions).
func (au *AutoUpdater) rebuildNodeRules(outbounds []config.Outbound, subID string) ([]config.Outbound, error) {
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
