package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dualstack"
	"github.com/cloudwego/hertz/pkg/app"
)

type DualStackRequest = diagnostics.DualStackRequest

// ProbeDualStack tests a domain over IPv4 and IPv6 in one pass.
//
// POST because it performs live DNS queries and, unless skip_dial is set,
// opens real connections from this host.
func (h *Handler) ProbeDualStack(ctx context.Context, c *app.RequestContext) {
	var req DualStackRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	run, err := h.diagnostics().PrepareDualStack(req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	// The run holds a handle on sing-box's cache file for rule-set
	// evaluation, so it is released on every return path.
	defer run.Close()

	result, err := run.Run(ctx)
	if err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}
	respOK(ctx, c, result)
}

// StreamProbeDualStack is the same probe, reported phase by phase.
//
// Worth streaming because the phases carry very different latencies: the
// config read is instant, the resolve is a round trip per family, and the
// traffic phase is a dial plus connection-table polls per address — seconds,
// on a request that may cover ten domains. Unary, that is one silent wait.
//
// The stream's context bounds the dials, so abandoning the request stops the
// probe opening connections for a browser tab that has closed.
func (h *Handler) StreamProbeDualStack(ctx context.Context, c *app.RequestContext) {
	var req DualStackRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	run, err := h.diagnostics().PrepareDualStack(req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	defer run.Close()

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := NewSSEStream(c)
	result, err := run.Stream(streamCtx, func(stage dualstack.Stage, partial *dualstack.Result) error {
		if Done(streamCtx) {
			return context.Canceled
		}
		return stream.Event(string(stage), partial)
	})
	if err != nil {
		logStreamEnd("dual-stack", stream.Error(CodeValidationError, err.Error()))
		return
	}
	logStreamEnd("dual-stack", stream.Event("done", result))
}
