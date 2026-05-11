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
			continue
		}

		logger.Info("Updating subscription", zap.String("id", sub.ID), zap.String("name", sub.Name))

		// Update subscription sequentially to ensure config integrity
		if err := au.UpdateSubscription(sub); err != nil {
			au.recordFailure(sub.ID, err)
			failedCount++
			logger.Error("Failed to update subscription",
				zap.String("id", sub.ID),
				zap.String("name", sub.Name),
				zap.Error(err))
		} else {
			au.recordSuccess(sub.ID)
			updatedCount++
			logger.Info("Successfully updated subscription",
				zap.String("id", sub.ID),
				zap.String("name", sub.Name))
		}
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

// UpdateSubscription updates a single subscription
func (au *AutoUpdater) UpdateSubscription(sub *Subscription) error {
	// Step 1: Fetch new nodes from subscription URL
	lines := []string{sub.URL}
	newNodes, err := au.sublinkManager.ListNodes(lines)
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %w", err)
	}

	if len(newNodes) == 0 {
		return fmt.Errorf("no nodes returned from subscription")
	}

	// Step 2: Get current configuration
	cfg, err := au.configManager.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Step 3: Compare and diff nodes
	toDelete, toAdd, toUpdate := au.diffNodes(cfg, newNodes)

	// Step 4: Apply changes
	if len(toDelete) > 0 || len(toAdd) > 0 || len(toUpdate) > 0 {
		err = au.applyChanges(cfg, toDelete, toAdd, toUpdate, sub.ID)
		if err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
	}

	// Step 5: Update subscription's last update time
	err = au.subscriptionManager.UpdateLastUpdate(sub.ID)
	if err != nil {
		logger.Warn("Failed to update last update time", zap.String("id", sub.ID), zap.Error(err))
	}

	return nil
}

// diffNodes compares current outbounds with new nodes and returns differences
func (au *AutoUpdater) diffNodes(cfg *config.SingBoxConfig, newNodes []*node.SubNode) (toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound) {
	// Initialize named return values (Go does not zero-init maps from named returns)
	toDelete = make(map[string]struct{})
	toUpdate = make(map[string]config.Outbound)

	// Create a map of new nodes by server key (server:port) for quick lookup
	newNodeMap := make(map[string]config.Outbound)
	sub_servers := make(map[string]struct{})
	for _, n := range newNodes {
		// Convert SubNode to Outbound
		outbound := config.Outbound{
			Tag:     n.Tag,
			Type:    n.Type,
			Options: n.Options,
		}
		svr := config.GetOutboundServer(outbound)
		if svr != "" {
			sub_servers[svr] = struct{}{}
		}

		serverKey := config.GetOutboundServerKey(outbound)
		if serverKey != "" {
			// Generate unique tag
			outbound.Tag = config.GenerateUniqueTag(n.Tag, outbound)
			newNodeMap[serverKey] = outbound
		}
	}

	// Track which indices to update instead of delete+add
	updateIndices := make(map[string]config.Outbound)

	// Track which server keys from new nodes have been processed
	processedKeys := make(map[string]bool)

	// Check existing outbounds
	for _, outbound := range cfg.Outbounds {
		svr := config.GetOutboundServer(outbound)
		// Skip outbounds not from this subscription
		if _, ok := sub_servers[svr]; !ok {
			continue
		}

		serverKey := config.GetOutboundServerKey(outbound)
		if serverKey == "" {
			continue
		}

		if newOutbound, exists := newNodeMap[serverKey]; exists {
			// Node with same server key exists in new nodes
			processedKeys[serverKey] = true
			// Check if the outbound has changed
			if !outboundsDeepEqual(outbound, newOutbound) {
				// Mark for update (replace at same index)
				updateIndices[serverKey] = newOutbound
			}
			// If they're equal, keep the existing one (no action needed)
			continue
		}

		toDelete[serverKey] = struct{}{}
	}

	// Add new nodes that weren't in the existing config
	for serverKey, outbound := range newNodeMap {
		if !processedKeys[serverKey] {
			toAdd = append(toAdd, outbound)
		}
	}

	return toDelete, toAdd, updateIndices
}

// applyChanges applies the calculated changes to the configuration
func (au *AutoUpdater) applyChanges(cfg *config.SingBoxConfig, toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound, subID string) error {
	return au.configManager.UpdateConfig(func(c *config.SingBoxConfig) error {
		// Create new outbounds slice
		newOutbounds := make([]config.Outbound, 0, len(c.Outbounds)-len(toDelete)+len(toAdd))

		// First pass: keep non-deleted outbounds and apply updates
		for _, outbound := range c.Outbounds {
			svr_key := config.GetOutboundServerKey(outbound)
			if svr_key == "" {
				newOutbounds = append(newOutbounds, outbound)
				continue
			}

			if _, ok := toDelete[svr_key]; ok {
				continue // Skip deleted
			}

			if updatedOutbound, ok := toUpdate[svr_key]; ok {
				newOutbounds = append(newOutbounds, updatedOutbound)
				continue // Skip updated
			}

			// Keep existing outbound
			newOutbounds = append(newOutbounds, outbound)
		}

		// Add new outbounds
		newOutbounds = append(newOutbounds, toAdd...)

		// Update the configuration
		c.Outbounds = newOutbounds

		logger.Info("Applied subscription changes",
			zap.String("subscription", subID),
			zap.Int("deleted", len(toDelete)),
			zap.Int("added", len(toAdd)),
			zap.Int("updated", len(toUpdate)))

		return nil
	})
}

// getServerFromOutbound extracts the server address from an outbound configuration
func getServerFromOutbound(outbound config.Outbound) string {
	// Handle different outbound types using type assertion on the Options field
	if outbound.Options == nil {
		return ""
	}

	// All standard outbound types have ServerOptions embedded
	// We need to check the specific types for the correct field access
	switch outbound.Type {
	case "shadowsocks":
		// ShadowsocksOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "vmess":
		// VMessOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "trojan":
		// TrojanOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "hysteria":
		// HysteriaOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "hysteria2":
		// Hysteria2OutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "vless":
		// VLESSOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "wireguard":
		// WireGuard has a different structure - server is in peers or direct server field
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			// Check if it's using the new structure with direct server field
			if server, ok := opts["server"].(string); ok {
				return server
			}
			// Check if it's using peers structure
			if peers, ok := opts["peers"].([]interface{}); ok && len(peers) > 0 {
				if peer, ok := peers[0].(map[string]interface{}); ok {
					// Try address field first
					if address, ok := peer["address"].(string); ok {
						return address
					}
					// Try server field for LegacyWireGuardPeer
					if server, ok := peer["server"].(string); ok {
						return server
					}
				}
			}
		}
	case "ssh":
		// SSHOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "shadowtls":
		// ShadowTLSOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	case "tuic":
		// TUICOutboundOptions has ServerOptions embedded
		if opts, ok := outbound.Options.(map[string]interface{}); ok {
			if server, ok := opts["server"].(string); ok {
				return server
			}
		}
	}
	return ""
}

// outboundsDeepEqual performs deep comparison of two outbounds
func outboundsDeepEqual(a, b config.Outbound) bool {
	// Compare type
	if a.Type != b.Type {
		return false
	}

	// For now, do a simple JSON comparison of Options
	// This ensures all fields are compared
	aJSON, err1 := json.Marshal(a.Options)
	bJSON, err2 := json.Marshal(b.Options)

	if err1 != nil || err2 != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}

// outboundsEqual compares two outbounds for equality (simplified)
func outboundsEqual(a, b config.Outbound) bool {
	// This is a simplified comparison
	// You might want to implement a more thorough comparison
	return getServerFromOutbound(a) == getServerFromOutbound(b) &&
		a.Type == b.Type
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
