package trafficflow

import "github.com/SealinGp/sing-box-easy/app/pkg/clashapi"

// RuleIndex maps the rule string a connection carries back to its position in
// `route.rules`.
//
// The mapping is sing-box's own: `GET /rules` returns `router.Rules()` — the
// slice itself, in config order — with `payload = rule.String()` and
// `proxy = rule.Action().String()`, and a connection's `rule` field is
// exactly `payload + " => " + proxy` (trafficontrol/tracker.go). Reproducing
// `String()` for every matcher type here instead would be a second
// implementation that drifts on every sing-box release.
type RuleIndex struct {
	byString map[string]int
}

// BuildRuleIndex indexes a `/rules` listing.
//
// On a duplicate string the FIRST index is kept: sing-box stops at the first
// match, so a later identical rule can never be the one that decided.
func BuildRuleIndex(rules []clashapi.Rule) *RuleIndex {
	byString := make(map[string]int, len(rules))
	for i, rule := range rules {
		key := ruleKey(rule)
		if _, seen := byString[key]; !seen {
			byString[key] = i
		}
	}
	return &RuleIndex{byString: byString}
}

// ruleKey is the connection-side spelling of a rule.
func ruleKey(rule clashapi.Rule) string {
	return rule.Payload + " => " + rule.Proxy
}

// Lookup resolves a connection's rule string. `ok` is false for a string the
// running rule list does not contain — and for FinalRule, which is not a rule.
func (r *RuleIndex) Lookup(rule string) (index int, ok bool) {
	if r == nil {
		return 0, false
	}
	index, ok = r.byString[rule]
	return index, ok
}
