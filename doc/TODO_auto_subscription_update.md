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
Create a new scheduler package at `app/pkg/scheduler/subscription_scheduler.go`:

```go
type SubscriptionScheduler struct {
    subscriptionManager *subscription.Manager
    sublinkManager     *sublink.Manager
    configManager      *config.Manager
    ticker             *time.Ticker
    stopChan           chan struct{}
}
```

Features needed:
- Run periodic checks (e.g., every 5 minutes)
- Check all enabled subscriptions with `AutoUpdate = true`
- Calculate if update is needed based on `LastUpdate` + `UpdateInterval`
- Trigger update for expired subscriptions

### 2. Update Logic
When a subscription needs updating:
1. Fetch new nodes from subscription URL using `sublink.Manager`
2. Parse nodes and validate them
3. Update outbound configuration
4. Save new configuration
5. Update `LastUpdate` timestamp in database
6. Log success/failure

### 3. Duration Parsing
Implement duration parsing to convert strings like "24h", "7d", "30min" to Go `time.Duration`:
```go
func ParseUpdateInterval(interval string) (time.Duration, error) {
    // Support formats: 24h, 7d, 30min, 2w, 1mo
    // Convert to hours then to time.Duration
}
```

### 4. Integration Points

#### a. Main Application (`app/svr.go` or `main.go`)
- Initialize and start the scheduler on app startup
- Ensure graceful shutdown on app termination

#### b. API Endpoints
Add new endpoints for scheduler control:
- `GET /api/1.12.12/scheduler/status` - Get scheduler status
- `POST /api/1.12.12/scheduler/start` - Start scheduler
- `POST /api/1.12.12/scheduler/stop` - Stop scheduler
- `POST /api/1.12.12/scheduler/trigger` - Manually trigger check

#### c. Configuration
Add scheduler configuration to `app.yml`:
```yaml
scheduler:
  enabled: true
  check_interval: 5m  # How often to check subscriptions
  retry_attempts: 3
  retry_delay: 1m
```

### 5. Error Handling & Logging

- Implement retry mechanism for failed updates
- Log all update attempts with results
- Track failed update count per subscription
- Optional: Send notifications on repeated failures

### 6. Database Updates

Consider adding fields to subscription table:
- `last_error`: String to store last error message
- `failed_count`: Number of consecutive failed updates
- `next_update`: Calculated next update time for easier querying

### 7. Outbound Update Strategy

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