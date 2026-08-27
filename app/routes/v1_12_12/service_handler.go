package v1_13_0

import (
	"context"
	"strconv"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/service"
	"github.com/cloudwego/hertz/pkg/app"
)

// keepAliveInterval bounds how long a log stream may stay silent before it
// sends a comment frame. sing-box at `level: info` on an idle link logs nothing
// for minutes, and an intermediate proxy reads that as a dead connection.
const keepAliveInterval = 20 * time.Second

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
		"version":    info.Version,
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

// StreamServiceLogs pushes sing-box log lines to the client as they are
// written, over SSE.
//
// WHY THIS EXISTS ALONGSIDE GetServiceLogs
// ────────────────────────────────────────
// The polling viewer re-read a fixed tail window every 1.5s. On the two
// backends with no cursor (procd, file) that means it re-serialized up to 500
// lines per tick and diffed them client-side, and a line written just after a
// poll waited the full interval to appear. Neither is fatal, but both are pure
// waste on a router.
//
// GetServiceLogs is NOT replaced. It still serves the initial window — the
// backlog the viewer opens with — and it remains the fallback when a proxy
// between here and the browser will not pass a stream. A viewer that polls is
// worse than one that streams; a viewer that shows nothing is worse than both.
//
// Query params:
//   - cursor: journald cursor from the seed request, so the stream resumes
//     exactly where the backlog ended. Ignored by the other backends, which
//     have no cursor to resume from.
func (h *Handler) StreamServiceLogs(ctx context.Context, c *app.RequestContext) {
	// Cancelled on every return path below, which is what kills the child
	// process the systemd and procd followers hold open. See service/follow.go.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, err := h.serviceController.FollowLogs(streamCtx)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	stream := NewSSEStream(c)

	// No log source on this host. Say so as a first-class event and close:
	// the client then falls back to polling, which will report the same
	// `source: none` and render the existing explanation.
	if events == nil {
		logStreamEnd("logs", stream.Event("unsupported", map[string]any{
			"source": service.LogSourceNone,
		}))
		return
	}

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return

		case event, ok := <-events:
			if !ok {
				// The follower stopped on its own — the child exited, or the
				// file went away. Tell the client rather than just hanging up,
				// so it can decide to fall back instead of reconnecting into
				// the same dead end.
				logStreamEnd("logs", stream.Event("ended", map[string]any{}))
				return
			}
			if len(event.Lines) == 0 {
				continue
			}
			if err := stream.Event("lines", map[string]any{
				"lines":  event.Lines,
				"cursor": event.Cursor,
			}); err != nil {
				// The client is gone. Returning cancels streamCtx, which kills
				// the child. Ignoring this error is exactly how a journalctl
				// leaks per closed tab.
				logStreamEnd("logs", err)
				return
			}

		case <-keepAlive.C:
			if err := stream.Comment(); err != nil {
				logStreamEnd("logs", err)
				return
			}
		}
	}
}
