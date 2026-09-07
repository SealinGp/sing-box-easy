package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	singjson "github.com/sagernet/sing/common/json"
)

// ValidationStage identifies which validation boundary rejected a document.
type ValidationStage string

const (
	ValidationStageJSONSyntax   ValidationStage = "json_syntax"
	ValidationStagePanelGuard   ValidationStage = "panel_guard"
	ValidationStageStagingWrite ValidationStage = "staging_write"
	ValidationStageCoreCheck    ValidationStage = "core_check"
)

// ValidationError gives API clients stable diagnostic fields without hiding
// the installed core's actionable error message.
type ValidationError struct {
	Stage       ValidationStage
	CoreVersion string
	Err         error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ConfigDocument stores the complete user configuration without decoding it
// through the version-pinned sing-box option structs.
type ConfigDocument struct {
	Raw []byte
}

// NewConfigDocument checks only version-independent JSON structure. Semantic
// validity belongs to the installed core.
func NewConfigDocument(raw []byte) (ConfigDocument, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return ConfigDocument{}, &ValidationError{
			Stage: ValidationStageJSONSyntax,
			Err:   errors.New("configuration must be valid JSON"),
		}
	}
	if trimmed[0] != '{' {
		return ConfigDocument{}, &ValidationError{
			Stage: ValidationStagePanelGuard,
			Err:   errors.New("configuration must be a JSON object"),
		}
	}
	return ConfigDocument{Raw: bytes.Clone(raw)}, nil
}

// MarshalJSON embeds the original document as an object when used in the
// standard API response envelope.
func (d ConfigDocument) MarshalJSON() ([]byte, error) {
	if len(d.Raw) == 0 {
		return []byte("null"), nil
	}
	return d.Raw, nil
}

// GetConfigDocument reads the active config without interpreting its fields.
func (m *Manager) GetConfigDocument() (ConfigDocument, error) {
	raw, err := os.ReadFile(m.configPath)
	if err != nil {
		return ConfigDocument{}, fmt.Errorf("failed to read config file: %w", err)
	}
	document, err := NewConfigDocument(raw)
	if err != nil {
		return ConfigDocument{}, fmt.Errorf("failed to parse config file: %w", err)
	}
	return document, nil
}

// GetRuntimeConfig decodes only the sections needed by service lifecycle
// integration. A new DNS or route feature therefore cannot prevent host
// network setup or log discovery.
func (m *Manager) GetRuntimeConfig() (*SingBoxConfig, error) {
	document, err := m.GetConfigDocument()
	if err != nil {
		return nil, err
	}
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(document.Raw, &sections); err != nil {
		return nil, fmt.Errorf("failed to read config sections: %w", err)
	}

	var cfg SingBoxConfig
	jsonCtx := CreateContext(context.Background())
	if raw := sections["log"]; len(raw) > 0 {
		if err := singjson.UnmarshalContext(jsonCtx, raw, &cfg.Log); err != nil {
			return nil, fmt.Errorf("failed to parse log config: %w", err)
		}
	}
	if raw := sections["inbounds"]; len(raw) > 0 {
		if err := singjson.UnmarshalContext(jsonCtx, raw, &cfg.Inbounds); err != nil {
			return nil, fmt.Errorf("failed to parse inbound config: %w", err)
		}
	}
	return &cfg, nil
}

// ValidateRawConfig validates an HTTP request body without passing it through
// the panel's compiled sing-box schema.
func (m *Manager) ValidateRawConfig(ctx context.Context, raw []byte) error {
	document, err := NewConfigDocument(raw)
	if err != nil {
		return err
	}
	return m.ValidateDocument(ctx, document)
}

// ValidateDocument stages an exact copy for the installed core and always
// removes it afterwards. It never changes the active configuration.
func (m *Manager) ValidateDocument(ctx context.Context, document ConfigDocument) error {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()
	return m.validateDocumentLocked(ctx, document)
}

func (m *Manager) validateDocumentLocked(ctx context.Context, document ConfigDocument) error {
	validated, err := NewConfigDocument(document.Raw)
	if err != nil {
		return err
	}
	document = validated
	if err := os.WriteFile(m.newConfigPath, document.Raw, 0600); err != nil {
		return &ValidationError{Stage: ValidationStageStagingWrite, Err: fmt.Errorf("failed to write temp config file: %w", err)}
	}
	defer os.Remove(m.newConfigPath)

	if err := m.core.Validate(ctx, m.newConfigPath); err != nil {
		version, _ := m.core.Version(ctx)
		return &ValidationError{Stage: ValidationStageCoreCheck, CoreVersion: version.Raw, Err: err}
	}
	return nil
}

// SaveRawConfig validates and atomically promotes a complete raw document.
func (m *Manager) SaveRawConfig(ctx context.Context, raw []byte) error {
	document, err := NewConfigDocument(raw)
	if err != nil {
		return err
	}
	return m.SaveDocument(ctx, document)
}

// SaveDocument fails closed: the proposed document must pass the installed
// core before the active configuration is snapshotted and replaced.
func (m *Manager) SaveDocument(ctx context.Context, document ConfigDocument) error {
	m.mutationMu.Lock()
	defer m.mutationMu.Unlock()
	return m.saveDocumentLocked(ctx, document)
}

func (m *Manager) saveDocumentLocked(ctx context.Context, document ConfigDocument) error {
	validated, err := NewConfigDocument(document.Raw)
	if err != nil {
		return err
	}
	document = validated
	if err := os.WriteFile(m.newConfigPath, document.Raw, 0600); err != nil {
		return &ValidationError{Stage: ValidationStageStagingWrite, Err: fmt.Errorf("failed to write temp config file: %w", err)}
	}
	if err := m.core.Validate(ctx, m.newConfigPath); err != nil {
		os.Remove(m.newConfigPath)
		version, _ := m.core.Version(ctx)
		return &ValidationError{Stage: ValidationStageCoreCheck, CoreVersion: version.Raw, Err: err}
	}

	m.snapshotCurrent()
	if err := os.Rename(m.newConfigPath, m.configPath); err != nil {
		os.Remove(m.newConfigPath)
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// ValidateCurrentConfig asks the installed core to check the active file
// directly, avoiding a typed read during start/restart/reload.
func (m *Manager) ValidateCurrentConfig(ctx context.Context) error {
	if err := m.core.Validate(ctx, m.configPath); err != nil {
		version, _ := m.core.Version(ctx)
		return &ValidationError{Stage: ValidationStageCoreCheck, CoreVersion: version.Raw, Err: err}
	}
	return nil
}

// CoreCapabilities returns the installed version and its UI feature gates.
func (m *Manager) CoreCapabilities(ctx context.Context) (CoreCapabilities, error) {
	version, err := m.core.Version(ctx)
	if err != nil {
		return CoreCapabilities{}, err
	}
	return CapabilitiesForCoreVersion(version), nil
}
