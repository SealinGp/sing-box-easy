package dns

// The per-condition matchers, all operating on a raw rule.
//
// Three of these decide conditions the probe used to throw away, each for a
// different bad reason: `query_type` was discarded even though the probe knew
// the record type it had asked for; `rule_set` was declared runtime-only even
// though this repo has had an offline evaluator for it since the route probe
// was built; and the domain matchers only ever saw a rule the PINNED schema
// could decode, which on a 1.14 config was none of them.

import (
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
)

// matchRawDomainConditions reports whether the rule carries domain conditions
// (present) and whether the query satisfies them (matched).
//
// sing-box treats domain/domain_suffix as one matcher and keyword/regex as
// separate items; a rule matches when every present item matches.
func matchRawDomainConditions(rule rawDNSRule, queryDomain string) (present, matched bool) {
	matched = true

	if len(rule.Domain) > 0 || len(rule.DomainSuffix) > 0 {
		present = true
		if !matchExactOrSuffix(rule, queryDomain) {
			matched = false
		}
	}
	if len(rule.DomainKeyword) > 0 {
		present = true
		if !anyKeyword(rule.DomainKeyword, queryDomain) {
			matched = false
		}
	}
	if len(rule.DomainRegex) > 0 {
		present = true
		if !anyRegex(rule.DomainRegex, queryDomain) {
			matched = false
		}
	}
	return present, matched
}

// matchExactOrSuffix implements sing-box's combined domain matcher: `domain`
// is an exact name and `domain_suffix` matches the name or anything under it.
//
// The suffix rule is not plain string suffixing. sing-box's domain matcher
// treats a value without a leading dot as matching both the name itself and
// its subdomains, so "google.com" matches "google.com" and "www.google.com" —
// but must NOT match "notgoogle.com", which a bare strings.HasSuffix does.
func matchExactOrSuffix(rule rawDNSRule, queryDomain string) bool {
	for _, exact := range rule.Domain {
		if strings.EqualFold(normalizeDomain(exact), queryDomain) {
			return true
		}
	}
	for _, suffix := range rule.DomainSuffix {
		if matchDomainSuffix(normalizeDomain(suffix), queryDomain) {
			return true
		}
	}
	return false
}

func matchDomainSuffix(suffix, queryDomain string) bool {
	if suffix == "" {
		return false
	}
	if strings.HasPrefix(suffix, ".") {
		// An explicit leading dot means subdomains only.
		return strings.HasSuffix(queryDomain, suffix)
	}
	if queryDomain == suffix {
		return true
	}
	return strings.HasSuffix(queryDomain, "."+suffix)
}

// matchRawQueryType decides a rule's `query_type` against the record type
// asked for.
//
// `present` is returned separately from the verdict because "the rule has no
// query_type" and "it has one and it does not match" are opposite facts that a
// three-state verdict cannot both carry: an absent condition must not
// constrain the rule, while a failed one rules it out outright.
//
// The verdict is tri-state because a probe that did not name a type must leave
// the condition open rather than assume A. An AAAA-suppression rule would
// otherwise read as "does not match" on every probe, which is the exact
// opposite of what an IPv6 split does.
func matchRawQueryType(rule rawDNSRule, queryType string) (present bool, verdict ruleset.Verdict) {
	if len(rule.QueryType) == 0 {
		return false, ruleset.VerdictNo
	}
	if queryType == "" {
		return true, ruleset.VerdictUnknown
	}
	for _, candidate := range rule.QueryType {
		if strings.EqualFold(candidate, queryType) {
			return true, ruleset.VerdictYes
		}
	}
	return true, ruleset.VerdictNo
}

// matchRuleSets evaluates a rule's `rule_set` tags.
//
// The tags are OR'd with each other and the result is AND'd with the rest of
// the rule — sing-box builds them into a single RuleSetItem in `items`
// (route/rule/rule_dns.go). Verdicts stay tri-state all the way up: one
// unreadable set makes the rule undecidable, never a miss.
//
// `present` carries the same distinction matchRawQueryType documents.
func matchRuleSets(tags []string, query Query) (present bool, verdict ruleset.Verdict, statuses []RuleSetStatus) {
	if len(tags) == 0 {
		return false, ruleset.VerdictNo, nil
	}

	statuses = make([]RuleSetStatus, 0, len(tags))
	matched, unknown := false, false

	for _, tag := range tags {
		if query.Sets == nil {
			// No loader: the tag is real, we just cannot consult it. Say so
			// rather than omitting the set, so the UI can name what is missing.
			statuses = append(statuses, RuleSetStatus{Tag: tag, State: MatchStateUnevaluated})
			unknown = true
			continue
		}

		result := query.Sets.MatchTarget(tag, ruleset.Target{Domain: query.Domain})
		set := result.Set

		detail := set.Detail
		if result.Detail != "" {
			detail = result.Detail
		}
		status := RuleSetStatus{
			Tag:    tag,
			State:  verdictState(result.Verdict),
			Reason: set.Reason,
			Detail: detail,
			Tier:   result.Tier,
		}
		if !set.UpdatedAt.IsZero() {
			status.UpdatedAtUnix = set.UpdatedAt.Unix()
		}
		statuses = append(statuses, status)

		switch result.Verdict {
		case ruleset.VerdictYes:
			matched = true
		case ruleset.VerdictUnknown:
			unknown = true
		}
	}

	switch {
	// A hit on any tag settles the disjunction, even if a sibling set could
	// not be read: OR only needs one.
	case matched:
		return true, ruleset.VerdictYes, statuses
	case unknown:
		return true, ruleset.VerdictUnknown, statuses
	default:
		return true, ruleset.VerdictNo, statuses
	}
}

// verdictState maps a rule-set verdict onto the state vocabulary the UI's
// ladder already speaks.
func verdictState(verdict ruleset.Verdict) MatchState {
	switch verdict {
	case ruleset.VerdictYes:
		return MatchStateMatched
	case ruleset.VerdictNo:
		return MatchStateNotMatched
	default:
		return MatchStateUnevaluated
	}
}

// summarizeRawRule renders a rule's conditions compactly for the UI.
//
// Undecidable keys are included rather than hidden: a rule shown as
// "(no conditions)" that is in fact gated on `source_hostname` reads as a
// catch-all, which is the opposite of what it is.
func summarizeRawRule(rule rawDNSRule) string {
	var parts []string
	appendList := func(name string, values []string) {
		if len(values) > 0 {
			parts = append(parts, name+"="+joinCapped(values, 3))
		}
	}

	if rule.Type == "logical" {
		mode := rule.Mode
		if mode == "" {
			mode = "and"
		}
		children := make([]string, 0, len(rule.Children))
		for _, child := range rule.Children {
			children = append(children, summarizeRawRule(child))
		}
		if len(children) == 0 {
			return mode + "()"
		}
		return mode + "(" + strings.Join(children, ", ") + ")"
	}

	appendList("domain", rule.Domain)
	appendList("domain_suffix", rule.DomainSuffix)
	appendList("domain_keyword", rule.DomainKeyword)
	appendList("domain_regex", rule.DomainRegex)
	appendList("query_type", rule.QueryType)
	appendList("rule_set", rule.RuleSet)
	for _, key := range rule.Undecidable {
		parts = append(parts, key)
	}
	if rule.Invert {
		parts = append(parts, "invert=true")
	}

	if len(parts) == 0 {
		return "(no conditions)"
	}
	return strings.Join(parts, " ")
}
