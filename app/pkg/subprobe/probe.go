package subprobe

// The measurement half of the prober: what a single sweep of one
// subscription's nodes actually does. The scheduling half — when sweeps
// happen, and the state they leave behind — is runner.go.
//
// Split because the two answer different questions and change for different
// reasons: this file changes when the definition of "reachable" does, runner.go
// when the cadence or the lifecycle does.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"go.uber.org/zap"
)

// runTarget probes one subscription and stores the result.
func (r *Runner) runTarget(ctx context.Context, prober Prober, tags []string, target Target, settings Settings) error {
	owned := tagsFor(tags, target.SubID)
	if len(owned) == 0 {
		// Nothing imported for this subscription yet. Recording a sample would
		// draw a feed whose nodes simply have not been fetched as a provider
		// that is entirely down.
		return ErrNoOwnedNodes
	}

	results, fatal := r.probeAll(ctx, prober, owned, target.URL, settings)
	if fatal != nil {
		return fatal
	}

	sample, ok := Aggregate(results)

	// The snapshot is recorded even when there is no sample to store: the
	// all-untestable case is exactly when someone opens the node list to find
	// out what happened, and an empty list there would say nothing.
	now := time.Now()
	r.mu.Lock()
	r.snapshots[target.SubID] = &Snapshot{At: now, Sample: sample, Results: results}
	r.mu.Unlock()

	if !ok {
		// Every node was untestable, so there is nothing to record. Reported
		// rather than stored: a zeroed sample would be drawn as a 0% outage,
		// which is the opposite of what happened — nothing was measured.
		return fmt.Errorf("%w (%d nodes)", ErrNoNodesTestable, sample.Skipped)
	}

	if err := r.store.Insert(target.SubID, now, sample); err != nil {
		return fmt.Errorf("failed to store probe sample: %w", err)
	}

	// Trimmed on the write path, not on a separate cron: the bound exists to
	// protect a small disk, and a sweep that filled it between nightly sweeps
	// would have already done the damage.
	if _, err := r.store.Trim(target.SubID, settings.MaxAge, settings.MaxPoints); err != nil {
		logger.Warn("Failed to trim probe samples",
			zap.String("subscription", target.SubID), zap.Error(err))
	}

	logger.Info("Probed subscription",
		zap.String("subscription", target.SubID),
		zap.Int("total", sample.Total),
		zap.Int("reachable", sample.Reachable),
		zap.Int("skipped", sample.Skipped),
		zap.Int("avg_ms", sample.AvgMs))
	return nil
}

// probeAll runs the URL test over one subscription's nodes with a bounded pool.
//
// The second return value is a condition that invalidates the whole run rather
// than describing any node — today only an authentication failure, which fails
// every node identically and is the panel's fault, not the provider's.
func (r *Runner) probeAll(ctx context.Context, prober Prober, tags []string, testURL string, settings Settings) ([]NodeResult, error) {
	results := make([]NodeResult, len(tags))
	var fatalOnce sync.Once
	var fatal error

	workers := settings.Concurrency
	if workers > len(tags) {
		workers = len(tags)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				result, isFatal := probeOne(ctx, prober, tags[i], testURL, settings.Timeout)
				results[i] = result
				if isFatal {
					// First one wins; the rest of the pool drains rather than
					// being torn down mid-request.
					fatalOnce.Do(func() { fatal = clashapi.ErrUnauthorized })
				}
			}
		}()
	}

	for i := range tags {
		select {
		case <-ctx.Done():
			// Abandon the rest; already-dispatched jobs finish on their own
			// short timeouts. The partial results still aggregate correctly —
			// unfilled entries carry the zero NodeResult, which is why the
			// cancellation is recorded on them below.
			close(jobs)
			wg.Wait()
			markUnprobed(results, tags, ctx.Err())
			return results, fatal
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	return results, fatal
}

// probeOne classifies a single node's outcome. The bool reports a failure that
// says nothing about this node (see probeAll).
func probeOne(ctx context.Context, prober Prober, tag, testURL string, timeout time.Duration) (NodeResult, bool) {
	// Give the request slightly more room than the URL test itself, so a node
	// that legitimately takes the whole timeout is reported by sing-box as a
	// timeout rather than cut off here as a transport error.
	callCtx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()

	delay, err := prober.Delay(callCtx, tag, testURL, timeout)
	switch {
	case err == nil:
		return NodeResult{Tag: tag, DelayMs: delay}, false
	case errors.Is(err, clashapi.ErrUnauthorized):
		return NodeResult{Tag: tag, Skipped: true, Error: err.Error()}, true
	case errors.Is(err, clashapi.ErrProxyNotFound):
		// In the config, absent from the running sing-box. Untestable, not
		// down — see NodeResult.
		return NodeResult{Tag: tag, Skipped: true, Error: err.Error()}, false
	default:
		return NodeResult{Tag: tag, Error: err.Error()}, false
	}
}

// markUnprobed labels entries a cancelled sweep never reached, so they are
// reported as untestable rather than silently counted as failures.
func markUnprobed(results []NodeResult, tags []string, err error) {
	for i := range results {
		if results[i].Tag == "" {
			results[i] = NodeResult{Tag: tags[i], Skipped: true, Error: err.Error()}
		}
	}
}

// tagsFor selects the outbound tags owned by one subscription.
func tagsFor(tags []string, subID string) []string {
	owned := make([]string, 0, 16)
	for _, tag := range tags {
		// The ownership rule lives in the package that MINTS the tags, so a
		// change to the suffix format cannot leave the prober measuring a
		// stale set of nodes.
		if subscription.TagBelongsToSubscription(tag, subID) {
			owned = append(owned, tag)
		}
	}
	return owned
}
