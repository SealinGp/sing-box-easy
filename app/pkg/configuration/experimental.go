package configuration

import (
	"context"
	"encoding/json"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

func (s *Service) ReadSection(ctx context.Context, section string) (any, error) {
	raw, ok, err := s.configManager.GetConfigSection(section)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]any{}, nil
	}
	return raw, nil
}
func (s *Service) ReplaceSection(ctx context.Context, section string, body []byte) error {
	if err := requireJSONObject(body); err != nil {
		return fault.New(fault.Invalid, err.Error())
	}
	return s.configManager.UpdateConfigSection(ctx, section, func(json.RawMessage) (json.RawMessage, error) { return cloneRaw(body), nil })
}
func (s *Service) Experimental(ctx context.Context, name string) (any, error) {
	raw, _, err := s.configManager.GetConfigSection("experimental")
	if err != nil {
		return nil, err
	}
	obj, err := readRawObjectSection(raw, "experimental")
	if err != nil {
		return nil, err
	}
	if child, ok := obj[name]; ok {
		return child, nil
	}
	return map[string]any{}, nil
}
func (s *Service) ReplaceExperimental(ctx context.Context, name string, body []byte) error {
	if name == "cache_file" {
		body = dropEmptyJSONFields(body, "rdrc_timeout")
	}
	if err := requireJSONObject(body); err != nil {
		return fault.New(fault.Invalid, err.Error())
	}
	return s.configManager.UpdateConfigSection(ctx, "experimental", func(raw json.RawMessage) (json.RawMessage, error) {
		obj, err := readRawObjectSection(raw, "experimental")
		if err != nil {
			return nil, err
		}
		obj[name] = cloneRaw(body)
		return json.Marshal(obj)
	})
}
