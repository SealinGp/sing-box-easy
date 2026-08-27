package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/dnsprobe"
	"github.com/cloudwego/hertz/pkg/app"
)

// DNSProbeRequest asks what this deployment does with one domain.
type DNSProbeRequest struct {
	Domain string `json:"domain"`
	// Type is a DNS record type; defaults to A.
	Type string `json:"type"`
	// CompareServers additionally queries each directly reachable configured
	// resolver so their answers can be compared. Off by default because it
	// sends real traffic to every upstream.
	CompareServers bool `json:"compare_servers"`
}

// ProbeDNS resolves a domain through sing-box and explains the routing.
//
// POST /dns/probe
func (h *Handler) ProbeDNS(ctx context.Context, c *app.RequestContext) {
	var req DNSProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}

	result, err := dnsprobe.Run(&cfg.Options, dnsprobe.Options{
		Domain:         req.Domain,
		QueryType:      req.Type,
		CompareServers: req.CompareServers,
		Tailer:         h.logTailer(),
	})
	if err != nil {
		// The only errors here are invalid input; everything else degrades
		// into the result itself.
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	respOK(ctx, c, result)
}

// logTailer adapts the service controller's log tailing to what the probe
// needs, so the probe package does not depend on the service package.
func (h *Handler) logTailer() dnsprobe.LogTailer {
	return func(lines int, afterCursor string) ([]string, string, error) {
		chunk, err := h.serviceController.TailLogs(lines, afterCursor)
		if err != nil {
			return nil, "", err
		}
		return chunk.Lines, chunk.Cursor, nil
	}
}

// StreamProbeDNS runs a probe and reports each phase as it completes.
//
// WHAT IS ACTUALLY BEING STREAMED
// ───────────────────────────────
// The phases, because the phases are where the time goes. A probe holds the
// client for a live query over the Clash API, then a fixed 250ms log settle,
// then two log reads — and, with compare_servers on, one query to every
// configured resolver. Unary, that is one silent wait and then everything at
// once. Streamed, the rule ladder is on screen before the live query returns.
//
// What is NOT streamed is the per-rule verdict. dnsprobe.Attribute walks every
// rule in one synchronous pass, so pacing the rungs would mean the server
// sleeping between them. The client paces them and says so — see
// useRuleSequencer. A tool that exists to explain timing must not fake its own.
//
// POST /dns/probe/stream
func (h *Handler) StreamProbeDNS(ctx context.Context, c *app.RequestContext) {
	var req DNSProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeConfigError, err.Error())
		return
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := NewSSEStream(c)

	// Returning an error from the stage callback aborts the probe. That matters
	// most for compare_servers, whose remaining work is a query to every
	// configured upstream — real traffic, sent on behalf of a client that has
	// already hung up.
	onStage := func(stage dnsprobe.Stage, partial *dnsprobe.Result) error {
		if Done(streamCtx) {
			return context.Canceled
		}
		return stream.Event(string(stage), partial)
	}

	result, err := dnsprobe.RunStaged(&cfg.Options, dnsprobe.Options{
		Domain:         req.Domain,
		QueryType:      req.Type,
		CompareServers: req.CompareServers,
		Tailer:         h.logTailer(),
	}, onStage)
	if err != nil {
		// Invalid input. Everything else degrades into the result itself.
		logStreamEnd("dns-probe", stream.Error(CodeValidationError, err.Error()))
		return
	}

	// The terminal event carries the whole result, so a client that missed or
	// ignored the intermediate stages still ends up with exactly what the unary
	// endpoint would have returned.
	logStreamEnd("dns-probe", stream.Event("done", result))
}
