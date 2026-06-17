package config

import (
	"reflect"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// Rule-set reference reconciliation.
//
// A rule_set tag is referenced from three places in config.json: its definition
// in route.rule_set[], and as a matcher inside route.rules[] and dns.rules[]
// (including nested inside logical rules). Deleting only the definition leaves
// dangling matcher references, which sing-box rejects on validation — so the
// delete must scrub those references in the same transaction. This file holds
// the pure scan/scrub logic; the handler runs FindRuleSetReferences for a
// dry-run preview and ApplyRuleSetCascade to mutate before removing the
// definition.

const (
	// RefScopeRoute marks a reference found in route.rules[].
	RefScopeRoute = "route"
	// RefScopeDNS marks a reference found in dns.rules[].
	RefScopeDNS = "dns"

	// RefActionStrip removes just the tag from the rule's rule_set (the rule
	// keeps other matchers).
	RefActionStrip = "strip"
	// RefActionDelete removes the whole rule because the deleted tag was its
	// only matcher — leaving it would match everything.
	RefActionDelete = "delete"
)

// RuleSetRef describes one top-level rule that references a tag and how it will
// change. Index is the position in the ORIGINAL route.rules / dns.rules slice
// (a display hint for the preview; it is not valid after a cascade mutates the
// slice). For logical rules RuleSet is empty (the tag lives in a nested rule).
type RuleSetRef struct {
	Scope   string   `json:"scope"`    // route | dns
	Index   int      `json:"index"`    // pre-mutation position within the rule slice
	Action  string   `json:"action"`   // strip | delete
	RuleSet []string `json:"rule_set"` // the rule_set list before scrubbing
}

// RuleSetExists reports whether a rule-set definition with the given tag is
// declared in route.rule_set[].
func RuleSetExists(cfg *SingBoxConfig, tag string) bool {
	if cfg.Route == nil {
		return false
	}
	for _, ruleSet := range cfg.Route.RuleSet {
		if ruleSet.Tag == tag {
			return true
		}
	}
	return false
}

func listableContains(list badoption.Listable[string], tag string) bool {
	for _, v := range list {
		if v == tag {
			return true
		}
	}
	return false
}

func listableRemove(list badoption.Listable[string], tag string) badoption.Listable[string] {
	out := make(badoption.Listable[string], 0, len(list))
	for _, v := range list {
		if v != tag {
			out = append(out, v)
		}
	}
	return out
}

// routeRuleHasOtherMatcher reports whether the rule still matches on something
// besides rule_set. Action fields and rule_set modifier flags are NOT matchers,
// so a rule left with only an action (and no matcher) counts as "no other
// matcher" → it must be deleted, not stripped. This mirrors sing-box's own
// DefaultRule.IsValid(), exempting the rule_set-only knobs.
func routeRuleHasOtherMatcher(raw option.RawDefaultRule) bool {
	raw.RuleSet = nil
	var empty option.RawDefaultRule
	empty.Invert = raw.Invert
	empty.RuleSetIPCIDRMatchSource = raw.RuleSetIPCIDRMatchSource
	//nolint:staticcheck // SA1019: must mirror the deprecated flag, else a rule that only sets it reads as falsely empty.
	empty.Deprecated_RulesetIPCIDRMatchSource = raw.Deprecated_RulesetIPCIDRMatchSource
	return !reflect.DeepEqual(raw, empty)
}

func dnsRuleHasOtherMatcher(raw option.RawDefaultDNSRule) bool {
	raw.RuleSet = nil
	var empty option.RawDefaultDNSRule
	empty.Invert = raw.Invert
	empty.RuleSetIPCIDRMatchSource = raw.RuleSetIPCIDRMatchSource
	empty.RuleSetIPCIDRAcceptEmpty = raw.RuleSetIPCIDRAcceptEmpty
	//nolint:staticcheck // SA1019: must mirror the deprecated flag, else a rule that only sets it reads as falsely empty.
	empty.Deprecated_RulesetIPCIDRMatchSource = raw.Deprecated_RulesetIPCIDRMatchSource
	return !reflect.DeepEqual(raw, empty)
}

// scrubRouteRule returns a copy of rule with tag removed from any rule_set
// matcher (recursing into logical sub-rules), plus whether the rule should be
// kept. A default rule whose sole matcher was tag, or a logical rule left with
// no sub-rules, is dropped (keep=false). The input is never mutated.
func scrubRouteRule(rule option.Rule, tag string) (option.Rule, bool) {
	if rule.Type == C.RuleTypeLogical {
		subs := rule.LogicalOptions.Rules
		kept := make([]option.Rule, 0, len(subs))
		for _, sub := range subs {
			if s, keep := scrubRouteRule(sub, tag); keep {
				kept = append(kept, s)
			}
		}
		rule.LogicalOptions.Rules = kept
		return rule, len(kept) > 0
	}

	if listableContains(rule.DefaultOptions.RuleSet, tag) {
		rule.DefaultOptions.RuleSet = listableRemove(rule.DefaultOptions.RuleSet, tag)
		if len(rule.DefaultOptions.RuleSet) == 0 && !routeRuleHasOtherMatcher(rule.DefaultOptions.RawDefaultRule) {
			return rule, false
		}
	}
	return rule, true
}

func scrubDNSRule(rule option.DNSRule, tag string) (option.DNSRule, bool) {
	if rule.Type == C.RuleTypeLogical {
		subs := rule.LogicalOptions.Rules
		kept := make([]option.DNSRule, 0, len(subs))
		for _, sub := range subs {
			if s, keep := scrubDNSRule(sub, tag); keep {
				kept = append(kept, s)
			}
		}
		rule.LogicalOptions.Rules = kept
		return rule, len(kept) > 0
	}

	if listableContains(rule.DefaultOptions.RuleSet, tag) {
		rule.DefaultOptions.RuleSet = listableRemove(rule.DefaultOptions.RuleSet, tag)
		if len(rule.DefaultOptions.RuleSet) == 0 && !dnsRuleHasOtherMatcher(rule.DefaultOptions.RawDefaultDNSRule) {
			return rule, false
		}
	}
	return rule, true
}

// FindRuleSetReferences returns, without mutating cfg, every top-level route/dns
// rule that references tag (directly or nested in a logical rule) and whether it
// would be stripped or deleted.
func FindRuleSetReferences(cfg *SingBoxConfig, tag string) []RuleSetRef {
	refs := make([]RuleSetRef, 0)

	if cfg.Route != nil {
		for i, rule := range cfg.Route.Rules {
			scrubbed, keep := scrubRouteRule(rule, tag)
			if keep && reflect.DeepEqual(scrubbed, rule) {
				continue // unchanged ⇒ no reference
			}
			ref := RuleSetRef{Scope: RefScopeRoute, Index: i, Action: RefActionStrip}
			if !keep {
				ref.Action = RefActionDelete
			}
			if rule.Type != C.RuleTypeLogical {
				ref.RuleSet = []string(rule.DefaultOptions.RuleSet)
			}
			refs = append(refs, ref)
		}
	}

	if cfg.DNS != nil {
		for i, rule := range cfg.DNS.Rules {
			scrubbed, keep := scrubDNSRule(rule, tag)
			if keep && reflect.DeepEqual(scrubbed, rule) {
				continue
			}
			ref := RuleSetRef{Scope: RefScopeDNS, Index: i, Action: RefActionStrip}
			if !keep {
				ref.Action = RefActionDelete
			}
			if rule.Type != C.RuleTypeLogical {
				ref.RuleSet = []string(rule.DefaultOptions.RuleSet)
			}
			refs = append(refs, ref)
		}
	}

	return refs
}

// ApplyRuleSetCascade mutates cfg: strips tag from every referencing route/dns
// rule (including nested logical sub-rules) and drops any rule left with no
// matcher. The caller removes the definition from route.rule_set[] afterwards.
func ApplyRuleSetCascade(cfg *SingBoxConfig, tag string) {
	if cfg.Route != nil {
		kept := make([]option.Rule, 0, len(cfg.Route.Rules))
		for _, rule := range cfg.Route.Rules {
			if s, keep := scrubRouteRule(rule, tag); keep {
				kept = append(kept, s)
			}
		}
		cfg.Route.Rules = kept
	}

	if cfg.DNS != nil {
		kept := make([]option.DNSRule, 0, len(cfg.DNS.Rules))
		for _, rule := range cfg.DNS.Rules {
			if s, keep := scrubDNSRule(rule, tag); keep {
				kept = append(kept, s)
			}
		}
		cfg.DNS.Rules = kept
	}
}
