package configuration

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

func (h *Service) GetInbounds(ctx context.Context) (any, error) {
	inbounds, err := h.readRawArraySection("inbounds")
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	return map[string]any{"inbounds": inbounds}, nil
}

func (h *Service) GetInboundByTag(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag
	inbounds, err := h.readRawArraySection("inbounds")
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	for _, inbound := range inbounds {
		if rawStringField(inbound, "tag") == tag {
			return inbound, nil
		}
	}
	return nil, fault.New(fault.Missing, "inbound not found")
}

func (h *Service) AddInbound(ctx context.Context, body []byte) (any, error) {
	body, inbound, bindErr := bindInbound(ctx, body)
	if bindErr != nil {
		return nil, bindErr
	}
	if err := validateInbound(inbound); err != nil {
		return nil, fault.New(fault.Input, err.Error())
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
		return nil, err
	}
	return map[string]any{"message": "inbound added successfully", "tag": inbound.Tag}, nil
}

func (h *Service) UpdateInbound(ctx context.Context, body []byte, pathTag string) (any, error) {
	tag := pathTag
	_, inbound, bindErr := bindInbound(ctx, body)
	if bindErr != nil {
		return nil, bindErr
	}
	inbound.Tag = tag
	if err := validateInbound(inbound); err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	raw, err := withRawStringField(body, "tag", tag)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid inbound configuration: "+err.Error())
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
		return nil, err
	}
	return map[string]any{"message": "inbound updated successfully", "tag": tag}, nil
}

func (h *Service) DeleteInbound(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag
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
		return nil, err
	}
	return map[string]any{"message": "inbound deleted successfully", "tag": tag}, nil
}
