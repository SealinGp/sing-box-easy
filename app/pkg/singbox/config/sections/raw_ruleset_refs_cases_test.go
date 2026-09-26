package sections

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Ported from the typed implementation's suite (config/ruleset_refs_test.go)
// when that implementation was deleted as unused: these are the behaviours the
// raw cascade must keep, expressed as the JSON it actually operates on.

func rawRules(rules ...string) []json.RawMessage {
	out := make([]json.RawMessage, len(rules))
	for i, rule := range rules {
		out[i] = json.RawMessage(rule)
	}
	return out
}

func TestRawReferencesAction(t *testing.T) {
	cases := []struct {
		name       string
		rule       string
		wantAction string // "" = no reference
	}{
		{"sole matcher -> delete", `{"rule_set":["proxy"],"outbound":"x"}`, RefActionDelete},
		{"scalar sole matcher -> delete", `{"rule_set":"proxy","outbound":"x"}`, RefActionDelete},
		{"with other rule_set -> strip", `{"rule_set":["proxy","other"]}`, RefActionStrip},
		{"with domain matcher -> strip", `{"rule_set":["proxy"],"domain":["example.com"]}`, RefActionStrip},
		{"invert only -> delete (invert is not a matcher)", `{"rule_set":["proxy"],"invert":true}`, RefActionDelete},
		{"bool matcher -> strip", `{"rule_set":["proxy"],"source_ip_is_private":true}`, RefActionStrip},
		{"unreferenced -> none", `{"domain":["example.com"]}`, ""},
		{"different tag -> none", `{"rule_set":["something-else"]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs := rawReferences(rawRules(tc.rule), "proxy", RefScopeRoute, false)
			if tc.wantAction == "" {
				if len(refs) != 0 {
					t.Fatalf("expected no references, got %+v", refs)
				}
				return
			}
			if len(refs) != 1 {
				t.Fatalf("expected 1 reference, got %+v", refs)
			}
			if refs[0].Action != tc.wantAction || refs[0].Scope != RefScopeRoute || refs[0].Index != 0 {
				t.Errorf("got %+v, want route/0/%s", refs[0], tc.wantAction)
			}
		})
	}
}

func TestRawReferencesIndicesArePreMutation(t *testing.T) {
	refs := rawReferences(rawRules(
		`{"domain":["a.com"]}`,
		`{"rule_set":["proxy","keep"]}`,
		`{"rule_set":["other"]}`,
		`{"rule_set":["proxy"]}`,
	), "proxy", RefScopeRoute, false)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %+v", refs)
	}
	if refs[0].Index != 1 || refs[0].Action != RefActionStrip {
		t.Errorf("ref0 = %+v, want index 1 strip", refs[0])
	}
	if refs[1].Index != 3 || refs[1].Action != RefActionDelete {
		t.Errorf("ref1 = %+v, want index 3 delete", refs[1])
	}
}

func TestRawCascadeLogicalNestedKeepsSurvivingSubRule(t *testing.T) {
	rules := rawRules(`{"type":"logical","mode":"and","rules":[{"rule_set":["proxy"]},{"domain":["x.com"]}],"outbound":"x"}`)

	refs := rawReferences(rules, "proxy", RefScopeRoute, false)
	if len(refs) != 1 || refs[0].Action != RefActionStrip {
		t.Fatalf("expected 1 strip ref for logical (a sub-rule survives), got %+v", refs)
	}

	updated, err := scrubRawRules(rules, "proxy", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected logical rule kept, got %d rules", len(updated))
	}
	var logical struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := json.Unmarshal(updated[0], &logical); err != nil {
		t.Fatal(err)
	}
	if len(logical.Rules) != 1 || logical.Rules[0]["rule_set"] != nil {
		t.Errorf("expected only the domain sub-rule to survive, got %+v", logical.Rules)
	}
}

func TestRawCascadeDropsEmptyLogical(t *testing.T) {
	updated, err := scrubRawRules(rawRules(
		`{"type":"logical","mode":"or","rules":[{"rule_set":["proxy"]}],"outbound":"x"}`,
		`{"rule_set":["keep"]}`,
	), "proxy", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected the emptied logical rule dropped, got %s", updated)
	}
	if got := rawStringList(ruleField(t, updated[0], "rule_set")); !reflect.DeepEqual(got, []string{"keep"}) {
		t.Errorf("surviving rule_set = %v, want [keep]", got)
	}
}

func TestRawCascadeStripAndDeleteLeavesOthersIntact(t *testing.T) {
	updated, err := scrubRawRules(rawRules(
		`{"rule_set":["proxy","ai","asia"]}`,
		`{"rule_set":["proxy"]}`,
		`{"domain":["a.com"]}`,
	), "proxy", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 2 {
		t.Fatalf("expected 2 rules after cascade, got %s", updated)
	}
	if got := rawStringList(ruleField(t, updated[0], "rule_set")); !reflect.DeepEqual(got, []string{"ai", "asia"}) {
		t.Errorf("stripped rule_set = %v, want [ai asia]", got)
	}
	if string(updated[1]) != `{"domain":["a.com"]}` {
		t.Errorf("untouched rule was re-encoded: %s", updated[1])
	}
}

func TestRawReferencesDNSScope(t *testing.T) {
	refs := rawReferences(rawRules(
		`{"rule_set":["proxy","ai"],"server":"s"}`,
		`{"rule_set":["proxy"],"server":"s"}`,
	), "proxy", RefScopeDNS, true)
	if len(refs) != 2 || refs[0].Scope != RefScopeDNS || refs[1].Scope != RefScopeDNS {
		t.Fatalf("expected 2 dns refs, got %+v", refs)
	}
	if refs[0].Action != RefActionStrip || refs[1].Action != RefActionDelete {
		t.Errorf("actions = %s/%s, want strip/delete", refs[0].Action, refs[1].Action)
	}
}

func TestRawReferencesDoesNotMutate(t *testing.T) {
	rules := rawRules(`{"rule_set":["proxy","ai"]}`)
	_ = rawReferences(rules, "proxy", RefScopeRoute, false)
	if string(rules[0]) != `{"rule_set":["proxy","ai"]}` {
		t.Errorf("dry run mutated the rule: %s", rules[0])
	}
}

func ruleField(t *testing.T, raw json.RawMessage, key string) json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	return object[key]
}
