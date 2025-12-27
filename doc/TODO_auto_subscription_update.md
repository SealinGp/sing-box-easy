# TODO: Auto Subscription Update Feature

## Problem Statement
Currently, there is no automatic job/scheduler to check and refresh subscriptions when they expire based on their `update_interval`. The subscription managers (`subscription.go` and `subscription_xorm.go`) only provide basic CRUD operations without any background update mechanism.

## Current State Analysis

### What exists:
- `Subscription` struct has fields for auto-update functionality:
  - `Enabled`: Whether subscription is enabled
  - `AutoUpdate`: Whether auto-update is enabled
  - `UpdateInterval`: String duration (e.g., "24h", "7d")
  - `LastUpdate`: Timestamp of last update
- Basic CRUD operations in subscription managers
- Manual update API endpoint (`/api/1.12.12/subscriptions/{id}/update`)
- Frontend validation for duration formats using dayjs

### What's missing:
- Background job/scheduler to periodically check subscriptions
- Automatic update logic when interval expires
- Outbound nodes refresh after subscription update
- Error handling and retry mechanism for failed updates
- Update status tracking and notifications

## Implementation Requirements

### 1. Background Scheduler Component
Create auto-update feature in `app/pkg/subscription/auto_updater.go` using cron library:

```go
import (
    "github.com/robfig/cron/v3"
    "github.com/SealinGp/sing-box-easy/app/pkg/sublink"
    "github.com/SealinGp/sing-box-easy/app/pkg/config"
)

type AutoUpdater struct {
    subscriptionManager SubscriptionManager
    sublinkManager     *sublink.SubLink
    configManager      *config.Manager
    cron               *cron.Cron
}
```

Features needed:
- Use `github.com/robfig/cron` for job scheduling
- Run periodic checks (e.g., every 5 minutes: `*/5 * * * *`)
- Check all enabled subscriptions with `AutoUpdate = true`
- Calculate if update is needed based on `LastUpdate` + `UpdateInterval`
- Trigger update for expired subscriptions

### 2. Update Logic (Detailed Flow)
When a subscription needs updating:

#### Step 1: List and Check Subscriptions
```go
func (au *AutoUpdater) CheckSubscriptions() {
    // List all enabled subscriptions
    subs, _ := au.subscriptionManager.List()
    for _, sub := range subs {
        if !sub.Enabled || !sub.AutoUpdate {
            continue
        }
        // Check if time to update based on LastUpdate + UpdateInterval
        if time.Since(sub.LastUpdate) >= parseDuration(sub.UpdateInterval) {
            au.UpdateSubscription(sub)
        }
    }
}
```

#### Step 2: Fetch New Nodes
```go
func (au *AutoUpdater) UpdateSubscription(sub Subscription) {
    // Call ListNodes from sublink package
    lines := []string{sub.URL}
    newNodes, err := au.sublinkManager.ListNodes(lines)
    if err != nil {
        // Handle error
        return
    }
    // Continue to Step 3...
}
```

#### Step 3: Compare and Diff Nodes
```go
func (au *AutoUpdater) UpdateSubscription(sub Subscription) {
    // ... (Step 2 above)

    // Get current config
    cfg, _ := au.configManager.GetConfig()

    // Create maps for comparison using server field as key
    // A: Current outbounds with matching servers
    currentOutbounds := make(map[string]Outbound)
    for _, outbound := range cfg.Outbounds {
        if server := getServerField(outbound); server != "" {
            currentOutbounds[server] = outbound
        }
    }

    // B: New nodes from subscription
    newOutbounds := make(map[string]Outbound)
    for _, node := range newNodes {
        if server := getServerField(node.Outbound); server != "" {
            newOutbounds[server] = node.Outbound
        }
    }

    // Calculate differences
    // Nodes to delete: diff(A, B) - in current but not in new
    toDelete := []string{}
    for server := range currentOutbounds {
        if _, exists := newOutbounds[server]; !exists {
            toDelete = append(toDelete, server)
        }
    }

    // Nodes to add: diff(B, A) - in new but not in current
    toAdd := []Outbound{}
    for server, outbound := range newOutbounds {
        if _, exists := currentOutbounds[server]; !exists {
            toAdd = append(toAdd, outbound)
        }
    }

    // Apply changes to config
    au.ApplyChanges(cfg, toDelete, toAdd)
}
```

#### Step 4: Apply Changes to Config
```go
func (au *AutoUpdater) ApplyChanges(cfg *Config, toDelete []string, toAdd []Outbound) {
    // Remove old nodes
    newOutbounds := []Outbound{}
    for _, outbound := range cfg.Outbounds {
        server := getServerField(outbound)
        if !contains(toDelete, server) {
            newOutbounds = append(newOutbounds, outbound)
        }
    }

    // Add new nodes
    newOutbounds = append(newOutbounds, toAdd...)

    // Update config
    cfg.Outbounds = newOutbounds
    au.configManager.UpdateConfig(func(c *Config) error {
        c.Outbounds = newOutbounds
        return nil
    })
}
```

#### Note on Go Struct Comparison
Go can compare structs as map keys if all fields are comparable. For complex structs with slices/maps, use the server field as a unique identifier:
```go
// Helper function to extract server field from different outbound types
func getServerField(outbound Outbound) string {
    // Type switch or reflection to get server field
    // Each outbound type (VMess, Trojan, etc.) has a server field
    switch o := outbound.(type) {
    case *VMessOutbound:
        return o.Server
    case *TrojanOutbound:
        return o.Server
    // ... other types
    }
    return ""
}
```

### 3. Duration Parsing
Implement duration parsing to convert strings like "24h", "7d", "30min" to Go `time.Duration`:
```go
func ParseUpdateInterval(interval string) (time.Duration, error) {
    // Support formats: 24h, 7d, 30min, 2w, 1mo
    // Convert to hours then to time.Duration
}
```

### 4. Cron Job Setup

#### Using robfig/cron Library
```go
import "github.com/robfig/cron/v3"

func (au *AutoUpdater) Start() {
    au.cron = cron.New()

    // Schedule job to run every 5 minutes
    au.cron.AddFunc("*/5 * * * *", au.CheckSubscriptions)

    // Or use more readable format with cron.WithSeconds()
    c := cron.New(cron.WithSeconds())
    c.AddFunc("0 */5 * * * *", au.CheckSubscriptions)

    au.cron.Start()
}

func (au *AutoUpdater) Stop() {
    if au.cron != nil {
        au.cron.Stop()
    }
}
```

#### Cron Expression Examples:
- `*/5 * * * *` - Every 5 minutes
- `0 * * * *` - Every hour at minute 0
- `0 0 * * *` - Daily at midnight
- `0 0 * * 0` - Weekly on Sunday at midnight
- `@hourly` - Every hour (predefined)
- `@daily` - Every day at midnight (predefined)

### 5. Integration Points

#### a. Main Application (`app/svr.go` or `main.go`)
```go
func main() {
    // ... existing initialization

    // Initialize auto-updater
    autoUpdater := subscription.NewAutoUpdater(
        subscriptionManager,
        sublinkManager,
        configManager,
    )

    // Start cron job
    autoUpdater.Start()
    defer autoUpdater.Stop()

    // ... rest of application
}
```

#### b. API Endpoints
Add new endpoints for scheduler control:
- `GET /api/1.12.12/scheduler/status` - Get scheduler status
- `POST /api/1.12.12/scheduler/start` - Start scheduler
- `POST /api/1.12.12/scheduler/stop` - Stop scheduler
- `POST /api/1.12.12/scheduler/trigger` - Manually trigger check
- `GET /api/1.12.12/scheduler/jobs` - List scheduled jobs

#### c. Configuration
Add scheduler configuration to `app.yml`:
```yaml
scheduler:
  enabled: true
  cron_expression: "*/5 * * * *"  # Cron expression for check interval
  retry_attempts: 3
  retry_delay: 1m
  concurrent_updates: 3  # Max concurrent subscription updates
```

### 6. Error Handling & Logging

- Implement retry mechanism for failed updates
- Log all update attempts with results
- Track failed update count per subscription
- Optional: Send notifications on repeated failures

### 7. Database Updates

Consider adding fields to subscription table:
- `last_error`: String to store last error message
- `failed_count`: Number of consecutive failed updates
- `next_update`: Calculated next update time for easier querying

### 8. Outbound Update Strategy

When subscription nodes are updated:
1. **Replace Strategy**: Remove old nodes, add new ones
2. **Merge Strategy**: Keep manually added nodes, update subscription nodes
3. **Tag-based**: Use tags to identify subscription nodes for updates

Suggested approach: Use tags like `sub_{subscription_id}_{node_index}` to track which outbounds belong to which subscription.

## Implementation Steps

### Phase 1: Core Scheduler
1. Create scheduler package and basic structure
2. Implement duration parsing utilities
3. Add periodic check logic
4. Integrate with existing managers

### Phase 2: Update Logic
1. Implement subscription update workflow
2. Add node parsing and validation
3. Update outbound configuration
4. Handle errors and retries

### Phase 3: API & Control
1. Add scheduler control endpoints
2. Add status and monitoring endpoints
3. Implement manual trigger functionality

### Phase 4: Enhancement
1. Add notification system
2. Implement advanced scheduling (e.g., quiet hours)
3. Add update history tracking
4. Performance optimization for large subscription lists

## Testing Requirements

1. Unit tests for duration parsing
2. Integration tests for scheduler
3. Mock subscription URLs for testing
4. Test error scenarios and retries
5. Load testing with multiple subscriptions

## Security Considerations

1. Validate subscription URLs before fetching
2. Implement rate limiting for external requests
3. Sanitize and validate parsed nodes
4. Prevent infinite retry loops
5. Log security events (invalid URLs, malformed data)

## Future Enhancements

1. WebSocket notifications for real-time updates
2. Subscription health monitoring dashboard
3. Smart scheduling based on usage patterns
4. Bulk subscription management
5. Import/export subscription groups
6. Subscription node performance tracking

## References

- Current subscription implementation: `/app/pkg/subscription/`
- SubLink parser: `/app/pkg/sublink/`
- Config manager: `/app/pkg/config/`
- Frontend duration validation: `/frontend/src/plugins/dayjs.ts`