package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	singjson "github.com/sagernet/sing/common/json"
)

// GetConfigSubset decodes only the requested top-level sections through the
// compiled sing-box schema. It must not fail because an unrelated section
// uses a newer core feature.
func (m *Manager) GetConfigSubset(names ...string) (*SingBoxConfig, error) {
	document, err := m.GetConfigDocument()
	if err != nil {
		return nil, err
	}
	var allSections map[string]json.RawMessage
	if err := json.Unmarshal(document.Raw, &allSections); err != nil {
		return nil, fmt.Errorf("failed to read config sections: %w", err)
	}
	sections := make(map[string]json.RawMessage, len(names))
	for _, name := range names {
		raw, ok := allSections[name]
		if ok {
			sections[name] = raw
		}
	}
	encoded, err := json.Marshal(sections)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config subset: %w", err)
	}
	var cfg SingBoxConfig
	if err := singjson.UnmarshalContext(CreateContext(context.Background()), encoded, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config subset: %w", err)
	}
	return &cfg, nil
}

// ClashAPISettings is the stable scalar subset shared by panel clients. It is
// independent of sing-box option structs so newer experimental fields cannot
// prevent controller access.
type ClashAPISettings struct {
	ExternalController    string `json:"external_controller"`
	Secret                string `json:"secret"`
	ExternalUI            string `json:"external_ui"`
	ExternalUIDownloadURL string `json:"external_ui_download_url"`
}

func (m *Manager) GetClashAPISettings() (ClashAPISettings, error) {
	raw, ok, err := m.GetConfigSection("experimental")
	if err != nil || !ok {
		return ClashAPISettings{}, err
	}
	var experimental struct {
		ClashAPI ClashAPISettings `json:"clash_api"`
	}
	if err := json.Unmarshal(raw, &experimental); err != nil {
		return ClashAPISettings{}, fmt.Errorf("failed to parse experimental.clash_api: %w", err)
	}
	return experimental.ClashAPI, nil
}

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
