package configuration

import (
	"encoding/json"
	"fmt"
)

// Rule-set reference reconciliation.
//
// A rule_set tag is referenced from three places in config.json: its definition
// in route.rule_set[], and as a matcher inside route.rules[] and dns.rules[]
// (including nested inside logical rules). Deleting only the definition leaves
// dangling matcher references, which sing-box rejects on validation — so the
// delete scrubs those references in the same write. The walk is over raw JSON,
// not the pinned option structs, so rules written for a newer core (1.14's
// evaluate/respond) survive the cascade intact.

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

func (h *Service) rawRuleSetReferences(tag string) ([]RuleSetRef, bool, error) {
	route, err := h.readRouteDocument()
	if err != nil {
		return nil, false, err
	}
	exists := false
	for _, ruleSet := range rawList(route, "rule_set") {
		if rawStringField(ruleSet, "tag") == tag {
			exists = true
			break
		}
	}
	if !exists {
		return nil, false, nil
	}
	refs := rawReferences(rawList(route, "rules"), tag, RefScopeRoute, false)
	dns, _, err := h.configManager.GetConfigSection("dns")
	if err != nil {
		return nil, false, err
	}
	dnsObject, err := readRawObjectSection(dns, "dns")
	if err != nil {
		return nil, false, err
	}
	refs = append(refs, rawReferences(rawList(dnsObject, "rules"), tag, RefScopeDNS, true)...)
	return refs, true, nil
}

func rawReferences(rules []json.RawMessage, tag, scope string, dns bool) []RuleSetRef {
	refs := make([]RuleSetRef, 0)
	for index, rule := range rules {
		_, keep, changed, topRuleSets, err := scrubRawRule(rule, tag, dns)
		if err != nil || !changed {
			continue
		}
		action := RefActionStrip
		if !keep {
			action = RefActionDelete
		}
		refs = append(refs, RuleSetRef{Scope: scope, Index: index, Action: action, RuleSet: topRuleSets})
	}
	return refs
}

func scrubRawRules(rules []json.RawMessage, tag string, dns bool) ([]json.RawMessage, error) {
	kept := make([]json.RawMessage, 0, len(rules))
	for _, rule := range rules {
		updated, keep, _, _, err := scrubRawRule(rule, tag, dns)
		if err != nil {
			return nil, err
		}
		if keep {
			kept = append(kept, updated)
		}
	}
	return kept, nil
}

func scrubRawRule(raw json.RawMessage, tag string, dns bool) (json.RawMessage, bool, bool, []string, error) {
	var rule map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rule); err != nil {
		return nil, false, false, nil, fmt.Errorf("failed to parse rule: %w", err)
	}
	var ruleType string
	_ = json.Unmarshal(rule["type"], &ruleType)
	if ruleType == "logical" {
		var nested []json.RawMessage
		if err := json.Unmarshal(rule["rules"], &nested); err != nil {
			return nil, false, false, nil, fmt.Errorf("failed to parse logical rules: %w", err)
		}
		updated, err := scrubRawRules(nested, tag, dns)
		if err != nil {
			return nil, false, false, nil, err
		}
		if len(updated) == len(nested) {
			unchanged := true
			for i := range updated {
				if string(updated[i]) != string(nested[i]) {
					unchanged = false
					break
				}
			}
			if unchanged {
				return raw, true, false, nil, nil
			}
		}
		if len(updated) == 0 {
			return nil, false, true, nil, nil
		}
		rule["rules"], _ = json.Marshal(updated)
		encoded, err := json.Marshal(rule)
		return encoded, true, true, nil, err
	}

	ruleSets := rawStringList(rule["rule_set"])
	if !containsString(ruleSets, tag) {
		return raw, true, false, ruleSets, nil
	}
	remaining := make([]string, 0, len(ruleSets)-1)
	for _, value := range ruleSets {
		if value != tag {
			remaining = append(remaining, value)
		}
	}
	if len(remaining) == 0 {
		delete(rule, "rule_set")
	} else {
		rule["rule_set"], _ = json.Marshal(remaining)
	}
	if len(remaining) == 0 && !rawRuleHasOtherMatcher(rule, dns) {
		return nil, false, true, ruleSets, nil
	}
	encoded, err := json.Marshal(rule)
	return encoded, true, true, ruleSets, err
}

func rawStringList(raw json.RawMessage) []string {
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var scalar string
	if json.Unmarshal(raw, &scalar) == nil && scalar != "" {
		return []string{scalar}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Action/options keys do not make a rule conditional. If rule_set was its only
// matcher, retaining the rule would turn it into an unconditional catch-all.
func rawRuleHasOtherMatcher(rule map[string]json.RawMessage, dns bool) bool {
	nonMatchers := map[string]bool{
		"type": true, "action": true, "invert": true,
		"rule_set_ip_cidr_match_source": true, "rule_set_ip_cidr_accept_empty": true,
		"outbound": true, "server": true, "strategy": true, "disable_cache": true,
		"rewrite_ttl": true, "client_subnet": true, "method": true, "no_drop": true,
		"sniffer": true, "timeout": true, "tag": true,
		"race": true, "answer": true, "ns": true, "extra": true,
		"override_address": true, "override_port": true, "udp_timeout": true,
	}
	if !dns {
		// DNS response selectors are matchers; these keys have no matching
		// meaning in route rules.
		nonMatchers["match_response"] = true
		nonMatchers["response_rcode"] = true
	}
	for key := range rule {
		if !nonMatchers[key] {
			return true
		}
	}
	return false
}
