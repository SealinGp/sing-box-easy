package trafficflow

import (
	"context"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
)

// Source is the slice of the Clash API this package reads. `*clashapi.Client`
// satisfies it; tests use a fake.
type Source interface {
	Connections(ctx context.Context) (*clashapi.Snapshot, error)
	Rules(ctx context.Context) ([]clashapi.Rule, error)
}

// Options tune one Run.
type Options struct {
	// Interval between samples. sing-box's own WebSocket ticks at one second,
	// which is also the resolution a rate derived from byte counters needs.
	Interval time.Duration
	Filter   Filter
	// RulesRefreshCooldown bounds how often an unmatched rule string may
	// trigger a re-read of `/rules`. Zero means the default.
	RulesRefreshCooldown time.Duration
}

const (
	defaultInterval             = time.Second
	defaultRulesRefreshCooldown = 30 * time.Second
)

// Run samples the source on a ticker and emits a Frame per sample until the
// context ends, the source fails, or the emitter returns an error.
//
// The emitter's error is returned as-is: for the SSE handler it means the
// client is gone, and a stream that keeps polling sing-box for a browser tab
// that closed is the leak the log follower's comments warn about.
//
// The first frame is emitted before the first tick, so a connecting client
// sees the diagram light up immediately rather than one interval later —
// with every connection Fresh and every rate zero, which the client must
// treat as "no baseline yet", not "idle".
func Run(ctx context.Context, source Source, opts Options, emit func(*Frame) error) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	cooldown := opts.RulesRefreshCooldown
	if cooldown <= 0 {
		cooldown = defaultRulesRefreshCooldown
	}

	rules, err := loadRules(ctx, source)
	if err != nil {
		return err
	}
	rulesAt := time.Now()

	differ := NewDiffer()

	sample := func() error {
		snapshot, err := source.Connections(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		live, closed := differ.Step(now, snapshot)
		frame := Aggregate(now, live, rules, opts.Filter)
		frame.Totals.Closed = closed

		// A rule string the index does not know means sing-box's rule list
		// changed since it was read — a reload after a config edit. Re-read
		// it, but not on every tick: a rule that is unmatched because the
		// config really did diverge stays unmatched, and hammering `/rules`
		// for it helps nobody.
		if frame.Unmatched > 0 && now.Sub(rulesAt) >= cooldown {
			if fresh, err := loadRules(ctx, source); err == nil {
				rules = fresh
				rulesAt = now
			}
		}

		return emit(frame)
	}

	if err := sample(); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := sample(); err != nil {
				return err
			}
		}
	}
}

func loadRules(ctx context.Context, source Source) (*RuleIndex, error) {
	rules, err := source.Rules(ctx)
	if err != nil {
		return nil, err
	}
	return BuildRuleIndex(rules), nil
}
