package v1_13_0

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/trafficflow"
	"github.com/cloudwego/hertz/pkg/app"
)

// trafficFlowInterval is the sample cadence. One second is what sing-box's own
// WebSocket ticks at, and the finest resolution a rate derived from byte
// counters is meaningful at.
const trafficFlowInterval = time.Second

// StreamTrafficFlow pushes the live traffic overlay for the Overview diagram.
//
// GET /traffic/flow/stream?source_ip=&host=  (SSE)
//
// WHY THE PANEL PROXIES THIS INSTEAD OF THE BROWSER OPENING A WEBSOCKET
// ─────────────────────────────────────────────────────────────────────
// The Clash API's WebSocket endpoints authenticate with `?token=<secret>` in
// the URL — browsers cannot set headers on an upgrade — so a dashboard that
// talks to sing-box directly writes the secret into every proxy and access log
// on the path, and needs `external_controller` bound to a reachable address
// rather than the loopback it usually is. Neither is true here: the secret
// stays in this process (clashapi.New reads it from the config), the poll is
// plain HTTP to loopback, and what reaches the browser is an aggregate of a
// few kilobytes per second over the panel's own Bearer-authenticated SSE.
//
// Events: `frame` per sample; `error` (terminal) when sing-box cannot be
// reached — which includes "not running", the case the Live toggle is meant
// to be disabled for, but a race between the status poll and a stop is
// ordinary, so it is reported rather than assumed away.
func (h *Handler) StreamTrafficFlow(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}

	client, err := clashapi.New(cfg.Options.Experimental)
	if err != nil {
		if errors.Is(err, clashapi.ErrDisabled) {
			respErr(ctx, c, CodeServiceError, "live traffic needs experimental.clash_api.external_controller to be set")
			return
		}
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}

	filter := trafficflow.Filter{
		SourceIP: strings.TrimSpace(string(c.Query("source_ip"))),
		Host:     strings.TrimSpace(string(c.Query("host"))),
	}

	// Cancelled on every return path: the poll loop must not outlive the
	// client, or a closed tab keeps sing-box's connection list being
	// serialised once a second for nobody.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := NewSSEStream(c)

	runErr := trafficflow.Run(streamCtx, client, trafficflow.Options{
		Interval: trafficFlowInterval,
		Filter:   filter,
	}, func(frame *trafficflow.Frame) error {
		return stream.Event("frame", frame)
	})

	switch {
	case runErr == nil, errors.Is(runErr, context.Canceled):
		return
	case errors.Is(runErr, clashapi.ErrUnauthorized):
		logStreamEnd("traffic", stream.Error(CodeServiceError, runErr.Error()))
	default:
		// Most often: sing-box is not running, so the controller refused the
		// connection. Say so as an event the client can render; a bare
		// disconnect looks like the panel failing.
		logStreamEnd("traffic", stream.Error(CodeServiceError, "sing-box is not reachable: "+runErr.Error()))
	}
}
