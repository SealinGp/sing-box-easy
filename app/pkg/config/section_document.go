package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// GetConfigSection returns one top-level JSON value without decoding any
// sibling section. The returned bytes are detached from the read buffer.
func (m *Manager) GetConfigSection(name string) (json.RawMessage, bool, error) {
	document, err := m.GetConfigDocument()
	if err != nil {
		return nil, false, err
	}
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(document.Raw, &sections); err != nil {
		return nil, false, fmt.Errorf("failed to read config sections: %w", err)
	}
	section, ok := sections[name]
	if !ok {
		return nil, false, nil
	}
	return bytes.Clone(section), true, nil
}

// UpdateConfigSection applies an atomic mutation to one top-level section.
// Every sibling is retained as raw JSON and the complete candidate is checked
// by the installed core before promotion.
func (m *Manager) UpdateConfigSection(
	ctx context.Context,
	name string,
	updateFn func(json.RawMessage) (json.RawMessage, error),
) error {
	return m.UpdateConfigSections(ctx, func(sections map[string]json.RawMessage) error {
		updatedSection, err := updateFn(bytes.Clone(sections[name]))
		if err != nil {
			return err
		}
		if len(updatedSection) == 0 || !json.Valid(updatedSection) {
			return &ValidationError{
				Stage: ValidationStageJSONSyntax,
				Err:   fmt.Errorf("updated %s section must be valid JSON", name),
			}
		}
		sections[name] = updatedSection
		return nil
	})
}

// UpdateConfigSections atomically changes one or more top-level raw sections.
// It is used for operations such as cascading rule-set deletion, where route
// and DNS references must change in the same core validation transaction.
func (m *Manager) UpdateConfigSections(
	ctx context.Context,
	updateFn func(map[string]json.RawMessage) error,
) error {
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

	if err := updateFn(sections); err != nil {
		return err
	}
	for name, section := range sections {
		if len(section) == 0 || !json.Valid(section) {
			return &ValidationError{
				Stage: ValidationStageJSONSyntax,
				Err:   fmt.Errorf("updated %s section must be valid JSON", name),
			}
		}
	}
	updated, err := json.MarshalIndent(sections, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config document: %w", err)
	}
	return m.saveDocumentLocked(ctx, ConfigDocument{Raw: updated})
}
