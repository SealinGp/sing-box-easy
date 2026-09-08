package configuration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func (h *Service) readRawArraySection(name string) ([]json.RawMessage, error) {
	raw, ok, err := h.configManager.GetConfigSection(name)
	if err != nil || !ok {
		return []json.RawMessage{}, err
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("failed to parse %s section: %w", name, err)
	}
	if items == nil {
		items = []json.RawMessage{}
	}
	return items, nil
}

func (h *Service) updateRawArraySection(
	ctx context.Context,
	name string,
	update func([]json.RawMessage) ([]json.RawMessage, error),
) error {
	return h.configManager.UpdateConfigSection(ctx, name, func(raw json.RawMessage) (json.RawMessage, error) {
		items := []json.RawMessage{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &items); err != nil {
				return nil, fmt.Errorf("failed to parse %s section: %w", name, err)
			}
		}
		updated, err := update(items)
		if err != nil {
			return nil, err
		}
		return json.Marshal(updated)
	})
}

func readRawObjectSection(raw json.RawMessage, name string) (map[string]json.RawMessage, error) {
	object := map[string]json.RawMessage{}
	if len(raw) == 0 || string(raw) == "null" {
		return object, nil
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("failed to parse %s section: %w", name, err)
	}
	return object, nil
}

func (h *Service) readClashAPISettings() (config.ClashAPISettings, error) {
	return h.configManager.GetClashAPISettings()
}
