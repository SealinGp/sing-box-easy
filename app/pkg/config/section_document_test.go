package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateConfigSectionPreservesUnownedSections(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{
  "dns":{"rules":[{"action":"evaluate","server":"dns_router"}]},
  "future_section":{"value":7},
  "outbounds":[{"type":"future-protocol","tag":"first"}]
}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(configPath, "sing-box", "")
	manager.core = &fakeCoreAdapter{version: CoreVersion{Major: 1, Minor: 14, Raw: "1.14.0"}}

	err := manager.UpdateConfigSection(context.Background(), "dns", func(section json.RawMessage) (json.RawMessage, error) {
		var dns map[string]json.RawMessage
		if err := json.Unmarshal(section, &dns); err != nil {
			return nil, err
		}
		dns["timeout"] = json.RawMessage(`"8s"`)
		return json.Marshal(dns)
	})
	if err != nil {
		t.Fatalf("UpdateConfigSection() error = %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(got, &document); err != nil {
		t.Fatal(err)
	}
	var dns map[string]json.RawMessage
	if err := json.Unmarshal(document["dns"], &dns); err != nil || !containsJSONField(dns["rules"], "server") {
		t.Fatalf("newer DNS rule was not preserved: %s (%v)", document["dns"], err)
	}
	var future map[string]int
	if err := json.Unmarshal(document["future_section"], &future); err != nil || future["value"] != 7 {
		t.Fatalf("future section changed: %s (%v)", document["future_section"], err)
	}
	var outbounds []map[string]any
	if err := json.Unmarshal(document["outbounds"], &outbounds); err != nil || len(outbounds) != 1 || outbounds[0]["tag"] != "first" {
		t.Fatalf("outbounds changed: %s (%v)", document["outbounds"], err)
	}
}
