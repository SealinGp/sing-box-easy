package v1_13_0

import (
	"context"
	stdjson "encoding/json"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

func (h *Handler) GetInbounds(ctx context.Context, c *app.RequestContext) {
	inbounds, err := h.readRawArraySection("inbounds")
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"inbounds": inbounds})
}

func (h *Handler) GetInboundByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	inbounds, err := h.readRawArraySection("inbounds")
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	for _, inbound := range inbounds {
		if rawStringField(inbound, "tag") == tag {
			respOK(ctx, c, inbound)
			return
		}
	}
	respErr(ctx, c, CodeNotFound, "inbound not found")
}

func (h *Handler) AddInbound(ctx context.Context, c *app.RequestContext) {
	body, inbound, ok := h.bindInbound(ctx, c)
	if !ok {
		return
	}
	if err := validateInbound(inbound); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	err := h.updateRawArraySection(ctx, "inbounds", func(inbounds []stdjson.RawMessage) ([]stdjson.RawMessage, error) {
		for _, existing := range inbounds {
			if rawStringField(existing, "tag") == inbound.Tag {
				return nil, fmt.Errorf("inbound with tag '%s' already exists", inbound.Tag)
			}
		}
		return append(inbounds, cloneRaw(body)), nil
	})
	if err != nil {
		respondConfigError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "inbound added successfully", "tag": inbound.Tag})
}

func (h *Handler) UpdateInbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	_, inbound, ok := h.bindInbound(ctx, c)
	if !ok {
		return
	}
	inbound.Tag = tag
	if err := validateInbound(inbound); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	raw, err := withRawStringField(c.Request.Body(), "tag", tag)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid inbound configuration: "+err.Error())
		return
	}
	err = h.updateRawArraySection(ctx, "inbounds", func(inbounds []stdjson.RawMessage) ([]stdjson.RawMessage, error) {
		for i, existing := range inbounds {
			if rawStringField(existing, "tag") == tag {
				inbounds[i] = raw
				return inbounds, nil
			}
		}
		return nil, fmt.Errorf("inbound not found")
	})
	if err != nil {
		respondConfigError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "inbound updated successfully", "tag": tag})
}

func (h *Handler) DeleteInbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	err := h.updateRawArraySection(ctx, "inbounds", func(inbounds []stdjson.RawMessage) ([]stdjson.RawMessage, error) {
		updated := make([]stdjson.RawMessage, 0, len(inbounds))
		for _, inbound := range inbounds {
			if rawStringField(inbound, "tag") != tag {
				updated = append(updated, inbound)
			}
		}
		if len(updated) == len(inbounds) {
			return nil, fmt.Errorf("inbound not found")
		}
		return updated, nil
	})
	if err != nil {
		respondConfigError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "inbound deleted successfully", "tag": tag})
}

func (h *Handler) bindInbound(ctx context.Context, c *app.RequestContext) ([]byte, option.Inbound, bool) {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return nil, option.Inbound{}, false
	}
	var inbound option.Inbound
	if err := singjson.UnmarshalContext(config.CreateContext(ctx), body, &inbound); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid inbound configuration: "+err.Error())
		return nil, option.Inbound{}, false
	}
	return body, inbound, true
}
