package config

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeCoreAdapter struct {
	version       CoreVersion
	validateErr   error
	validateCalls int
	validated     []byte
}

func (f *fakeCoreAdapter) Version(context.Context) (CoreVersion, error) {
	return f.version, nil
}

func (f *fakeCoreAdapter) Validate(_ context.Context, path string) error {
	f.validateCalls++
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	f.validated = data
	return f.validateErr
}

func TestValidateDocumentDelegatesUnknownFieldsToInstalledCore(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "config.json"), "sing-box", "")
	core := &fakeCoreAdapter{version: CoreVersion{Major: 1, Minor: 14, Patch: 0, Raw: "1.14.0"}}
	manager.core = core

	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","server":"dns_router","future_field":true}]},"future_section":{"enabled":true}}`)
	document, err := NewConfigDocument(raw)
	if err != nil {
		t.Fatalf("NewConfigDocument() error = %v", err)
	}

	if err := manager.ValidateDocument(context.Background(), document); err != nil {
		t.Fatalf("ValidateDocument() error = %v", err)
	}
	if core.validateCalls != 1 {
		t.Fatalf("core validation calls = %d, want 1", core.validateCalls)
	}
	if !bytes.Equal(core.validated, raw) {
		t.Fatalf("installed core received %s, want exact document %s", core.validated, raw)
	}
	if _, err := os.Stat(manager.newConfigPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("validation staging file was not removed: %v", err)
	}
}

func TestValidateRawConfigRejectsInvalidJSONBeforeCallingCore(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "config.json"), "sing-box", "")
	core := &fakeCoreAdapter{}
	manager.core = core

	err := manager.ValidateRawConfig(context.Background(), []byte(`{"dns":`))
	if err == nil {
		t.Fatal("ValidateRawConfig() error = nil, want syntax error")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ValidateRawConfig() error type = %T, want *ValidationError", err)
	}
	if validationErr.Stage != ValidationStageJSONSyntax {
		t.Fatalf("validation stage = %q, want %q", validationErr.Stage, ValidationStageJSONSyntax)
	}
	if core.validateCalls != 0 {
		t.Fatalf("core validation calls = %d, want 0", core.validateCalls)
	}
}

func TestSaveDocumentPreservesRawConfiguration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	baseline := []byte(`{"log":{"level":"info"}}`)
	if err := os.WriteFile(configPath, baseline, 0600); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(configPath, "sing-box", "")
	core := &fakeCoreAdapter{version: CoreVersion{Major: 1, Minor: 14, Patch: 0, Raw: "1.14.0"}}
	manager.core = core
	raw := []byte("{\n  \"dns\": {\"rules\": [{\"action\": \"evaluate\", \"unknown\": 7}]}\n}\n")
	document, err := NewConfigDocument(raw)
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.SaveDocument(context.Background(), document); err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("saved config = %q, want exact raw bytes %q", got, raw)
	}
}

func TestSaveDocumentDoesNotReplaceActiveConfigWhenCoreRejectsCandidate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	baseline := []byte(`{"log":{"level":"info"}}`)
	if err := os.WriteFile(configPath, baseline, 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(configPath, "sing-box", "")
	manager.core = &fakeCoreAdapter{
		version:     CoreVersion{Major: 1, Minor: 13, Patch: 11, Raw: "1.13.11"},
		validateErr: errors.New("unknown DNS rule action: evaluate"),
	}

	err := manager.SaveRawConfig(context.Background(), []byte(`{"dns":{"rules":[{"action":"evaluate"}]}}`))
	if err == nil {
		t.Fatal("SaveRawConfig() error = nil, want core validation failure")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Stage != ValidationStageCoreCheck {
		t.Fatalf("SaveRawConfig() error = %T %v, want core_check ValidationError", err, err)
	}
	if validationErr.CoreVersion != "1.13.11" {
		t.Fatalf("core version = %q, want 1.13.11", validationErr.CoreVersion)
	}
	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, baseline) {
		t.Fatalf("active config changed after failed validation: %s", got)
	}
}

func TestGetRuntimeConfigIgnoresUnrelatedFutureDNSActions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{
  "log":{"level":"debug","output":"/tmp/sing-box.log"},
  "dns":{"rules":[{"action":"evaluate","server":"dns_router"}]},
  "inbounds":[{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":7893}]
}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(configPath, "sing-box", "")
	cfg, err := manager.GetRuntimeConfig()
	if err != nil {
		t.Fatalf("GetRuntimeConfig() error = %v", err)
	}
	if cfg.Log == nil || cfg.Log.Output != "/tmp/sing-box.log" {
		t.Fatalf("runtime log config = %+v", cfg.Log)
	}
	if len(cfg.Inbounds) != 1 || cfg.Inbounds[0].Tag != "mixed-in" {
		t.Fatalf("runtime inbounds = %+v", cfg.Inbounds)
	}
}
