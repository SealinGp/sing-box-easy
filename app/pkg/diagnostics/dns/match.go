// Package dnsprobe answers "what does this deployment actually do with this
// domain?" — the live answer sing-box returns, which rule produced it, and
// whether the configured upstreams agree with each other.
//
// It deliberately separates fact from prediction. The answer comes from
// sing-box itself; rule attribution is reconstructed here and is explicitly
// marked inexact whenever a condition could not be evaluated offline.
package dnsprobe

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
)

// MatchState is the verdict for one DNS rule against one domain.
type MatchState string

const (
	// MatchStateMatched means every condition present on the rule was
	// evaluated and all of them matched.
	MatchStateMatched MatchState = "matched"
	// MatchStateNotMatched means at least one evaluated condition failed, so
	// the rule cannot match regardless of what we could not evaluate.
	MatchStateNotMatched MatchState = "not_matched"
	// MatchStateUnevaluated means the rule carries at least one condition we
	// cannot decide without sing-box's runtime state (rule_set contents,
	// client IP, process, inbound, …) and nothing else ruled it out.
	MatchStateUnevaluated MatchState = "unevaluated"
)

// RuleEvaluation is the verdict for a single rule, in config order.
type RuleEvaluation struct {
	Index int        `json:"index"`
	Type  string     `json:"type"`
	State MatchState `json:"state"`
	// Summary renders the rule's conditions for display, e.g.
	// "domain_suffix=[owolist.cn liangxin.xyz]".
	Summary string `json:"summary"`
	// Unevaluated names the conditions that could not be decided offline.
	Unevaluated []string `json:"unevaluated,omitempty"`
	// Action is the rule's action ("route" when unset), and Server/Strategy
	// are the routing target when the action is a route.
	Action   string `json:"action"`
	Server   string `json:"server,omitempty"`
	Strategy string `json:"strategy,omitempty"`
	// RuleSets carries per-set detail when the rule references any, so a rule
	// that could not be decided names the set responsible rather than saying
	// only "rule_set". Same shape the route probe reports.
	RuleSets []RuleSetStatus `json:"rule_sets,omitempty"`
	// Terminal reports whether a match on this rule ENDS the walk.
	//
	// Not decoration, and not derivable from the action name: sing-box's
	// matchDNS switch returns only for route, reject and predefined, so
	// `evaluate` and `route-options` match and hand over. A ladder that showed
	// every match as the decision would name the wrong server on any config
	// using them — which, on 1.14, is most of them.
	Terminal bool `json:"terminal"`
	// Effect describes what a non-terminal match changed for the rules below.
	Effect string `json:"effect,omitempty"`
}

// RuleSetStatus reports one rule set referenced by a DNS rule.
//
// Deliberately the same shape routeprobe.RuleSetStatus uses: the two probes
// answer the same kind of question and the UI renders them with one component,
// so two near-identical payloads would only invite them to drift.
type RuleSetStatus struct {
	Tag string `json:"tag"`
	// State is the set's own verdict: matched, not_matched or unevaluated.
	State MatchState `json:"state"`
	// Reason is populated when the set could not be read at all.
	Reason ruleset.Reason `json:"reason,omitempty"`
	Detail string         `json:"detail,omitempty"`
	// UpdatedAtUnix is when sing-box last downloaded the set, 0 when unknown.
	UpdatedAtUnix int64 `json:"updated_at_unix,omitempty"`
	// Tier says which layer decided it: decoded here, or answered by the
	// installed sing-box binary.
	Tier ruleset.Tier `json:"tier,omitempty"`
}

// Query is everything known about the lookup being attributed.
//
// It replaces the bare domain string because two of the most decisive
// conditions in a real config were being discarded: the record type (an
// AAAA-suppression rule is the basis of every IPv6 split) and the rule sets
// (on the config this was built against, most rules carry nothing else).
// Both were already available at the call site and simply never passed down.
type Query struct {
	// Domain is the name being looked up; it is normalized here.
	Domain string
	// Type is the record type asked for ("A", "AAAA", …). Empty means the
	// caller did not say, which leaves query_type rules undecidable rather
	// than assuming one.
	Type string
	// Sets resolves rule_set tags. Optional: without it, every rule carrying
	// a rule_set stays undecidable, which is what this probe did before.
	Sets *ruleset.Loader
}

// Attribution is the reconstructed routing decision for a domain.
type Attribution struct {
	Rules []RuleEvaluation `json:"rules"`
	// MatchedIndex is the first matching rule, or -1 when the query falls
	// through to dns.final.
	MatchedIndex int `json:"matched_index"`
	// Server is the DNS server tag the query is predicted to use, and
	// Strategy the effective domain strategy.
	Server   string `json:"server"`
	Strategy string `json:"strategy"`
	// StrategySource says which config key supplied Strategy: "rule" when the
	// matched rule set its own, "default" when it came from dns.strategy.
	// Without this the UI cannot name the key a value came from, which is the
	// difference between showing a setting and explaining it.
	StrategySource string `json:"strategy_source,omitempty"`
	// FinalUsed reports that no rule matched and dns.final applies.
	FinalUsed bool `json:"final_used"`
	// Exact is true only when the verdict cannot be wrong: every rule ahead
	// of the decision was fully evaluated. A single unevaluated rule earlier
	// in the list could have matched first, so the prediction below it is a
	// best guess and must be presented as one.
	Exact bool `json:"exact"`
	// UnevaluatedBefore counts unevaluated rules ahead of the decision — the
	// reason Exact is false.
	UnevaluatedBefore int `json:"unevaluated_before"`
}

// NOTE ON THE ENTRY POINT
// ───────────────────────
// There is deliberately no `Attribute(*option.DNSOptions, ...)` wrapper. One
// was written and removed: re-encoding typed options to hand them to this walk
// is LOSSY in upstream sing-box. `DNSRuleAction.MarshalJSON` emits the route
// options only when `Action` is set explicitly, and `Action` is empty for
// every rule that relies on the default — the most common spelling in real
// configs. A rule decoded as {"domain":"x","server":"s"} marshals back as
// {"domain":"x"}, silently losing the server.
//
// So the raw section is the only input. That is also the shape a config is
// stored and read in, which means nothing has to round-trip at all.

// AttributeRaw walks a DNS section given as raw config JSON.
//
// Raw rather than typed because the panel is compiled against one sing-box
// version and the host runs another. sing-box's decoder is strict, so a single
// 1.14 key — `evaluate`, `match_response`, `optimistic` — rejects the entire
// section, and the probe's previous response was to disable attribution
// outright: a ladder with zero rungs on a config with twenty-eight rules.
//
// The walk is a STATE MACHINE, not a filter. Only route, reject and predefined
// end it (dns/router.go); everything else matches, may change what the rules
// below it see, and hands over.
func AttributeRaw(section json.RawMessage, query Query) Attribution {
	result := Attribution{
		Rules:        []RuleEvaluation{},
		MatchedIndex: -1,
		Exact:        true,
	}

	rules := decodeRawDNSRules(section)
	serverExists := rawServerLookup(section)

	query.Domain = normalizeDomain(query.Domain)
	query.Type = strings.ToUpper(strings.TrimSpace(query.Type))

	decided := false
	// A non-terminal match may set the strategy for the rules below it
	// (dns/router.go applies action.Strategy before continuing), so the
	// deciding rule inherits it when it names none of its own.
	pendingStrategy := ""

	for i, rule := range rules {
		evaluation := evaluateRawRule(i, rule, query, serverExists)
		result.Rules = append(result.Rules, evaluation)

		if decided {
			continue
		}
		switch evaluation.State {
		case MatchStateMatched:
			if !evaluation.Terminal {
				if evaluation.Strategy != "" {
					pendingStrategy = evaluation.Strategy
				}
				continue
			}
			result.MatchedIndex = i
			result.Server = evaluation.Server
			result.Strategy = evaluation.Strategy
			if evaluation.Strategy != "" {
				result.StrategySource = "rule"
			} else if pendingStrategy != "" {
				result.Strategy = pendingStrategy
				result.StrategySource = "rule"
			}
			decided = true
		case MatchStateUnevaluated:
			// Cannot rule it out, so everything after this point is uncertain.
			result.UnevaluatedBefore++
			result.Exact = false
		}
	}

	if !decided {
		result.FinalUsed = true
		result.Server = rawSectionString(section, "final")
		if pendingStrategy != "" {
			result.Strategy = pendingStrategy
			result.StrategySource = "rule"
		}
	}
	if result.Strategy == "" {
		result.Strategy = rawSectionString(section, "strategy")
		if result.Strategy != "" {
			result.StrategySource = "default"
		}
	}

	return result
}

// evaluateRawRule decides one rule, logical or default.
func evaluateRawRule(
	index int, rule rawDNSRule, query Query, serverExists func(string) bool,
) RuleEvaluation {
	evaluation := RuleEvaluation{
		Index:    index,
		Type:     rule.Type,
		Summary:  summarizeRawRule(rule),
		Action:   rule.Action,
		Server:   rule.Server,
		Strategy: rule.Strategy,
		Terminal: dnsActionTerminates(rule.Action, rule.Server, serverExists),
	}
	if !evaluation.Terminal {
		evaluation.Effect = dnsActionEffect(rule.Action)
	}

	state, sets := matchRawRule(rule, query)
	evaluation.State = state
	evaluation.RuleSets = sets
	if state == MatchStateUnevaluated {
		evaluation.Unevaluated = undecidableNames(rule, query)
	}
	return evaluation
}

// matchRawRule is the recursive matcher: logical rules combine their children,
// default rules test their own conditions.
func matchRawRule(rule rawDNSRule, query Query) (MatchState, []RuleSetStatus) {
	if rule.Type == "logical" {
		return matchLogicalRule(rule, query)
	}
	return matchDefaultRule(rule, query)
}

// matchLogicalRule combines child verdicts.
//
// The short-circuits are the point, and they are not symmetric: an AND with
// one decided miss is a decided miss even if a sibling is undecidable, and an
// OR with one decided hit is a decided hit. Collapsing either into
// "unevaluated" would make every logical rule undecidable and defeat walking
// into them at all — which is what the previous implementation did.
func matchLogicalRule(rule rawDNSRule, query Query) (MatchState, []RuleSetStatus) {
	var sets []RuleSetStatus
	if len(rule.Children) == 0 {
		return applyInvert(MatchStateMatched, rule.Invert), sets
	}

	isAnd := rule.Mode != "or"
	sawUnevaluated := false
	sawMatch := false

	for _, child := range rule.Children {
		state, childSets := matchRawRule(child, query)
		sets = append(sets, childSets...)

		switch state {
		case MatchStateNotMatched:
			if isAnd {
				return applyInvert(MatchStateNotMatched, rule.Invert), sets
			}
		case MatchStateMatched:
			sawMatch = true
			if !isAnd {
				return applyInvert(MatchStateMatched, rule.Invert), sets
			}
		case MatchStateUnevaluated:
			sawUnevaluated = true
		}
	}

	if sawUnevaluated {
		return MatchStateUnevaluated, sets
	}
	if isAnd {
		return applyInvert(MatchStateMatched, rule.Invert), sets
	}
	if sawMatch {
		return applyInvert(MatchStateMatched, rule.Invert), sets
	}
	return applyInvert(MatchStateNotMatched, rule.Invert), sets
}

// matchDefaultRule tests one rule's own conditions.
//
// Every condition is an AND item, so a single decided miss rules the rule out
// even when something else could not be evaluated. That asymmetry is why each
// condition is checked for a definite NO before the undecidable list matters.
func matchDefaultRule(rule rawDNSRule, query Query) (MatchState, []RuleSetStatus) {
	domainPresent, domainMatched := matchRawDomainConditions(rule, query.Domain)
	if domainPresent && !domainMatched {
		return applyInvert(MatchStateNotMatched, rule.Invert), nil
	}

	typePresent, typeVerdict := matchRawQueryType(rule, query.Type)
	if typePresent && typeVerdict == ruleset.VerdictNo {
		return applyInvert(MatchStateNotMatched, rule.Invert), nil
	}

	setsPresent, setVerdict, sets := matchRuleSets(rule.RuleSet, query)
	if setsPresent && setVerdict == ruleset.VerdictNo {
		return applyInvert(MatchStateNotMatched, rule.Invert), sets
	}

	// Nothing decided a miss. Anything we could not evaluate now governs.
	if len(rule.Undecidable) > 0 ||
		(typePresent && typeVerdict == ruleset.VerdictUnknown) ||
		(setsPresent && setVerdict == ruleset.VerdictUnknown) {
		return MatchStateUnevaluated, sets
	}

	// A rule with no conditions at all matches every query, which sing-box
	// does explicitly (rule_abstract.go:55). It must not be confused with a
	// rule whose conditions we merely failed to read.
	return applyInvert(MatchStateMatched, rule.Invert), sets
}

// applyInvert flips a decided verdict. An undecidable one stays undecidable —
// the negation of "I do not know" is still "I do not know".
func applyInvert(state MatchState, invert bool) MatchState {
	if !invert {
		return state
	}
	switch state {
	case MatchStateMatched:
		return MatchStateNotMatched
	case MatchStateNotMatched:
		return MatchStateMatched
	}
	return state
}

// undecidableNames lists what blocked a decision, for the UI.
func undecidableNames(rule rawDNSRule, query Query) []string {
	names := append([]string{}, rule.Undecidable...)
	if len(rule.QueryType) > 0 && query.Type == "" {
		names = append(names, "query_type")
	}
	if len(rule.RuleSet) > 0 {
		names = append(names, "rule_set")
	}
	for _, child := range rule.Children {
		names = append(names, undecidableNames(child, query)...)
	}
	return dedupeStrings(names)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := values[:0]
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// rawServerLookup builds a membership test over the configured server tags, so
// a route naming a server that does not exist can be reported as what
// sing-box does with it: skip the rule and keep matching.
func rawServerLookup(section json.RawMessage) func(string) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(section, &object) != nil {
		return nil
	}
	raw, ok := object["servers"]
	if !ok {
		return nil
	}
	var servers []map[string]json.RawMessage
	if json.Unmarshal(raw, &servers) != nil {
		return nil
	}
	tags := make(map[string]struct{}, len(servers))
	for _, server := range servers {
		var tag string
		if json.Unmarshal(server["tag"], &tag) == nil && tag != "" {
			tags[tag] = struct{}{}
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return func(tag string) bool {
		_, ok := tags[tag]
		return ok
	}
}

// rawSectionString reads a top-level string field of the dns section.
func rawSectionString(section json.RawMessage, key string) string {
	var object map[string]json.RawMessage
	if json.Unmarshal(section, &object) != nil {
		return ""
	}
	return rawJSONString(object[key])
}


func anyKeyword(keywords []string, queryDomain string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(queryDomain, keyword) {
			return true
		}
	}
	return false
}

// anyRegex ignores patterns that fail to compile: sing-box would have rejected
// the config, so an invalid pattern here means the config is not running as
// written and guessing would be worse than skipping.
func anyRegex(patterns []string, queryDomain string) bool {
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if compiled.MatchString(queryDomain) {
			return true
		}
	}
	return false
}




// joinCapped renders at most `limit` values, marking the rest with an ellipsis
// so a 40-entry rule_set does not flood the UI.
func joinCapped(values []string, limit int) string {
	if len(values) == 1 {
		return values[0]
	}
	if len(values) > limit {
		return fmt.Sprintf("[%s ...+%d]", strings.Join(values[:limit], " "), len(values)-limit)
	}
	return "[" + strings.Join(values, " ") + "]"
}

// normalizeDomain lowercases and strips the trailing dot so "Example.COM." and
// "example.com" are treated as the same name.
func normalizeDomain(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}
