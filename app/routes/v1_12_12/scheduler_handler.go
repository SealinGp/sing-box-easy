package v1_13_0

import (
	"context"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

// schedulerHandler handles auto-updater scheduler endpoints
type schedulerHandler struct {
	autoUpdater *subscription.AutoUpdater
}

// newSchedulerHandler creates a new scheduler handler
func newSchedulerHandler(autoUpdater *subscription.AutoUpdater) *schedulerHandler {
	return &schedulerHandler{
		autoUpdater: autoUpdater,
	}
}

// GetStatus returns the current status of the scheduler
func (sh *schedulerHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	status := struct {
		Running       bool                                `json:"running"`
		LastCheckTime *time.Time                          `json:"last_check_time,omitempty"`
		Stats         map[string]*subscription.UpdateStats `json:"stats"`
	}{
		Running: sh.autoUpdater.IsRunning(),
		Stats:   sh.autoUpdater.GetStats(),
	}

	if lastCheck := sh.autoUpdater.GetLastCheckTime(); !lastCheck.IsZero() {
		status.LastCheckTime = &lastCheck
	}

	respOK(ctx, c, status)
}

// Start starts the scheduler
func (sh *schedulerHandler) Start(ctx context.Context, c *app.RequestContext) {
	type request struct {
		CronExpression string `json:"cron_expression"`
	}

	var req request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Default cron expression if not provided
	if req.CronExpression == "" {
		req.CronExpression = "*/5 * * * *" // Every 5 minutes
	}

	if err := sh.autoUpdater.Start(req.CronExpression); err != nil {
		respErr(ctx, c, CodeOperationFailed, "Failed to start scheduler: "+err.Error())
		return
	}

	respOK(ctx, c, struct {
		Message string `json:"message"`
		Cron    string `json:"cron_expression"`
	}{
		Message: "Scheduler started successfully",
		Cron:    req.CronExpression,
	})
}

// Stop stops the scheduler
func (sh *schedulerHandler) Stop(ctx context.Context, c *app.RequestContext) {
	sh.autoUpdater.Stop()

	respOK(ctx, c, struct {
		Message string `json:"message"`
	}{
		Message: "Scheduler stopped successfully",
	})
}

// Trigger manually triggers a subscription check
func (sh *schedulerHandler) Trigger(ctx context.Context, c *app.RequestContext) {
	if !sh.autoUpdater.IsRunning() {
		respErr(ctx, c, CodeOperationFailed, "Scheduler is not running")
		return
	}

	sh.autoUpdater.TriggerCheck()

	respOK(ctx, c, struct {
		Message string `json:"message"`
	}{
		Message: "Subscription check triggered",
	})
}

// GetJobs returns information about scheduled jobs
func (sh *schedulerHandler) GetJobs(ctx context.Context, c *app.RequestContext) {
	jobs := struct {
		Jobs []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Schedule    string `json:"schedule"`
		} `json:"jobs"`
		NextRun *time.Time `json:"next_run,omitempty"`
	}{
		Jobs: []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Schedule    string `json:"schedule"`
		}{
			{
				ID:          "subscription_updater",
				Description: "Auto-update subscriptions",
				Schedule:    "*/5 * * * *", // This should be dynamic based on actual config
			},
		},
	}

	respOK(ctx, c, jobs)
}