package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	singjson "github.com/sagernet/sing/common/json"
)

type outboundRecord struct {
	raw   json.RawMessage
	typed *Outbound
}

func decodeOutboundRecords(raw json.RawMessage) ([]outboundRecord, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("failed to parse outbounds array: %w", err)
	}

	jsonCtx := CreateContext(context.Background())
	records := make([]outboundRecord, 0, len(items))
	for _, item := range items {
		record := outboundRecord{raw: item}
		var outbound Outbound
		if err := singjson.UnmarshalContext(jsonCtx, item, &outbound); err == nil {
			record.typed = &outbound
		}
		records = append(records, record)
	}
	return records, nil
}

func typedOutbounds(records []outboundRecord) []Outbound {
	outbounds := make([]Outbound, 0, len(records))
	for _, record := range records {
		if record.typed != nil {
			outbounds = append(outbounds, *record.typed)
		}
	}
	return outbounds
}

func encodeUpdatedOutbounds(records []outboundRecord, outbounds []Outbound) (json.RawMessage, error) {
	items := make([]json.RawMessage, 0, len(outbounds)+len(records))
	used := make([]bool, len(outbounds))
	jsonCtx := CreateContext(context.Background())

	// Walk the original array first. Opaque records remain at their exact
	// position; recognized records are replaced, retained, or deleted in place.
	// This is semantically important because route.final defaults to the first
	// outbound when it is empty.
	for _, record := range records {
		if record.typed == nil {
			items = append(items, record.raw)
			continue
		}

		matched := -1
		for i := range outbounds {
			if !used[i] && outbounds[i].Tag == record.typed.Tag {
				matched = i
				break
			}
		}
		if matched < 0 {
			continue // The mutator deleted or renamed this record.
		}
		used[matched] = true
		outbound := outbounds[matched]
		if reflect.DeepEqual(*record.typed, outbound) {
			// Preserve the original object when the mutator did not change it,
			// including fields the compiled adapter does not understand.
			items = append(items, record.raw)
			continue
		}
		encoded, err := singjson.MarshalContext(jsonCtx, outbound)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outbound %q: %w", outbound.Tag, err)
		}
		items = append(items, encoded)
	}

	// Additions and renames have no original slot. Append them in the order
	// supplied by the mutator without disturbing any surviving record.
	for i, outbound := range outbounds {
		if used[i] {
			continue
		}
		encoded, err := singjson.MarshalContext(jsonCtx, outbound)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outbound %q: %w", outbound.Tag, err)
		}
		items = append(items, encoded)
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal outbounds array: %w", err)
	}
	return encoded, nil
}

// GetOutboundsConfig decodes only recognized outbounds. Unknown protocols stay
// in the raw document and are ignored by typed subscription diff logic.
func (m *Manager) GetOutboundsConfig() (*SingBoxConfig, error) {
	document, err := m.GetConfigDocument()
	if err != nil {
		return nil, err
	}
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(document.Raw, &sections); err != nil {
		return nil, fmt.Errorf("failed to read config sections: %w", err)
	}
	records, err := decodeOutboundRecords(sections["outbounds"])
	if err != nil {
		return nil, err
	}
	cfg := &SingBoxConfig{}
	cfg.Outbounds = typedOutbounds(records)
	return cfg, nil
}

// UpdateOutboundsConfig applies one typed outbound mutation while retaining all
// unrelated raw sections and every outbound the compiled adapter cannot read.
func (m *Manager) UpdateOutboundsConfig(ctx context.Context, updateFn func(*SingBoxConfig) error) error {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()

	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	document, err := NewConfigDocument(raw)
	if err != nil {
		return err
	}
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(document.Raw, &sections); err != nil {
		return fmt.Errorf("failed to read config sections: %w", err)
	}
	records, err := decodeOutboundRecords(sections["outbounds"])
	if err != nil {
		return err
	}
	cfg := &SingBoxConfig{}
	cfg.Outbounds = typedOutbounds(records)
	if err := updateFn(cfg); err != nil {
		return err
	}
	sections["outbounds"], err = encodeUpdatedOutbounds(records, cfg.Outbounds)
	if err != nil {
		return err
	}
	updated, err := json.MarshalIndent(sections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config document: %w", err)
	}
	return m.saveDocumentLocked(ctx, ConfigDocument{Raw: updated})
}
