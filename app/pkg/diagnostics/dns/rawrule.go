package dnsprobe

// Reading a DNS rule out of raw config JSON, rather than through the pinned
// sing-box option schema.
//
// WHY
// ───
// The panel is compiled against one sing-box version (the repo pins 1.12.12);
// the host runs whatever the operator installed. A 1.14 DNS section uses
// `action: evaluate`, `match_response`, `race` and `optimistic`, none of which
// the pinned decoder knows — and sing-box's decoder is STRICT, so one unknown
// key rejects the whole section. The probe's response was to disable
// attribution entirely, which on a real 1.14 config meant a ladder with zero
// rungs out of twenty-eight.
//
// Raw JSON has no such cliff. A rule is a flat object; the conditions this
// build can decide are decided, and everything else is NAMED as undecidable
// rather than dropped.
//
// THE KEY CLASSIFICATION IS THE WHOLE DESIGN
// ──────────────────────────────────────────
// A DNS rule mixes conditions and action options in one flat object, so a key
// has to be placed in one of three buckets, and the two ways of getting it
// wrong are not symmetric:
//
//   - Mistaking a CONDITION for an action option makes the rule look
//     unconditional. It then "matches" everything and the probe reports a
//     confident, wrong answer. This is unacceptable.
//   - Mistaking an ACTION OPTION for a condition makes the rule undecidable.
//     The probe says so and marks the walk inexact. Noisy, but honest.
//
// So unknown keys fall to the second bucket: anything not recognised is
// treated as a condition that cannot be evaluated. That also means a condition
// added by a future sing-box is handled correctly on the day it ships, with no
// change here — it simply reports as undecidable.

import (
	"encoding/json"
	"strings"
)

// rawDNSRule is one rule, decoded permissively.
type rawDNSRule struct {
	// Type is "logical" for a logical rule, "" or "default" otherwise.
	Type string
	// Mode is "and" or "or", for a logical rule.
	Mode string
	// Children are a logical rule's nested rules.
	Children []rawDNSRule

	Action   string
	Server   string
	Strategy string

	// Conditions this build can decide.
	Domain        []string
	DomainSuffix  []string
	DomainKeyword []string
	DomainRegex   []string
	QueryType     []string
	RuleSet       []string
	Invert        bool

	// Undecidable names every condition key present that this build cannot
	// evaluate, including keys it has never seen. Sorted by appearance order
	// in the known list so the UI reads consistently.
	Undecidable []string
}

// decidableConditionKeys are the conditions this build evaluates itself.
var decidableConditionKeys = map[string]bool{
	"domain": true, "domain_suffix": true, "domain_keyword": true,
	"domain_regex": true, "query_type": true, "rule_set": true, "invert": true,
}

// actionKeys are the non-condition keys: the rule's shape and its action's
// options, flattened alongside the conditions.
//
// Enumerated across 1.12 through 1.14 rather than inferred, because this is
// the list that decides whether an unknown key is treated as a condition.
// A missing entry here costs a spurious "undecidable"; a WRONG entry here
// costs a confidently wrong prediction, so entries are only added from the
// upstream option structs.
var actionKeys = map[string]bool{
	// Rule shape.
	"type": true, "mode": true, "rules": true, "action": true,
	// Route / evaluate / route-options (AbstractDNSRouteActionOptions).
	"server": true, "strategy": true, "disable_cache": true,
	"disable_optimistic_cache": true, "rewrite_ttl": true,
	"client_subnet": true, "remove_client_subnet": true, "timeout": true,
	"speculative": true, "race": true, "tag": true,
	// Reject.
	"method": true, "no_drop": true,
	// Predefined / respond payloads.
	"rcode": true, "answer": true, "ns": true, "extra": true,
	// Legacy spellings kept for older configs.
	"disable_expire": true, "rule_set_ip_cidr_match_source": true,
	"rule_set_ipcidr_match_source": true, "rule_set_ip_cidr_accept_empty": true,
}

// decodeRawDNSRules reads `rules` out of a raw dns section.
//
// Every malformed shape yields no rules rather than an error: this is called
// from an HTTP handler on third-party config text.
func decodeRawDNSRules(section json.RawMessage) []rawDNSRule {
	var object map[string]json.RawMessage
	if json.Unmarshal(section, &object) != nil {
		return nil
	}
	return decodeRawRuleList(object["rules"])
}

func decodeRawRuleList(raw json.RawMessage) []rawDNSRule {
	var list []json.RawMessage
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	rules := make([]rawDNSRule, 0, len(list))
	for _, entry := range list {
		rules = append(rules, decodeRawDNSRule(entry))
	}
	return rules
}

// decodeRawDNSRule classifies every key of one rule object.
//
// An entry that is not an object at all still produces a rule — an
// undecidable one — because dropping it would renumber every rule after it,
// and the index is how the UI, the logs and sing-box itself refer to a rule.
func decodeRawDNSRule(raw json.RawMessage) rawDNSRule {
	rule := rawDNSRule{}

	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		rule.Undecidable = []string{"(unreadable rule)"}
		return rule
	}

	for key, value := range fields {
		switch key {
		case "type":
			rule.Type = rawJSONString(value)
		case "mode":
			rule.Mode = strings.ToLower(rawJSONString(value))
		case "rules":
			rule.Children = decodeRawRuleList(value)
		case "action":
			rule.Action = rawJSONString(value)
		case "server":
			rule.Server = rawJSONString(value)
		case "strategy":
			rule.Strategy = rawJSONString(value)
		case "domain":
			rule.Domain = rawJSONStringList(value)
		case "domain_suffix":
			rule.DomainSuffix = rawJSONStringList(value)
		case "domain_keyword":
			rule.DomainKeyword = rawJSONStringList(value)
		case "domain_regex":
			rule.DomainRegex = rawJSONStringList(value)
		case "rule_set":
			rule.RuleSet = rawJSONStringList(value)
		case "query_type":
			rule.QueryType = rawQueryTypes(value)
		case "invert":
			rule.Invert = rawJSONBool(value)
		default:
			if actionKeys[key] {
				continue
			}
			// Either a known runtime-only condition or one from a sing-box
			// newer than this build. Both are undecidable, and both are named
			// so the UI can say which key blocked the decision.
			rule.Undecidable = append(rule.Undecidable, key)
		}
	}

	sortStrings(rule.Undecidable)
	if rule.Type == "" {
		rule.Type = "default"
	}
	if rule.Action == "" {
		rule.Action = dnsActionRoute
	}
	return rule
}

// hasCondition reports whether the rule constrains anything at all.
//
// A rule with no conditions matches every query (sing-box's
// rule_abstract.go:55 returns true when nothing was tested), so this must not
// be confused with "nothing we could decide".
func (r rawDNSRule) hasCondition() bool {
	if r.Type == "logical" {
		return len(r.Children) > 0
	}
	return len(r.Domain) > 0 || len(r.DomainSuffix) > 0 || len(r.DomainKeyword) > 0 ||
		len(r.DomainRegex) > 0 || len(r.QueryType) > 0 || len(r.RuleSet) > 0 ||
		len(r.Undecidable) > 0
}

// rawJSONString reads a string, or "" for any other shape.
func rawJSONString(raw json.RawMessage) string {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func rawJSONBool(raw json.RawMessage) bool {
	var value bool
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	return value
}

// rawJSONStringList reads a sing-box Listable: a bare string or an array.
// Both spellings are legal and both appear in real configs.
func rawJSONStringList(raw json.RawMessage) []string {
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return list
	}
	var single string
	if json.Unmarshal(raw, &single) == nil && single != "" {
		return []string{single}
	}
	return nil
}

// rawQueryTypes normalises query_type, which sing-box accepts as record NAMES
// ("AAAA") or as numbers (28), in either a list or a bare value.
func rawQueryTypes(raw json.RawMessage) []string {
	var entries []json.RawMessage
	if json.Unmarshal(raw, &entries) != nil {
		entries = []json.RawMessage{raw}
	}

	types := make([]string, 0, len(entries))
	for _, entry := range entries {
		if name := rawJSONString(entry); name != "" {
			types = append(types, strings.ToUpper(name))
			continue
		}
		var number int
		if json.Unmarshal(entry, &number) == nil {
			types = append(types, recordTypeName(number))
		}
	}
	return types
}

// sortStrings is an insertion sort, used on lists of a handful of keys.
// Avoids pulling in sort for one call site in a package that is otherwise
// dependency-light.
func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

