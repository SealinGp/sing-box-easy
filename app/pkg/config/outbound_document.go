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
	knownByTag := make(map[string][]outboundRecord)
	unknown := make([]json.RawMessage, 0)
	for _, record := range records {
		if record.typed == nil {
			unknown = append(unknown, record.raw)
			continue
		}
		knownByTag[record.typed.Tag] = append(knownByTag[record.typed.Tag], record)
	}

	items := make([]json.RawMessage, 0, len(outbounds)+len(unknown))
	jsonCtx := CreateContext(context.Background())
	for _, outbound := range outbounds {
		candidates := knownByTag[outbound.Tag]
		if len(candidates) > 0 && reflect.DeepEqual(*candidates[0].typed, outbound) {
			// Preserve the original object when the mutator did not change it,
			// including fields the compiled adapter does not understand.
			items = append(items, candidates[0].raw)
			knownByTag[outbound.Tag] = candidates[1:]
			continue
		}
		encoded, err := singjson.MarshalContext(jsonCtx, outbound)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal outbound %q: %w", outbound.Tag, err)
		}
		items = append(items, encoded)
	}
	// Records from protocols unknown to this panel are outside the mutation's
	// ownership. Keep them byte-for-byte as JSON values.
	items = append(items, unknown...)
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
