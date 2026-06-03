package configversion

import (
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// DefaultMaxAge is the retention window for historical config versions. Anything
// older is removed by the cleanup cron.
const DefaultMaxAge = 60 * 24 * time.Hour // 60 days

// DefaultCleanupCron runs the retention sweep once a day at 03:30 (a quiet hour).
const DefaultCleanupCron = "30 3 * * *"

// Cleaner periodically deletes config versions older than maxAge.
type Cleaner struct {
	store  *StoreXORM
	maxAge time.Duration
	cron   *cron.Cron
	done   chan struct{} // closed by Stop to cancel the delayed initial sweep
	mu     sync.Mutex
}

// NewCleaner builds a retention cleaner. A non-positive maxAge falls back to the
// default so a misconfiguration never disables retention silently.
func NewCleaner(store *StoreXORM, maxAge time.Duration) *Cleaner {
	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	return &Cleaner{store: store, maxAge: maxAge}
}

// Start schedules the cron and also runs one sweep shortly after boot (so a
// long-lived process that just started still trims a stale backlog promptly).
func (c *Cleaner) Start(spec string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cron != nil {
		return nil // already running
	}
	if spec == "" {
		spec = DefaultCleanupCron
	}

	cr := cron.New()
	if _, err := cr.AddFunc(spec, c.run); err != nil {
		return err
	}
	cr.Start()
	c.cron = cr
	c.done = make(chan struct{})
	done := c.done
	logger.Info("Config version cleaner started",
		zap.String("cron", spec), zap.Duration("max_age", c.maxAge))

	// Initial sweep, delayed so startup isn't competing for the DB. Cancellable
	// via Stop() so a stopped cleaner leaves no lingering goroutine.
	go func() {
		select {
		case <-time.After(60 * time.Second):
			c.run()
		case <-done:
		}
	}()
	return nil
}

// Stop halts the cron.
func (c *Cleaner) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cron != nil {
		c.cron.Stop()
		c.cron = nil
		if c.done != nil {
			close(c.done)
			c.done = nil
		}
		logger.Info("Config version cleaner stopped")
	}
}

// run performs one retention sweep. Best-effort: errors are logged, not fatal.
func (c *Cleaner) run() {
	n, err := c.store.DeleteOlderThan(c.maxAge)
	if err != nil {
		logger.Warn("config version cleanup failed", zap.Error(err))
		return
	}
	if n > 0 {
		logger.Info("Config version cleanup removed old versions",
			zap.Int64("deleted", n), zap.Duration("older_than", c.maxAge))
	}
}
