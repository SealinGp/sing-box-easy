package subscription

import (
	"fmt"
	"time"
)

func (s *Service) SchedulerStatus() any {
	var last *time.Time
	if t := s.GetLastCheckTime(); !t.IsZero() {
		last = &t
	}
	return struct {
		Running bool                    `json:"running"`
		Last    *time.Time              `json:"last_check_time,omitempty"`
		Stats   map[string]*UpdateStats `json:"stats"`
	}{s.IsRunning(), last, s.GetStats()}
}
func (s *Service) StartScheduler(expression string) (string, error) {
	if expression == "" {
		expression = "*/5 * * * *"
	}
	return expression, s.Start(expression)
}
func (s *Service) TriggerScheduler() error {
	if !s.IsRunning() {
		return fmt.Errorf("Scheduler is not running")
	}
	s.TriggerCheck()
	return nil
}
func (s *Service) SchedulerJobs() any {
	expression := s.scheduler.Expression()
	if expression == "" {
		expression = "*/5 * * * *"
	}
	return map[string]any{"jobs": []map[string]string{{"id": "subscription_updater", "description": "Auto-update subscriptions", "schedule": expression}}}
}
