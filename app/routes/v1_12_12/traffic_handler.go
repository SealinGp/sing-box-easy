package v1_13_0

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/traffic"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) StreamTrafficFlow(ctx context.Context, c *app.RequestContext) {
	run, err := h.trafficService().Prepare(trafficflow.Filter{SourceIP: string(c.Query("source_ip")), Host: string(c.Query("host"))})
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := NewSSEStream(c)
	err = run.Run(streamCtx, func(frame *trafficflow.Frame) error { return stream.Event("frame", frame) })
	if err != nil && !errors.Is(err, context.Canceled) {
		logStreamEnd("traffic", stream.Error(CodeServiceError, err.Error()))
	}
}
