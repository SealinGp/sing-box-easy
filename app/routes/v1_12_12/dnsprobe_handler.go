package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dns"
	"github.com/cloudwego/hertz/pkg/app"
)

type DNSProbeRequest = diagnostics.DNSProbeRequest

func (h *Handler) ProbeDNS(ctx context.Context, c *app.RequestContext) {
	var req DNSProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	run, err := h.diagnostics().PrepareDNS(req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	result, err := run.Run()
	if err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) StreamProbeDNS(ctx context.Context, c *app.RequestContext) {
	var req DNSProbeRequest
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	run, err := h.diagnostics().PrepareDNS(req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := NewSSEStream(c)
	result, err := run.Stream(func(stage dnsprobe.Stage, partial *dnsprobe.Result) error {
		if Done(streamCtx) {
			return context.Canceled
		}
		return stream.Event(string(stage), partial)
	})
	if err != nil {
		logStreamEnd("dns-probe", stream.Error(CodeValidationError, err.Error()))
		return
	}
	logStreamEnd("dns-probe", stream.Event("done", result))
}
