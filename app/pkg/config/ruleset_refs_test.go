package config

import (
	"reflect"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// --- fixtures ---

func rRule(ruleSet ...string) option.Rule {
	var r option.Rule
	if len(ruleSet) > 0 {
		r.DefaultOptions.RuleSet = badoption.Listable[string](ruleSet)
	}
	return r
}

func withDomain(r option.Rule, d string) option.Rule {
	r.DefaultOptions.Domain = badoption.Listable[string]{d}
	return r
}

func withInvert(r option.Rule) option.Rule {
	r.DefaultOptions.Invert = true
	return r
}

func withSourcePrivate(r option.Rule) option.Rule {
	r.DefaultOptions.SourceIPIsPrivate = true
	return r
}

func logicalRoute(subs ...option.Rule) option.Rule {
	var r option.Rule
	r.Type = C.RuleTypeLogical
	r.LogicalOptions.Rules = subs
	return r
}

func dRule(ruleSet ...string) option.DNSRule {
	var r option.DNSRule
	if len(ruleSet) > 0 {
		r.DefaultOptions.RuleSet = badoption.Listable[string](ruleSet)
	}
	return r
}

func routeCfg(rules ...option.Rule) *SingBoxConfig {
	cfg := &SingBoxConfig{}
	cfg.Route = &option.RouteOptions{Rules: rules}
	return cfg
}

func ruleSetSlice(r option.Rule) []string { return []string(r.DefaultOptions.RuleSet) }

// --- RuleSetExists ---

func TestRuleSetExists(t *testing.T) {
	cfg := &SingBoxConfig{}
	cfg.Route = &option.RouteOptions{RuleSet: []option.RuleSet{{Tag: "a"}, {Tag: "b"}}}

	if !RuleSetExists(cfg, "a") {
		t.Errorf("expected tag 'a' to exist")
	}
	if RuleSetExists(cfg, "missing") {
		t.Errorf("expected tag 'missing' to not exist")
	}
	if RuleSetExists(&SingBoxConfig{}, "a") {
		t.Errorf("expected false when route is nil")
	}
}

// --- strip vs delete decision (via FindRuleSetReferences) ---

func TestFindRuleSetReferences_Action(t *testing.T) {
	tag := "proxy"
	cases := []struct {
		name       string
		rule       option.Rule
		wantRef    bool
		wantAction string
	}{
		{"sole matcher → delete", rRule(tag), true, RefActionDelete},
		{"with other rule_set → strip", rRule(tag, "other"), true, RefActionStrip},
		{"with domain matcher → strip", withDomain(rRule(tag), "example.com"), true, RefActionStrip},
		{"invert only → delete (invert exempt)", withInvert(rRule(tag)), true, RefActionDelete},
		{"source_ip_is_private → strip (bool matcher)", withSourcePrivate(rRule(tag)), true, RefActionStrip},
		{"unreferenced → no ref", withDomain(rRule(), "example.com"), false, ""},
		{"different tag → no ref", rRule("something-else"), false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs := FindRuleSetReferences(routeCfg(tc.rule), tag)
			if !tc.wantRef {
				if len(refs) != 0 {
					t.Fatalf("expected no references, got %+v", refs)
				}
				return
			}
			if len(refs) != 1 {
				t.Fatalf("expected 1 reference, got %d: %+v", len(refs), refs)
			}
			if refs[0].Action != tc.wantAction {
				t.Errorf("action = %q, want %q", refs[0].Action, tc.wantAction)
			}
			if refs[0].Scope != RefScopeRoute || refs[0].Index != 0 {
				t.Errorf("scope/index = %q/%d, want route/0", refs[0].Scope, refs[0].Index)
			}
		})
	}
}

func TestFindRuleSetReferences_Indices(t *testing.T) {
	tag := "proxy"
	cfg := routeCfg(
		withDomain(rRule(), "a.com"), // 0: unrelated
		rRule(tag, "keep"),           // 1: strip
		rRule("other"),               // 2: unrelated
		rRule(tag),                   // 3: delete
	)
	refs := FindRuleSetReferences(cfg, tag)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].Index != 1 || refs[0].Action != RefActionStrip {
		t.Errorf("ref0 = idx %d %s, want 1 strip", refs[0].Index, refs[0].Action)
	}
	if refs[1].Index != 3 || refs[1].Action != RefActionDelete {
		t.Errorf("ref1 = idx %d %s, want 3 delete", refs[1].Index, refs[1].Action)
	}
}

// --- DNS scope ---

func TestFindRuleSetReferences_DNS(t *testing.T) {
	tag := "proxy"
	cfg := &SingBoxConfig{}
	cfg.DNS = &option.DNSOptions{}
	cfg.DNS.Rules = []option.DNSRule{dRule(tag, "ai"), dRule(tag)}

	refs := FindRuleSetReferences(cfg, tag)
	if len(refs) != 2 {
		t.Fatalf("expected 2 dns refs, got %d", len(refs))
	}
	for _, r := range refs {
		if r.Scope != RefScopeDNS {
			t.Errorf("scope = %q, want dns", r.Scope)
		}
	}
	if refs[0].Action != RefActionStrip || refs[1].Action != RefActionDelete {
		t.Errorf("actions = %s/%s, want strip/delete", refs[0].Action, refs[1].Action)
	}
}

// --- logical rule recursion (the review's HIGH-1) ---

func TestRuleSetReferences_LogicalNested(t *testing.T) {
	tag := "proxy"
	// logical rule whose sub-rule references the tag; another sub-rule survives,
	// so the logical rule is kept and the tag is scrubbed from the nested rule.
	cfg := routeCfg(logicalRoute(rRule(tag), withDomain(rRule(), "x.com")))

	refs := FindRuleSetReferences(cfg, tag)
	if len(refs) != 1 || refs[0].Action != RefActionStrip {
		t.Fatalf("expected 1 strip ref for logical (sub survives), got %+v", refs)
	}

	ApplyRuleSetCascade(cfg, tag)
	if len(cfg.Route.Rules) != 1 {
		t.Fatalf("expected logical rule kept, got %d rules", len(cfg.Route.Rules))
	}
	subs := cfg.Route.Rules[0].LogicalOptions.Rules
	if len(subs) != 1 {
		t.Fatalf("expected 1 surviving sub-rule, got %d", len(subs))
	}
	if reflect.DeepEqual([]string(subs[0].DefaultOptions.RuleSet), []string{tag}) {
		t.Errorf("tag should have been scrubbed from nested sub-rule")
	}
}

func TestApplyRuleSetCascade_DropsEmptyLogical(t *testing.T) {
	tag := "proxy"
	// logical whose only sub-rule is deleted → logical becomes empty → dropped.
	cfg := routeCfg(logicalRoute(rRule(tag)), rRule("keep"))
	ApplyRuleSetCascade(cfg, tag)
	if len(cfg.Route.Rules) != 1 {
		t.Fatalf("expected empty logical dropped, got %d rules", len(cfg.Route.Rules))
	}
	if got := ruleSetSlice(cfg.Route.Rules[0]); !reflect.DeepEqual(got, []string{"keep"}) {
		t.Errorf("surviving rule rule_set = %v, want [keep]", got)
	}
}

// --- cascade application strips/deletes correctly and leaves others intact ---

func TestApplyRuleSetCascade_StripAndDelete(t *testing.T) {
	tag := "proxy"
	cfg := routeCfg(
		rRule(tag, "ai", "asia"),     // strip → [ai, asia]
		rRule(tag),                   // delete
		withDomain(rRule(), "a.com"), // untouched
	)
	ApplyRuleSetCascade(cfg, tag)

	if len(cfg.Route.Rules) != 2 {
		t.Fatalf("expected 2 rules after cascade, got %d", len(cfg.Route.Rules))
	}
	if got := ruleSetSlice(cfg.Route.Rules[0]); !reflect.DeepEqual(got, []string{"ai", "asia"}) {
		t.Errorf("stripped rule_set = %v, want [ai asia]", got)
	}
	if d := []string(cfg.Route.Rules[1].DefaultOptions.Domain); !reflect.DeepEqual(d, []string{"a.com"}) {
		t.Errorf("untouched rule changed: domain = %v", d)
	}
}

// FindRuleSetReferences must not mutate the config (dry-run guarantee).
func TestFindRuleSetReferences_NoMutation(t *testing.T) {
	tag := "proxy"
	cfg := routeCfg(rRule(tag, "ai"))
	_ = FindRuleSetReferences(cfg, tag)
	if got := ruleSetSlice(cfg.Route.Rules[0]); !reflect.DeepEqual(got, []string{tag, "ai"}) {
		t.Errorf("FindRuleSetReferences mutated config: rule_set = %v", got)
	}
}
