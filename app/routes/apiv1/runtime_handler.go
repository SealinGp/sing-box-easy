package apiv1

import (
	"context"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/proxies"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetRuntimeProxies returns the groups of the running sing-box.
func (h *Handler) GetRuntimeProxies(ctx context.Context, c *app.RequestContext) {
	view, err := h.proxiesModule.View(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, view)
}

// SelectRuntimeProxy switches a selector group to one of its members.
func (h *Handler) SelectRuntimeProxy(ctx context.Context, c *app.RequestContext) {
	var command proxies.SelectCommand
	if err := c.Bind(&command); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	if err := h.proxiesModule.Select(ctx, command); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, command)
}

// TestRuntimeProxy URL-tests one outbound.
func (h *Handler) TestRuntimeProxy(ctx context.Context, c *app.RequestContext) {
	var command proxies.TestCommand
	if err := c.Bind(&command); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.proxiesModule.TestNode(ctx, command)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}

// TestRuntimeGroup URL-tests every member of a group.
func (h *Handler) TestRuntimeGroup(ctx context.Context, c *app.RequestContext) {
	var command proxies.TestCommand
	if err := c.Bind(&command); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.proxiesModule.TestGroup(ctx, command)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}

// StreamRuntimeConnections streams the live connection table (SSE), one frame
// per second. `?rows=0` asks for the totals alone — what the Overview card
// charts — so that card does not pull the whole table to read three numbers.
func (h *Handler) StreamRuntimeConnections(ctx context.Context, c *app.RequestContext) {
	run, err := h.trafficService().PrepareConnections()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	// Cancelled on every return path: the poller must not outlive the tab.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := NewSSEStream(c)
	summaryOnly := string(c.Query("rows")) == "0"
	err = run.Run(streamCtx, func(frame *traffic.ConnectionsFrame) error {
		if summaryOnly {
			frame = frame.Summary()
		}
		return stream.Event("frame", frame)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		logStreamEnd("connections", stream.Error(CodeServiceError, err.Error()))
	}
}

// CloseRuntimeConnection closes one connection.
func (h *Handler) CloseRuntimeConnection(ctx context.Context, c *app.RequestContext) {
	if err := h.trafficService().CloseConnection(ctx, c.Param("id")); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK[any](ctx, c, nil)
}

// CloseRuntimeConnections closes every connection.
func (h *Handler) CloseRuntimeConnections(ctx context.Context, c *app.RequestContext) {
	if err := h.trafficService().CloseAllConnections(ctx); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK[any](ctx, c, nil)
}
