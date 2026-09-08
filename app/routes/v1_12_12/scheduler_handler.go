package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	"github.com/cloudwego/hertz/pkg/app"
)

type schedulerHandler struct{ subscriptions *subscription.Service }

func newSchedulerHandler(s *subscription.Service) *schedulerHandler { return &schedulerHandler{s} }
func (h *schedulerHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.subscriptions.SchedulerStatus())
}
func (h *schedulerHandler) GetJobs(ctx context.Context, c *app.RequestContext) {
	respOK(ctx, c, h.subscriptions.SchedulerJobs())
}
func (h *schedulerHandler) Start(ctx context.Context, c *app.RequestContext) {
	var req struct {
		CronExpression string `json:"cron_expression"`
	}
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	expression, err := h.subscriptions.StartScheduler(req.CronExpression)
	if err != nil {
		respErr(ctx, c, CodeOperationFailed, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"message": "Scheduler started successfully", "cron_expression": expression})
}
func (h *schedulerHandler) Stop(ctx context.Context, c *app.RequestContext) {
	h.subscriptions.Stop()
	respOK(ctx, c, map[string]string{"message": "Scheduler stopped successfully"})
}
func (h *schedulerHandler) Trigger(ctx context.Context, c *app.RequestContext) {
	if err := h.subscriptions.TriggerScheduler(); err != nil {
		respErr(ctx, c, CodeOperationFailed, err.Error())
		return
	}
	respOK(ctx, c, map[string]string{"message": "Subscription check triggered"})
}
