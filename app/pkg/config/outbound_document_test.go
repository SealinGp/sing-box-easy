package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateOutboundsConfigPreservesNewerAndUnknownConfiguration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{
  "dns":{"rules":[{"action":"evaluate","server":"dns_router","future_rule_field":true}]},
  "future_section":{"value":7},
  "outbounds":[
    {"type":"future-protocol","tag":"future-node","token":"keep-secret"},
    {"type":"direct","tag":"direct","future_outbound_field":"keep"}
  ]
}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(configPath, "sing-box", "")
	manager.core = &fakeCoreAdapter{version: CoreVersion{Major: 1, Minor: 14, Patch: 0, Raw: "1.14.0"}}

	if err := manager.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
		cfg.Outbounds = append(cfg.Outbounds, Outbound{
			Type: "block",
			Tag:  "blocked",
		})
		return nil
	}); err != nil {
		t.Fatalf("UpdateOutboundsConfig() error = %v", err)
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(got, &document); err != nil {
		t.Fatal(err)
	}
	var futureSection map[string]int
	if err := json.Unmarshal(document["future_section"], &futureSection); err != nil || futureSection["value"] != 7 {
		t.Fatalf("future section changed or was lost: %s (%v)", document["future_section"], err)
	}
	var dns map[string]json.RawMessage
	if err := json.Unmarshal(document["dns"], &dns); err != nil {
		t.Fatal(err)
	}
	if len(dns["rules"]) == 0 || !containsJSONField(dns["rules"], "future_rule_field") {
		t.Fatalf("new DNS action fields changed or were lost: %s", document["dns"])
	}

	var outbounds []map[string]json.RawMessage
	if err := json.Unmarshal(document["outbounds"], &outbounds); err != nil {
		t.Fatal(err)
	}
	byTag := make(map[string]map[string]json.RawMessage)
	tags := make([]string, 0, len(outbounds))
	for _, outbound := range outbounds {
		var tag string
		if err := json.Unmarshal(outbound["tag"], &tag); err == nil {
			byTag[tag] = outbound
			tags = append(tags, tag)
		}
	}
	// route.final defaults to the first outbound, so opaque entries must stay
	// in place during a mutation that does not own them.
	if len(tags) == 0 || tags[0] != "future-node" {
		t.Fatalf("unknown first outbound moved and changed the implicit route.final: %v", tags)
	}
	if _, ok := byTag["blocked"]; !ok {
		t.Fatalf("new outbound was not added: %s", document["outbounds"])
	}
	if string(byTag["future-node"]["token"]) != `"keep-secret"` {
		t.Fatalf("unknown outbound was not preserved: %s", document["outbounds"])
	}
	if string(byTag["direct"]["future_outbound_field"]) != `"keep"` {
		t.Fatalf("unknown field on untouched known outbound was not preserved: %s", document["outbounds"])
	}
}

func containsJSONField(raw []byte, field string) bool {
	var values []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return false
	}
	for _, value := range values {
		if _, ok := value[field]; ok {
			return true
		}
	}
	return false
}
