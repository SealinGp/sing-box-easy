package autoselect

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

// probe records only confirmed results. Controller failures abort the decision.
func (m *Manager) probe(ctx context.Context, runtime Runtime, tag string) (int, error) {
	delay, err := runtime.Delay(ctx, tag, clashapi.DefaultDelayURL, 5*time.Second)
	if err != nil && !errors.Is(err, clashapi.ErrDelayFailed) {
		return 0, err
	}
	if err != nil {
		delay = 0
	}
	if store, ok := m.history.(interface {
		RecordBatch([]nodehistory.Measurement) error
	}); ok {
		err = store.RecordBatch([]nodehistory.Measurement{{Tag: tag, SubID: nodehistory.SubscriptionID(tag), TestURL: clashapi.DefaultDelayURL, Source: "auto-selection", At: time.Now(), Delay: delay, Success: delay > 0}})
		if err != nil {
			return 0, err
		}
	}
	return delay, nil
}

func (m *Manager) chooseLive(ctx context.Context, runtime Runtime, group clashapi.Proxy, list []clashapi.Proxy, history map[string][]nodehistory.Point, rules settings.NodeQuality, now time.Time, maxAge time.Duration) (string, string, error) {
	current, err := m.probe(ctx, runtime, group.Now)
	if err != nil {
		return "", "", err
	}
	if current == 0 {
		current, err = m.probe(ctx, runtime, group.Now)
		if err != nil {
			return "", "", err
		}
	}
	if current == 0 {
		delete(m.pending, group.Name)
		return m.failover(ctx, runtime, group, list)
	}
	selected, reason := Decide(group, list, history, rules, now, maxAge)
	if !isSwitch(reason) {
		delete(m.pending, group.Name)
		return group.Now, reason, nil
	}
	state := m.state(group.Name)
	if now.Sub(state.LastSwitched) < 5*time.Minute {
		return group.Now, "cooldown", nil
	}
	if m.pending[group.Name] != selected {
		m.pending[group.Name] = selected
		return group.Now, "confirming", nil
	}
	delay, err := m.probe(ctx, runtime, selected)
	if err != nil {
		return "", "", err
	}
	delete(m.pending, group.Name)
	if delay <= 0 || current-delay < 50 || delay*100 > current*80 {
		return group.Now, "no_improvement", nil
	}
	return selected, reason, nil
}

// Probe members concurrently with a bounded fan-out. A fresh successful result
// allows emergency recovery even when a node has insufficient or stale history.
func (m *Manager) failover(ctx context.Context, runtime Runtime, group clashapi.Proxy, list []clashapi.Proxy) (string, string, error) {
	members := map[string]bool{}
	for _, tag := range group.All {
		members[tag] = true
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, 5)
	type result struct {
		tag   string
		delay int
	}
	var results []result
	var failure error
	for _, p := range list {
		if !members[p.Name] || p.Name == group.Now || len(p.All) > 0 {
			continue
		}
		switch strings.ToLower(p.Type) {
		case "", "selector", "urltest", "fallback", "direct", "reject", "block", "dns":
			continue
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return "", "", ctx.Err()
		}
		mu.Lock()
		recovered := len(results) > 0
		mu.Unlock()
		if recovered {
			<-slots
			break
		}
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			defer func() { <-slots }()
			delay, err := m.probe(ctx, runtime, tag)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failure = err
			}
			if err == nil && delay > 0 {
				results = append(results, result{tag, delay})
			}
		}(p.Name)
	}
	wg.Wait()
	if len(results) == 0 {
		if failure != nil {
			return "", "", failure
		}
		return group.Now, "no_healthy_candidate", nil
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].delay != results[j].delay {
			return results[i].delay < results[j].delay
		}
		return results[i].tag < results[j].tag
	})
	return results[0].tag, "failover", nil
}
