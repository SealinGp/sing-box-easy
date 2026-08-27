package v1_13_0

// The panel's own log, as the second tab on the Logs page.
//
// Deliberately the same request/response shape as the sing-box pair
// (`/service/logs` + `/service/logs/stream`) so the viewer switches feeds by
// swapping two URLs rather than by branching on which log it is showing. The
// `cursor` field is present and always empty: a ring buffer has no resumable
// position, and reporting one it could not honour would be worse than admitting
// it has none — the client already treats an empty cursor as "re-read the
// window", which is exactly right here.

import (
	"context"
	"strconv"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/applog"
	"github.com/cloudwego/hertz/pkg/app"
)

// appLogSource labels this feed for the UI, alongside the sing-box feed's
// journald/syslog/file/none. It is its own value because the caveat is its own:
// this log lives only as long as the process.
const appLogSource = "memory"

// GetAppLogs returns the most recent sing-box-easy log lines.
//
// GET /system/logs?lines=N
func (h *Handler) GetAppLogs(ctx context.Context, c *app.RequestContext) {
	lines, _ := strconv.Atoi(c.Query("lines"))
	if lines <= 0 {
		lines = 300
	}
	if lines > applog.DefaultCapacity {
		lines = applog.DefaultCapacity
	}

	respOK(ctx, c, map[string]any{
		"lines":  applog.Default().Tail(lines),
		"cursor": "",
		"source": appLogSource,
	})
}

// StreamAppLogs pushes sing-box-easy log lines as they are written.
//
// Cheaper than the sing-box stream in every respect: no child process, no file
// polling, just a channel the logger already fans out to. Which also means the
// failure mode that dominates the other stream — an orphaned `journalctl -f` —
// cannot occur here at all. Cancelling the context unsubscribes, and that is
// the whole of the cleanup.
//
// GET /system/logs/stream
func (h *Handler) StreamAppLogs(ctx context.Context, c *app.RequestContext) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := applog.Default().Subscribe(streamCtx)

	stream := NewSSEStream(c)

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return

		case batch, ok := <-events:
			if !ok {
				return
			}
			if len(batch) == 0 {
				continue
			}
			if err := stream.Event("lines", map[string]any{
				"lines":  batch,
				"cursor": "",
			}); err != nil {
				// The client is gone. Returning cancels streamCtx, which
				// unsubscribes — leaving the subscription behind would make the
				// logger fan out to a channel nobody reads until it fills and
				// starts counting drops forever.
				logStreamEnd("app-logs", err)
				return
			}

		case <-keepAlive.C:
			if err := stream.Comment(); err != nil {
				logStreamEnd("app-logs", err)
				return
			}
		}
	}
}
