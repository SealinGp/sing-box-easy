// Package autoupdate schedules subscription operations and drains them on stop.
package autoupdate

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"sync"
	"time"
)

type Scheduler struct {
	lifecycleMu sync.Mutex
	mu          sync.Mutex
	cron        *cron.Cron
	cancel      context.CancelFunc
	ctx         context.Context
	wg          sync.WaitGroup
	running     bool
	expression  string
	run         func(context.Context)
}

func New(run func(context.Context)) *Scheduler { return &Scheduler{run: run} }
func (s *Scheduler) Start(expression string) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return fmt.Errorf("auto-updater is already running")
	}
	if expression == "" {
		expression = "*/5 * * * *"
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	c := cron.New()
	if _, err := c.AddFunc(expression, s.Trigger); err != nil {
		s.cancel()
		return err
	}
	s.cron = c
	s.running = true
	s.expression = expression
	c.Start()
	ctx := s.ctx
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.Trigger()
		}
	}()
	return nil
}
func (s *Scheduler) Trigger() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	ctx := s.ctx
	s.wg.Add(1)
	s.mu.Unlock()
	go func() { defer s.wg.Done(); s.run(ctx) }()
}
func (s *Scheduler) Stop() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	done := s.cron.Stop()
	s.mu.Unlock()
	<-done.Done()
	s.wg.Wait()
}
func (s *Scheduler) Running() bool      { s.mu.Lock(); defer s.mu.Unlock(); return s.running }
func (s *Scheduler) Expression() string { s.mu.Lock(); defer s.mu.Unlock(); return s.expression }
