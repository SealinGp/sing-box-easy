package dnsprobe

// The two conditions this probe used to discard, decided.
//
// Both were already knowable at the call site. `query_type` is literally the
// record type the probe was asked for, and `rule_set` has had a working
// offline evaluator in this repo since the route probe was built — the DNS
// probe simply never called it. On the config this was written against that
// left almost every rule undecidable, and an attribution that answers
// "cannot tell" for most rules is not an attribution.

import (
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	mkdns "github.com/miekg/dns"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// matchQueryType decides a rule's `query_type` against the record type asked
// for.
//
// `present` is returned separately from the verdict rather than folded into
// it, because "the rule has no query_type" and "the rule has one and it does
// not match" are opposite facts that a three-state Verdict cannot both carry:
// an absent condition must not constrain the rule, while a failed one rules it
// out outright.
//
// The verdict is tri-state for the usual reason: a probe that did not name a
// type must leave the condition open rather than assume A. An AAAA-suppression
// rule would otherwise read as "does not match" on every probe, which is the
// exact opposite of what an IPv6 split does.
func matchQueryType(rule option.DefaultDNSRule, queryType string) (present bool, verdict ruleset.Verdict) {
	if len(rule.QueryType) == 0 {
		return false, ruleset.VerdictNo
	}
	if queryType == "" {
		return true, ruleset.VerdictUnknown
	}

	wanted, ok := mkdns.StringToType[queryType]
	if !ok {
		// A type name this build does not know cannot be compared to the
		// numeric forms in the config. Undecidable, not a miss.
		return true, ruleset.VerdictUnknown
	}

	for _, candidate := range rule.QueryType {
		if uint16(candidate) == wanted {
			return true, ruleset.VerdictYes
		}
	}
	return true, ruleset.VerdictNo
}

// matchRuleSets evaluates a DNS rule's `rule_set` tags.
//
// The tags are OR'd with each other and the result is AND'd with the rest of
// the rule — sing-box builds them into a single RuleSetItem in `items`
// (route/rule/rule_dns.go:250-262), which is the same shape the route probe
// mirrors. Verdicts stay tri-state all the way up: one unreadable set makes
// the rule undecidable, never a miss.
// `present` carries the same distinction matchQueryType documents: a rule with
// no rule_set is unconstrained by one, not failed by one.
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

// queryTypeNames renders configured query types as their text form.
//
// option.DNSQueryType is a uint16 with a custom unmarshaller that accepts both
// "AAAA" and 28, so the decoded value is always numeric — and a summary
// reading "query_type=28" is a lookup task, not a label.
func queryTypeNames(types badoption.Listable[option.DNSQueryType]) []string {
	names := make([]string, 0, len(types))
	for _, qType := range types {
		if name, ok := mkdns.TypeToString[uint16(qType)]; ok {
			names = append(names, name)
			continue
		}
		// A type this build has no name for still has to appear: omitting it
		// would render a two-type rule as a one-type rule.
		names = append(names, "TYPE"+strconv.Itoa(int(qType)))
	}
	return names
}
