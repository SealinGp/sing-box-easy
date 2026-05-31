package v1_13_0

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetServiceStatus returns the current status of sing-box service, enriched
// with PID and start time so the overview page can show "last started N ago".
func (h *Handler) GetServiceStatus(ctx context.Context, c *app.RequestContext) {
	info, err := h.serviceController.Info()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	status := "stopped"
	if info.Running {
		status = "running"
	}

	respOK(ctx, c, map[string]any{
		"status":     status,
		"running":    info.Running,
		"pid":        info.PID,
		"started_at": info.StartedAtUnix,
		"uptime":     info.Uptime,
	})
}

// StartService starts the sing-box service
func (h *Handler) StartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Start(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service started successfully"})
}

// StopService stops the sing-box service
func (h *Handler) StopService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Stop(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service stopped successfully"})
}

// RestartService restarts the sing-box service
func (h *Handler) RestartService(ctx context.Context, c *app.RequestContext) {
	if err := h.serviceController.Restart(); err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "service restarted successfully"})
}

// GetServiceLogs returns the most recent sing-box log lines for the live viewer.
//
// Query params:
//   - lines: max lines to return (clamped server-side; default 300, max 1000)
//   - cursor: opaque journald cursor; when set, only entries newer than it are
//     returned (incremental polling). Feed back the returned cursor each poll.
func (h *Handler) GetServiceLogs(ctx context.Context, c *app.RequestContext) {
	lines, _ := strconv.Atoi(c.Query("lines"))
	cursor := c.Query("cursor")

	chunk, err := h.serviceController.TailLogs(lines, cursor)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"lines":  chunk.Lines,
		"cursor": chunk.Cursor,
		"source": chunk.Source,
	})
}
