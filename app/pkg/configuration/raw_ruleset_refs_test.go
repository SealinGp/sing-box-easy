package configuration

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRawRuleSetCascadeSupportsNewerDNSActions(t *testing.T) {
	rules := []json.RawMessage{
		json.RawMessage(`{"action":"evaluate","server":"router","tag":"result","rule_set":["remove","keep"]}`),
		json.RawMessage(`{"action":"respond","match_response":"result","rule_set":"remove"}`),
		json.RawMessage(`{"action":"route","server":"router","rule_set":"remove"}`),
	}
	updated, err := scrubRawRules(rules, "remove", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 2 || !bytes.Contains(updated[0], []byte(`"action":"evaluate"`)) || !bytes.Contains(updated[0], []byte(`"keep"`)) || !bytes.Contains(updated[1], []byte(`"match_response":"result"`)) {
		t.Fatalf("unexpected cascade result: %s", updated)
	}
}
