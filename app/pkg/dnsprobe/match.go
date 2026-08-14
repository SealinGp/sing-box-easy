// Package dnsprobe answers "what does this deployment actually do with this
// domain?" — the live answer sing-box returns, which rule produced it, and
// whether the configured upstreams agree with each other.
//
// It deliberately separates fact from prediction. The answer comes from
// sing-box itself; rule attribution is reconstructed here and is explicitly
// marked inexact whenever a condition could not be evaluated offline.
package dnsprobe

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/domain"
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

// Attribute walks the DNS rules in order and reports what would happen to
// `domain`, mirroring sing-box's first-match-wins evaluation.
func Attribute(dns *option.DNSOptions, queryDomain string) Attribution {
	result := Attribution{
		Rules:        []RuleEvaluation{},
		MatchedIndex: -1,
		Exact:        true,
	}
	if dns == nil {
		return result
	}

	normalized := normalizeDomain(queryDomain)
	decided := false

	for i, rule := range dns.Rules {
		evaluation := evaluateRule(i, rule, normalized)
		result.Rules = append(result.Rules, evaluation)

		if decided {
			continue
		}
		switch evaluation.State {
		case MatchStateMatched:
			result.MatchedIndex = i
			result.Server = evaluation.Server
			result.Strategy = evaluation.Strategy
			if evaluation.Strategy != "" {
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
		result.Server = dns.Final
	}
	if result.Strategy == "" {
		result.Strategy = dns.Strategy.String()
		if result.Strategy != "" {
			result.StrategySource = "default"
		}
	}

	return result
}

// evaluateRule decides one rule. Logical rules are not decomposed — they are
// reported as unevaluated rather than guessed at.
func evaluateRule(index int, rule option.DNSRule, queryDomain string) RuleEvaluation {
	switch rule.Type {
	case "", "default":
		return evaluateDefaultRule(index, rule.DefaultOptions, queryDomain)
	default:
		return RuleEvaluation{
			Index:       index,
			Type:        "logical",
			State:       MatchStateUnevaluated,
			Summary:     "logical rule",
			Unevaluated: []string{"logical"},
			Action:      actionName(rule.LogicalOptions.DNSRuleAction),
			Server:      rule.LogicalOptions.RouteOptions.Server,
			Strategy:    rule.LogicalOptions.RouteOptions.Strategy.String(),
		}
	}
}

func evaluateDefaultRule(index int, rule option.DefaultDNSRule, queryDomain string) RuleEvaluation {
	evaluation := RuleEvaluation{
		Index:    index,
		Type:     "default",
		Summary:  summarizeRule(rule),
		Action:   actionName(rule.DNSRuleAction),
		Server:   rule.RouteOptions.Server,
		Strategy: rule.RouteOptions.Strategy.String(),
	}

	// Conditions we cannot decide without sing-box's runtime state. rule_set
	// is by far the most common: the sets are usually remote and live inside
	// sing-box's cache, so their contents are not available here.
	evaluation.Unevaluated = unevaluableConditions(rule)

	// Domain conditions are decidable, and they are the ones that matter for a
	// domain probe. A failure here rules the rule out outright, even when
	// other conditions are unevaluable — sing-box requires all to match.
	domainDecided, domainMatched := matchDomainConditions(rule, queryDomain)
	if domainDecided && !domainMatched {
		evaluation.State = MatchStateNotMatched
		return evaluation
	}

	if len(evaluation.Unevaluated) > 0 {
		evaluation.State = MatchStateUnevaluated
		return evaluation
	}

	if !domainDecided {
		// No domain conditions and nothing unevaluable: the rule matches
		// everything we can see (e.g. a bare `{"server": "..."}`).
		evaluation.State = MatchStateMatched
		return evaluation
	}

	evaluation.State = MatchStateMatched
	// Inverted rules flip the verdict, matching sing-box's `invert`.
	if rule.Invert {
		evaluation.State = MatchStateNotMatched
	}
	return evaluation
}

// matchDomainConditions reports whether the rule carries domain conditions
// (decided) and whether the query satisfies them (matched). sing-box treats
// domain/domain_suffix as one matcher and keyword/regex as separate items;
// a rule matches when every present item matches.
func matchDomainConditions(rule option.DefaultDNSRule, queryDomain string) (decided bool, matched bool) {
	matched = true

	if len(rule.Domain) > 0 || len(rule.DomainSuffix) > 0 {
		decided = true
		// sing-box's own matcher, so exact/suffix semantics cannot drift from
		// the router's behaviour.
		matcher := domain.NewMatcher(rule.Domain, rule.DomainSuffix, false)
		if !matcher.Match(queryDomain) {
			matched = false
		}
	}

	if len(rule.DomainKeyword) > 0 {
		decided = true
		if !anyKeyword(rule.DomainKeyword, queryDomain) {
			matched = false
		}
	}

	if len(rule.DomainRegex) > 0 {
		decided = true
		if !anyRegex(rule.DomainRegex, queryDomain) {
			matched = false
		}
	}

	return decided, matched
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

// unevaluableConditions lists the conditions on a rule that need runtime state
// this process does not have.
func unevaluableConditions(rule option.DefaultDNSRule) []string {
	var fields []string
	add := func(present bool, name string) {
		if present {
			fields = append(fields, name)
		}
	}

	add(len(rule.RuleSet) > 0, "rule_set")
	add(len(rule.Geosite) > 0, "geosite")
	add(len(rule.GeoIP) > 0, "geoip")
	add(len(rule.SourceGeoIP) > 0, "source_geoip")
	add(len(rule.IPCIDR) > 0, "ip_cidr")
	add(rule.IPIsPrivate, "ip_is_private")
	add(rule.IPAcceptAny, "ip_accept_any")
	add(len(rule.SourceIPCIDR) > 0, "source_ip_cidr")
	add(rule.SourceIPIsPrivate, "source_ip_is_private")
	add(len(rule.Inbound) > 0, "inbound")
	add(len(rule.Outbound) > 0, "outbound")
	add(rule.ClashMode != "", "clash_mode")
	add(len(rule.ProcessName) > 0, "process_name")
	add(len(rule.ProcessPath) > 0, "process_path")
	add(len(rule.ProcessPathRegex) > 0, "process_path_regex")
	add(len(rule.PackageName) > 0, "package_name")
	add(len(rule.User) > 0, "user")
	add(len(rule.UserID) > 0, "user_id")
	add(len(rule.AuthUser) > 0, "auth_user")
	add(len(rule.Protocol) > 0, "protocol")
	add(len(rule.Network) > 0, "network")
	add(len(rule.NetworkType) > 0, "network_type")
	add(rule.NetworkIsExpensive, "network_is_expensive")
	add(rule.NetworkIsConstrained, "network_is_constrained")
	add(len(rule.WIFISSID) > 0, "wifi_ssid")
	add(len(rule.WIFIBSSID) > 0, "wifi_bssid")
	add(len(rule.Port) > 0, "port")
	add(len(rule.PortRange) > 0, "port_range")
	add(len(rule.SourcePort) > 0, "source_port")
	add(len(rule.SourcePortRange) > 0, "source_port_range")
	add(rule.IPVersion != 0, "ip_version")
	add(len(rule.QueryType) > 0, "query_type")

	return fields
}

// actionName reports the rule's action, defaulting to "route" which is what
// sing-box assumes when the field is omitted.
func actionName(action option.DNSRuleAction) string {
	if action.Action == "" {
		return "route"
	}
	return action.Action
}

// summarizeRule renders a rule's conditions compactly for the UI.
func summarizeRule(rule option.DefaultDNSRule) string {
	var parts []string
	appendList := func(name string, values []string) {
		if len(values) == 0 {
			return
		}
		parts = append(parts, name+"="+joinCapped(values, 3))
	}

	appendList("domain", rule.Domain)
	appendList("domain_suffix", rule.DomainSuffix)
	appendList("domain_keyword", rule.DomainKeyword)
	appendList("domain_regex", rule.DomainRegex)
	appendList("rule_set", rule.RuleSet)
	appendList("geosite", rule.Geosite)
	appendList("geoip", rule.GeoIP)
	appendList("ip_cidr", rule.IPCIDR)
	appendList("outbound", rule.Outbound)
	if rule.IPAcceptAny {
		parts = append(parts, "ip_accept_any=true")
	}
	if rule.IPIsPrivate {
		parts = append(parts, "ip_is_private=true")
	}
	if rule.ClashMode != "" {
		parts = append(parts, "clash_mode="+rule.ClashMode)
	}
	if rule.Invert {
		parts = append(parts, "invert=true")
	}

	if len(parts) == 0 {
		return "(no conditions)"
	}
	return strings.Join(parts, " ")
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
