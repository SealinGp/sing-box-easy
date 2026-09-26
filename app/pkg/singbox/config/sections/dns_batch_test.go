package sections

import (
	stdjson "encoding/json"
	"strings"
	"testing"
)

// decodeBatchRequest is the parsing half of AddDNSRulesBatch, exercised
// without a config manager: the atomicity guarantee rests on every entry being
// checked BEFORE the document is touched, so that check is what is pinned here.
func decodeBatchRequest(t *testing.T, body string) ([]stdjson.RawMessage, error) {
	t.Helper()
	var request struct {
		Rules []stdjson.RawMessage `json:"rules"`
	}
	if err := stdjson.Unmarshal([]byte(body), &request); err != nil {
		return nil, err
	}
	for _, rule := range request.Rules {
		if err := requireJSONObject(rule); err != nil {
			return nil, err
		}
	}
	return request.Rules, nil
}

func TestBatchRejectsANonObjectRuleBeforeWriting(t *testing.T) {
	// The rule at index 2 is a string. If validation happened during the
	// append, the first two would already be in the config — and for an
	// evaluate/respond group that half-state breaks resolution.
	_, err := decodeBatchRequest(t, `{"rules":[
	  {"action":"evaluate","server":"a","tag":"g_a"},
	  {"action":"evaluate","server":"b","tag":"g_b"},
	  "not-a-rule"
	]}`)
	if err == nil {
		t.Fatal("expected the malformed entry to be rejected")
	}
}

func TestBatchAcceptsAWellFormedGroup(t *testing.T) {
	rules, err := decodeBatchRequest(t, `{"rules":[
	  {"action":"evaluate","server":"a","tag":"g_a"},
	  {"action":"respond","match_response":"g_a","race":true}
	]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(rules))
	}
	if !strings.Contains(string(rules[1]), "match_response") {
		t.Fatalf("rule bodies must pass through verbatim, got %s", rules[1])
	}
}

func TestBatchPreservesOrder(t *testing.T) {
	// Order is the correctness requirement: every evaluate must precede the
	// respond that references it.
	rules, err := decodeBatchRequest(t, `{"rules":[
	  {"tag":"first"},{"tag":"second"},{"tag":"third"}
	]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, want := range []string{"first", "second", "third"} {
		if !strings.Contains(string(rules[i]), want) {
			t.Fatalf("position %d = %s, want %s", i, rules[i], want)
		}
	}
}
