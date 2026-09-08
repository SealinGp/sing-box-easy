package configuration

import (
	"encoding/json"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func (h *Service) rawRuleSetReferences(tag string) ([]config.RuleSetRef, bool, error) {
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
	refs := rawReferences(rawList(route, "rules"), tag, config.RefScopeRoute, false)
	dns, _, err := h.configManager.GetConfigSection("dns")
	if err != nil {
		return nil, false, err
	}
	dnsObject, err := readRawObjectSection(dns, "dns")
	if err != nil {
		return nil, false, err
	}
	refs = append(refs, rawReferences(rawList(dnsObject, "rules"), tag, config.RefScopeDNS, true)...)
	return refs, true, nil
}

func rawReferences(rules []json.RawMessage, tag, scope string, dns bool) []config.RuleSetRef {
	refs := make([]config.RuleSetRef, 0)
	for index, rule := range rules {
		_, keep, changed, topRuleSets, err := scrubRawRule(rule, tag, dns)
		if err != nil || !changed {
			continue
		}
		action := config.RefActionStrip
		if !keep {
			action = config.RefActionDelete
		}
		refs = append(refs, config.RuleSetRef{Scope: scope, Index: index, Action: action, RuleSet: topRuleSets})
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
