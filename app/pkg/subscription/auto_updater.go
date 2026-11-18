package subscription

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds/rules"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/autoupdate"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/probe"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
	"go.uber.org/zap"
)

// NodeRulesProvider supplies the current Outbound Node Rules (Filters + Groups)
// used to auto-organize freshly-fetched nodes on each subscription update. It is
// an interface (not the concrete manager) so the updater stays testable and the
// rules feature can be absent (nil) without breaking subscription updates.
type NodeRulesProvider interface {
	ListFilters() ([]*rules.Filter, error)
	ListGroups() ([]*rules.Group, error)
}

// Service handles automatic subscription updates
type Service struct {
	probeRunner      *probe.Runner
	probeStore       *repo.ProbeStore
	nodeHistory      *nodehistory.Store
	settingsManager  *settings.ManagerXORM
	repository       repo.Repository
	sublinkManager   feedResolver
	configManager    *config.Manager
	serviceRestarter ServiceRestarter
	nodeRules        NodeRulesProvider
	infoKeywords     InfoKeywordsProvider
	scheduler        *autoupdate.Scheduler
	operationLocks   sync.Map
	mutex            sync.RWMutex
	lastCheckTime    time.Time
	updateStats      map[string]*UpdateStats
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
	Restarted   bool     `json:"restarted"`    // running sing-box has loaded the refreshed config
}

// Counts returns convenient totals for logging/response payloads.
func (r *UpdateResult) Counts() (added, updated, deleted int) {
	if r == nil {
		return 0, 0, 0
	}
	return len(r.AddedTags), len(r.UpdatedTags), len(r.DeletedKeys)
}

// NewService creates a new auto-updater instance. serviceRestarter must be
// configured so a successful refresh can be applied to the running sing-box.
// nodeRules may be nil, in
// which case the legacy "append new nodes into non-group collections" behavior
// is used; when present, the Outbound Node Rules engine owns node placement.
// infoKeywords may also be nil, in which case the built-in
// DefaultInfoLabelKeywords decide which feed entries are account metadata.
func newService(
	configManager *config.Manager,
	repository repo.Repository,
	sublinkManager feedResolver,
	nodeRules NodeRulesProvider,
	infoKeywords InfoKeywordsProvider,
	serviceRestarter ServiceRestarter,
) *Service {
	s := &Service{
		repository:       repository,
		sublinkManager:   sublinkManager,
		configManager:    configManager,
		serviceRestarter: serviceRestarter,
		nodeRules:        nodeRules,
		infoKeywords:     infoKeywords,
		updateStats:      make(map[string]*UpdateStats),
	}
	s.scheduler = autoupdate.New(s.checkSubscriptions)
	return s
}

// infoLabelKeywords resolves the keyword list for this refresh. It is read per
// update (not cached) so an edit on the Subscriptions page takes effect on the
// very next refresh without a restart.
func (au *Service) infoLabelKeywords() []string {
	if au.infoKeywords == nil {
		return DefaultInfoLabelKeywords
	}
	return EffectiveInfoKeywords(au.infoKeywords.GetSubscriptionInfoKeywords())
}

func (s *Service) Start(expression string) error { return s.scheduler.Start(expression) }
func (s *Service) Stop() {
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
}
func (s *Service) IsRunning() bool { return s.scheduler != nil && s.scheduler.Running() }

// CheckSubscriptions checks all subscriptions and updates those that need it
func (au *Service) CheckSubscriptions() { au.checkSubscriptions(context.Background()) }
func (au *Service) checkSubscriptions(ctx context.Context) {
	au.mutex.Lock()
	au.lastCheckTime = time.Now()
	au.mutex.Unlock()

	logger.Info("Starting subscription check")

	// List all subscriptions
	subscriptions, err := au.repository.List()
	logger.Info("subscriptions list", zap.Any("subscriptions", subscriptions))
	if err != nil {
		logger.Error("Failed to list subscriptions", zap.Error(err))
		return
	}

	// Process each subscription sequentially to avoid config conflicts
	updatedCount := 0
	failedCount := 0

	for _, sub := range subscriptions {
		if ctx.Err() != nil {
			return
		}
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
		result, err := au.updateSubscription(ctx, sub)
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

	// Apply all successful refreshes with one restart. A scheduled sweep may
	// update several subscriptions; restarting once per row would repeatedly
	// interrupt traffic while producing the same final running config.
	if updatedCount > 0 {
		if err := au.restartService(); err != nil {
			logger.Error("Subscriptions updated but failed to restart sing-box",
				zap.Int("updated", updatedCount), zap.Error(err))
			failedCount++
		}
	}

	logger.Info("Subscription check completed",
		zap.Int("updated", updatedCount),
		zap.Int("failed", failedCount))
}

// shouldUpdate checks if a subscription needs updating
func (au *Service) shouldUpdate(sub *Subscription) bool {
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
// loop, and restarts sing-box before reporting success. The manual route
// handler should call this instead of touching configManager or the service
// controller directly.
func (au *Service) RefreshByID(id string) (*UpdateResult, error) {
	return au.Refresh(context.Background(), id)
}
func (au *Service) Refresh(ctx context.Context, id string) (*UpdateResult, error) {
	sub, err := au.repository.Get(id)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}
	result, err := au.updateSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}
	if err := au.restartService(); err != nil {
		return nil, fmt.Errorf("subscription updated but failed to restart sing-box: %w", err)
	}
	result.Restarted = true
	return result, nil
}

// restartService applies the config written by a successful refresh. It is a
// separate seam so updater tests do not need a real init system or process.
func (au *Service) restartService() error {
	if au.serviceRestarter == nil {
		return fmt.Errorf("sing-box service restarter is not configured")
	}
	return au.serviceRestarter.Restart()
}

// UpdateSubscription updates a single subscription: fetch → diff → apply.
// Records success/failure stats internally so the cron loop and the manual
// route both produce identical bookkeeping.
func (au *Service) UpdateSubscription(sub *Subscription) (*UpdateResult, error) {
	return au.updateSubscription(context.Background(), sub)
}
func (au *Service) updateSubscription(ctx context.Context, sub *Subscription) (result *UpdateResult, err error) {
	unlock := au.lockSubscription(sub.ID)
	defer unlock()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err := au.checkActive(sub.ID); err != nil {
		return nil, err
	}
	// A scheduled sweep may hold a record read before deletion or an edit.
	sub, err = au.repository.Get(sub.ID)
	if err != nil {
		return nil, err
	}

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
	newNodes, meta, err := au.sublinkManager.Resolve(ctx, lines, feed.FetchOptions{
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
	// Drain stale target snapshots before changing ownership. The history
	// membership fence also rejects already-running single-node tests.
	if au.probeRunner != nil {
		au.probeRunner.BeginDelete(sub.ID)
		defer au.probeRunner.Allow(sub.ID)
	}

	// Compute the diff and rewrite references against the same locked document.
	err = outbounds.NewReconciler(au.configManager, au.nodeRules).Apply(ctx, sub.ID, func(cfg *config.SingBoxConfig) outbounds.Changes {
		deleted, added, updated := au.diffNodes(cfg, sub, realNodes)
		result = &UpdateResult{AddedTags: collectTags(added), UpdatedTags: collectTagsFromMap(updated), DeletedKeys: collectKeys(deleted)}
		return outbounds.Changes{Delete: deleted, Add: added, Update: updated}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply changes: %w", err)
	}
	if err := au.syncNodeHistory(sub.ID); err != nil {
		return nil, fmt.Errorf("nodes updated but failed to prune node history; retry update: %w", err)
	}

	// Step 5: Update subscription's last update time. A failure here doesn't
	// roll back the config — we just log it.
	if uerr := au.repository.UpdateLastUpdate(sub.ID); uerr != nil {
		logger.Warn("Failed to update last update time", zap.String("id", sub.ID), zap.Error(uerr))
	}

	// Step 6: Persist the extracted account metadata (traffic/expiry/reset).
	// Always written (even empty) so stale info clears; a failure here is logged
	// but does not roll back the applied config.
	if ierr := au.repository.UpdateInfo(sub.ID, info); ierr != nil {
		logger.Warn("Failed to update subscription info", zap.String("id", sub.ID), zap.Error(ierr))
	}

	// Step 7: Fill in the provider's own site, if the feed named one and the
	// field is still empty. Only when empty: an operator who corrected the link
	// (providers move domains, and a mirror often reports the old one) must not
	// have that correction undone by the next refresh.
	if site := officialURLToPersist(sub.OfficialURL, meta.SiteURL, info); site != "" {
		if serr := au.repository.UpdateOfficialURL(sub.ID, site); serr != nil {
			logger.Warn("Failed to update subscription official url",
				zap.String("id", sub.ID), zap.Error(serr))
		}
	}

	return result, nil
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
func (au *Service) recordSuccess(subID string) {
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
func (au *Service) recordFailure(subID string, err error) {
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
func (au *Service) GetStats() map[string]*UpdateStats {
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
func (au *Service) GetLastCheckTime() time.Time {
	au.mutex.RLock()
	defer au.mutex.RUnlock()
	return au.lastCheckTime
}

// TriggerCheck manually triggers a subscription check
func (au *Service) TriggerCheck() {
	au.scheduler.Trigger()
}

func (s *Service) lockSubscription(id string) func() {
	value, _ := s.operationLocks.LoadOrStore(id, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

type feedResolver interface {
	Resolve(context.Context, []string, feed.FetchOptions) ([]*node.SubNode, *feed.FetchMeta, error)
}
