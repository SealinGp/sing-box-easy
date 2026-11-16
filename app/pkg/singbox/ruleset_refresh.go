package singbox

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/rulesetcache"
)

type RuleSetRefreshResult struct {
	Tag       string `json:"tag"`
	UpdatedAt int64  `json:"updated_at_unix"`
	Changed   bool   `json:"changed"`
}

// RefreshRuleSet makes the core download through its own configured transport.
// Both locks are held through recovery: edits and lifecycle actions cannot race
// cache invalidation. No configuration or historical version is written.
func (c *Controller) RefreshRuleSet(ctx context.Context, tag string) (RuleSetRefreshResult, error) {
	var result RuleSetRefreshResult
	if !c.lifecycleMu.TryLock() {
		return result, fault.New(fault.Conflict, "another service operation is in progress")
	}
	defer c.lifecycleMu.Unlock()
	// A disconnected browser must not abandon a stopped core midway through recovery.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
	defer cancel()
	if running, err := c.Status(); err != nil {
		return result, err
	} else if !running {
		return result, fault.New(fault.Conflict, "start sing-box before refreshing rule sets")
	}
	if err := c.validateConfig(); err != nil {
		return result, err
	}
	err := c.configManager.WithDocumentLock(ctx, func(doc config.ConfigDocument, _ func(config.ConfigDocument) error) error {
		running, pid, err := c.backend.ActiveAndPID()
		if err != nil {
			return err
		}
		if !running {
			return fault.New(fault.Conflict, "start sing-box before refreshing rule sets")
		}
		plan, err := planRuleSetRefresh(doc.Raw, tag, pid, c.backend.Kind() == BackendProcess)
		if err != nil {
			return fault.New(fault.Invalid, err.Error())
		}
		// Detect absent/misconfigured caches before interrupting traffic.
		id := plan.id
		if id != "" {
			id = "\x00" + id
		}
		cache, err := ruleset.OpenCacheFile(plan.path, id)
		if err != nil {
			return fault.New(fault.Unavailable, "could not read rule-set cache: "+err.Error())
		}
		entry := cache.LoadRuleSet(tag)
		cache.Close()
		if entry == nil || len(entry.Content) == 0 {
			return fault.New(fault.Unavailable, "rule set has no cached content to refresh")
		}
		result, err = c.refreshRuleSet(ctx, plan)
		return err
	})
	return result, err
}

func (c *Controller) refreshRuleSet(ctx context.Context, p ruleSetRefreshPlan) (RuleSetRefreshResult, error) {
	result := RuleSetRefreshResult{Tag: p.tag}
	if err := c.backend.Stop(); err != nil {
		return result, err
	}
	previous, err := rulesetcache.Evict(p.path, p.id, p.tag)
	if err != nil {
		return result, errors.Join(fmt.Errorf("could not prepare rule-set refresh: %w", err), c.backend.Start())
	}
	// Capture the old content hash independently of the cache wire format.
	// The entry was evicted already, so use its length-prefixed content field.
	before := rulesetcache.ContentHash(previous)
	started := time.Now().Unix()
	if err = c.backend.Start(); err == nil {
		var updated time.Time
		var after [32]byte
		updated, after, err = c.waitRuleSetRefresh(ctx, p, started)
		if err == nil {
			result.UpdatedAt = updated.Unix()
			result.Changed = before != after
			return result, nil
		}
	}
	// Never write a cache held by a running core. Preserve the old opaque bytes.
	if stopErr := c.backend.Stop(); stopErr != nil {
		return result, errors.Join(err, fmt.Errorf("could not stop core for recovery: %w", stopErr))
	}
	if restoreErr := rulesetcache.Restore(p.path, p.id, p.tag, previous); restoreErr != nil {
		return result, errors.Join(err, fmt.Errorf("could not restore previous rule set: %w", restoreErr), c.backend.Start())
	}
	if startErr := c.backend.Start(); startErr != nil {
		return result, errors.Join(err, fmt.Errorf("could not restart with previous rule set: %w", startErr))
	}
	return result, fmt.Errorf("rule-set refresh failed; previous content restored: %w", err)
}

func (c *Controller) waitRuleSetRefresh(ctx context.Context, p ruleSetRefreshPlan, started int64) (time.Time, [32]byte, error) {
	// Initial downloads finish before the core starts serving. Require both a
	// newly saved cache entry and a live core, never just a successful restart.
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	id := p.id
	if id != "" {
		id = "\x00" + id
	}
	for {
		running, _, err := c.backend.ActiveAndPID()
		if err != nil {
			return time.Time{}, [32]byte{}, err
		}
		if !running {
			return time.Time{}, [32]byte{}, fmt.Errorf("sing-box did not remain running after refresh")
		}
		cache, openErr := ruleset.OpenCacheFile(p.path, id)
		if openErr == nil {
			entry := cache.LoadRuleSet(p.tag)
			cache.Close()
			if entry != nil && len(entry.Content) > 0 && entry.LastUpdated.Unix() >= started {
				return entry.LastUpdated, sha256.Sum256(entry.Content), nil
			}
		}
		select {
		case <-ctx.Done():
			return time.Time{}, [32]byte{}, fmt.Errorf("timed out waiting for refreshed rule set: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
