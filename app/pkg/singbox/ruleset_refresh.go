package singbox

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
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
	results, err := c.RefreshRuleSets(ctx, []string{tag})
	if err != nil {
		return RuleSetRefreshResult{}, err
	}
	return results[0], nil
}

// RefreshRuleSets refreshes one batch with a single stop/start cycle. A failed
// download restores the entire batch; no partial application is reported.
func (c *Controller) RefreshRuleSets(ctx context.Context, tags []string) ([]RuleSetRefreshResult, error) {
	var result []RuleSetRefreshResult
	unique := make([]string, 0, len(tags))
	seen := make(map[string]bool)
	if len(tags) == 0 || len(tags) > 100 {
		return nil, fault.New(fault.Invalid, "select between 1 and 100 rule sets")
	}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return nil, fault.New(fault.Invalid, "rule-set tag is required")
		}
		if !seen[tag] {
			seen[tag] = true
			unique = append(unique, tag)
		}
	}
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
		plans := make([]ruleSetRefreshPlan, 0, len(unique))
		for _, tag := range unique {
			plan, err := planRuleSetRefresh(doc.Raw, tag, pid, c.backend.Kind() == BackendProcess)
			if err != nil {
				return fault.New(fault.Invalid, err.Error())
			}
			plans = append(plans, plan)
		}
		// Open one cache snapshot for the entire batch before interrupting traffic.
		id := plans[0].id
		if id != "" {
			id = "\x00" + id
		}
		cache, err := ruleset.OpenCacheFile(plans[0].path, id)
		if err != nil {
			return fault.New(fault.Unavailable, "could not read rule-set cache: "+err.Error())
		}
		for _, p := range plans {
			entry := cache.LoadRuleSet(p.tag)
			if entry == nil || len(entry.Content) == 0 {
				cache.Close()
				return fault.New(fault.Unavailable, "rule set has no cached content to refresh: "+p.tag)
			}
		}
		cache.Close()
		result, err = c.refreshRuleSets(ctx, plans)
		return err
	})
	return result, err
}

func (c *Controller) refreshRuleSets(ctx context.Context, plans []ruleSetRefreshPlan) ([]RuleSetRefreshResult, error) {
	p := plans[0]
	tags := make([]string, 0, len(plans))
	for _, item := range plans {
		tags = append(tags, item.tag)
	}
	if err := c.backend.Stop(); err != nil {
		return nil, err
	}
	previous, err := rulesetcache.EvictMany(p.path, p.id, tags)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("could not prepare rule-set refresh: %w", err), c.backend.Start())
	}
	started := time.Now().Unix()
	if err = c.backend.Start(); err == nil {
		var results []RuleSetRefreshResult
		results, err = c.waitRuleSetRefresh(ctx, p, tags, previous, started)
		if err == nil {
			return results, nil
		}
	}
	// All selected entries are restored together, including ones downloaded before
	// a later member failed. Never mutate a cache held by the running core.
	if stopErr := c.backend.Stop(); stopErr != nil {
		return nil, errors.Join(err, fmt.Errorf("could not stop core for recovery: %w", stopErr))
	}
	if restoreErr := rulesetcache.RestoreMany(p.path, p.id, previous); restoreErr != nil {
		return nil, errors.Join(err, fmt.Errorf("could not restore previous rule sets: %w", restoreErr), c.backend.Start())
	}
	if startErr := c.backend.Start(); startErr != nil {
		return nil, errors.Join(err, fmt.Errorf("could not restart with previous rule sets: %w", startErr))
	}
	return nil, fmt.Errorf("rule-set refresh failed; previous content restored for the entire batch: %w", err)
}

func (c *Controller) waitRuleSetRefresh(ctx context.Context, p ruleSetRefreshPlan, tags []string, previous map[string][]byte, started int64) ([]RuleSetRefreshResult, error) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	id := p.id
	if id != "" {
		id = "\x00" + id
	}
	for {
		running, _, err := c.backend.ActiveAndPID()
		if err != nil {
			return nil, err
		}
		if !running {
			return nil, fmt.Errorf("sing-box did not remain running after refresh")
		}
		cache, openErr := ruleset.OpenCacheFile(p.path, id)
		if openErr == nil {
			results := make([]RuleSetRefreshResult, 0, len(tags))
			for _, tag := range tags {
				entry := cache.LoadRuleSet(tag)
				if entry == nil || len(entry.Content) == 0 || entry.LastUpdated.Unix() < started {
					break
				}
				results = append(results, RuleSetRefreshResult{Tag: tag, UpdatedAt: entry.LastUpdated.Unix(), Changed: rulesetcache.ContentHash(previous[tag]) != sha256.Sum256(entry.Content)})
			}
			cache.Close()
			if len(results) == len(tags) {
				return results, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for refreshed rule sets: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}
