package routeprobe

import (
	"net/netip"
	"regexp"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/ruleset"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// evaluator walks the route rules for one target.
//
// The walk is a state machine, not a filter. Non-terminal actions mutate the
// context the rules below them see: `sniff` supplies the domain that later
// domain rules match on, and `resolve` replaces the destination with an
// address, which is what lets an ip_cidr rule fire for what the user typed as a
// name (route/route.go:472, 543-547). Evaluating each rule against the original
// input in isolation gets those configs wrong.
type evaluator struct {
	loader    *ruleset.Loader
	inbound   string
	clashMode string
	protocol  string
	target    ruleset.Target
}

// walk evaluates every rule in order and records the decision.
func (e *evaluator) walk(options *option.Options, result *Result) {
	var rules []option.Rule
	if options != nil && options.Route != nil {
		rules = options.Route.Rules
	}

	decided := false
	warnings := map[string]struct{}{}

	for index, rule := range rules {
		if decided {
			// Still evaluated for display: "which rule stole my traffic" is
			// answered by seeing the shadowed rule sitting below the winner.
			result.Rules = append(result.Rules, RuleEvaluation{
				Index:    index,
				Type:     ruleType(rule),
				State:    MatchStateSkipped,
				Summary:  summarize(rule),
				Action:   actionName(rule),
				Outcome:  actionOutcome(ruleAction(rule)),
				Terminal: isTerminal(actionName(rule)),
			})
			continue
		}

		evaluation := e.evaluate(index, rule)
		result.Rules = append(result.Rules, evaluation)

		for _, set := range evaluation.RuleSets {
			if set.Reason != ruleset.ReasonOK {
				warnings[string(set.Reason)] = struct{}{}
			}
		}

		switch evaluation.State {
		case MatchStateMatched:
			if evaluation.Terminal {
				result.MatchedIndex = index
				result.Action = evaluation.Action
				result.Outbound = evaluation.Outcome
				result.OutboundSource = "rule"
				decided = true
				continue
			}
			// Non-terminal: apply what it changes and keep going.
			e.apply(rule, &result.Rules[len(result.Rules)-1])

		case MatchStateUnevaluated:
			// Cannot be ruled out, so nothing below it is certain.
			result.UnevaluatedBefore++
			result.Exact = false
		}
	}

	if !decided {
		result.FinalUsed = true
		result.Action = C.RuleActionTypeRoute
		result.Outbound, result.OutboundSource = defaultOutbound(options)
	}

	for warning := range warnings {
		result.Warnings = append(result.Warnings, warning)
	}
	sortStrings(result.Warnings)
}

// apply mirrors the metadata a non-terminal action rewrites for later rules.
func (e *evaluator) apply(rule option.Rule, evaluation *RuleEvaluation) {
	switch actionName(rule) {
	case C.RuleActionTypeResolve:
		// sing-box replaces the destination with the resolved address here.
		// The probe already resolved up front, so the practical effect is to
		// record that address rules below this point are decidable.
		if e.target.IP.IsValid() {
			evaluation.Effect = "destination_resolved"
		} else {
			// Without an address the rules below cannot be decided, and
			// pretending otherwise would produce a confident wrong answer.
			evaluation.Effect = "destination_resolve_failed"
		}
	case C.RuleActionTypeSniff:
		// Sniffing is what supplies the domain for a connection opened to a
		// bare address. A probe that was given a name already has it.
		if e.target.Domain != "" {
			evaluation.Effect = "domain_known"
		} else {
			evaluation.Effect = "domain_would_be_sniffed"
		}
	}
}

// evaluate decides one rule.
func (e *evaluator) evaluate(index int, rule option.Rule) RuleEvaluation {
	evaluation := RuleEvaluation{
		Index:    index,
		Type:     ruleType(rule),
		Summary:  summarize(rule),
		Action:   actionName(rule),
		Outcome:  actionOutcome(ruleAction(rule)),
		Terminal: isTerminal(actionName(rule)),
	}

	switch rule.Type {
	case "", C.RuleTypeDefault:
		verdict, unevaluated, sets := e.matchDefault(rule.DefaultOptions)
		evaluation.State = toState(verdict)
		evaluation.Unevaluated = unevaluated
		evaluation.RuleSets = sets
	case C.RuleTypeLogical:
		verdict, unevaluated, sets := e.matchLogical(rule.LogicalOptions)
		evaluation.State = toState(verdict)
		evaluation.Unevaluated = unevaluated
		evaluation.RuleSets = sets
	default:
		evaluation.State = MatchStateUnevaluated
		evaluation.Unevaluated = []string{"unknown_rule_type"}
	}

	return evaluation
}

func (e *evaluator) matchLogical(rule option.LogicalRule) (ruleset.Verdict, []string, []RuleSetStatus) {
	and := strings.EqualFold(rule.Mode, C.LogicalTypeAnd)

	var (
		unevaluated []string
		sets        []RuleSetStatus
		unknown     bool
	)

	for _, sub := range rule.Rules {
		var verdict ruleset.Verdict
		var subUnevaluated []string
		var subSets []RuleSetStatus

		switch sub.Type {
		case "", C.RuleTypeDefault:
			verdict, subUnevaluated, subSets = e.matchDefault(sub.DefaultOptions)
		case C.RuleTypeLogical:
			verdict, subUnevaluated, subSets = e.matchLogical(sub.LogicalOptions)
		default:
			verdict = ruleset.VerdictUnknown
			subUnevaluated = []string{"unknown_rule_type"}
		}

		unevaluated = append(unevaluated, subUnevaluated...)
		sets = append(sets, subSets...)

		switch verdict {
		case ruleset.VerdictUnknown:
			unknown = true
		case ruleset.VerdictYes:
			if !and {
				return flip(ruleset.VerdictYes, rule.Invert), unevaluated, sets
			}
		case ruleset.VerdictNo:
			if and {
				return flip(ruleset.VerdictNo, rule.Invert), unevaluated, sets
			}
		}
	}

	if unknown {
		return ruleset.VerdictUnknown, unevaluated, sets
	}
	if and {
		return flip(ruleset.VerdictYes, rule.Invert), unevaluated, sets
	}
	return flip(ruleset.VerdictNo, rule.Invert), unevaluated, sets
}

// matchDefault mirrors route/rule.abstractDefaultRule.Match for a route rule.
//
// The grouping is the same as for headless rules, and just as easy to get
// wrong. Note in particular where each route-only condition lands
// (route/rule/rule_default.go):
//
//   - `rule_set` is an AND item, and the tags WITHIN one rule are OR'd
//     (rule_item_rule_set.go:43-52).
//   - `ip_is_private` joins the destination-address group, so it is OR'd with
//     the domain matchers rather than AND'd with them.
//   - `geosite` is in the destination-address group too, but is a deprecated
//     geo-database lookup this panel cannot perform, so it is unevaluable.
func (e *evaluator) matchDefault(rule option.DefaultRule) (ruleset.Verdict, []string, []RuleSetStatus) {
	var (
		unevaluated []string
		sets        []RuleSetStatus
	)
	raw := rule.RawDefaultRule

	unknown := func() (ruleset.Verdict, []string, []RuleSetStatus) {
		return ruleset.VerdictUnknown, unevaluated, sets
	}
	no := func() (ruleset.Verdict, []string, []RuleSetStatus) {
		return flip(ruleset.VerdictNo, raw.Invert), unevaluated, sets
	}

	// --- AND items -----------------------------------------------------
	if len(raw.Inbound) > 0 {
		if e.inbound == "" {
			unevaluated = append(unevaluated, "inbound")
		} else if !contains(raw.Inbound, e.inbound) {
			return no()
		}
	}

	if len(raw.Network) > 0 && !containsFold(raw.Network, e.target.Network) {
		return no()
	}

	if raw.IPVersion > 0 {
		if !e.target.IP.IsValid() {
			unevaluated = append(unevaluated, "ip_version")
		} else {
			is6 := e.target.IP.Unmap().Is6()
			if (raw.IPVersion == 6) != is6 {
				return no()
			}
		}
	}

	if raw.ClashMode != "" {
		if e.clashMode == "" {
			unevaluated = append(unevaluated, "clash_mode")
		} else if !strings.EqualFold(raw.ClashMode, e.clashMode) {
			return no()
		}
	}

	// The sniffed application protocol. Undecidable unless the caller says
	// what it would be — sniffing needs bytes on the wire, and there are none.
	if len(raw.Protocol) > 0 {
		if e.protocol == "" {
			unevaluated = append(unevaluated, "protocol")
		} else if !containsFold(raw.Protocol, e.protocol) {
			return no()
		}
	}

	// Conditions that need state only the client's machine has.
	for _, condition := range runtimeOnlyConditions(raw) {
		unevaluated = append(unevaluated, condition)
	}

	if len(raw.RuleSet) > 0 {
		verdict, statuses := e.matchRuleSets(raw.RuleSet)
		sets = append(sets, statuses...)
		switch verdict {
		case ruleset.VerdictNo:
			return no()
		case ruleset.VerdictUnknown:
			unevaluated = append(unevaluated, "rule_set")
		}
	}

	// --- destination address group (domain* OR ip_cidr OR ip_is_private) ---
	verdict := e.matchDestination(raw, &unevaluated)
	switch verdict {
	case ruleset.VerdictNo:
		return no()
	case ruleset.VerdictUnknown:
		return unknown()
	}

	// --- ports ---------------------------------------------------------
	if len(raw.Port) > 0 || len(raw.PortRange) > 0 {
		if !matchPortLists(e.target.Port, raw.Port, raw.PortRange) {
			return no()
		}
	}
	if len(raw.SourcePort) > 0 || len(raw.SourcePortRange) > 0 {
		if e.target.SourcePort == 0 {
			unevaluated = append(unevaluated, "source_port")
		} else if !matchPortLists(e.target.SourcePort, raw.SourcePort, raw.SourcePortRange) {
			return no()
		}
	}

	// --- source address group ------------------------------------------
	if len(raw.SourceIPCIDR) > 0 || raw.SourceIPIsPrivate {
		if !e.target.SourceIP.IsValid() {
			unevaluated = append(unevaluated, "source_ip_cidr")
		} else {
			matched := false
			if raw.SourceIPIsPrivate && e.target.SourceIP.IsPrivate() {
				matched = true
			}
			if !matched && len(raw.SourceIPCIDR) > 0 && inPrefixes(raw.SourceIPCIDR, e.target.SourceIP) {
				matched = true
			}
			if !matched {
				return no()
			}
		}
	}

	if len(unevaluated) > 0 {
		return unknown()
	}
	return flip(ruleset.VerdictYes, raw.Invert), unevaluated, sets
}

// matchDestination evaluates the destination-address group: the domain
// matchers, ip_cidr and ip_is_private are OR'd with one another because they
// all write the same DestinationAddressMatch flag upstream.
func (e *evaluator) matchDestination(raw option.RawDefaultRule, unevaluated *[]string) ruleset.Verdict {
	hasDomain := len(raw.Domain) > 0 || len(raw.DomainSuffix) > 0 ||
		len(raw.DomainKeyword) > 0 || len(raw.DomainRegex) > 0
	hasGeosite := len(raw.Geosite) > 0
	hasIP := len(raw.IPCIDR) > 0 || raw.IPIsPrivate
	hasGeoIP := len(raw.GeoIP) > 0

	if !hasDomain && !hasGeosite && !hasIP && !hasGeoIP {
		return ruleset.VerdictYes
	}

	undecided := false

	if hasDomain {
		if e.target.Domain == "" {
			undecided = true
		} else if matchDomain(raw, e.target.Domain) {
			return ruleset.VerdictYes
		}
	}
	// geoip/geosite are the pre-rule-set geo database, which this panel does
	// not carry. They are deprecated upstream but still appear in older
	// configs, and guessing at them would be worse than admitting the gap.
	if hasGeosite {
		*unevaluated = appendOnce(*unevaluated, "geosite")
		undecided = true
	}
	if hasGeoIP {
		*unevaluated = appendOnce(*unevaluated, "geoip")
		undecided = true
	}

	if hasIP {
		if !e.target.IP.IsValid() {
			undecided = true
		} else {
			if raw.IPIsPrivate && e.target.IP.IsPrivate() {
				return ruleset.VerdictYes
			}
			if len(raw.IPCIDR) > 0 && inPrefixes(raw.IPCIDR, e.target.IP) {
				return ruleset.VerdictYes
			}
		}
	}

	if undecided {
		return ruleset.VerdictUnknown
	}
	return ruleset.VerdictNo
}

// matchRuleSets evaluates the tags on one rule as a disjunction.
func (e *evaluator) matchRuleSets(tags []string) (ruleset.Verdict, []RuleSetStatus) {
	statuses := make([]RuleSetStatus, 0, len(tags))
	unknown := false
	matched := false

	for _, tag := range tags {
		set := e.loader.Get(tag)
		verdict := set.Match(e.target)

		status := RuleSetStatus{Tag: tag, State: toState(verdict), Reason: set.Reason, Detail: set.Detail}
		if !set.UpdatedAt.IsZero() {
			status.UpdatedAtUnix = set.UpdatedAt.Unix()
		}
		statuses = append(statuses, status)

		switch verdict {
		case ruleset.VerdictYes:
			matched = true
		case ruleset.VerdictUnknown:
			unknown = true
		}
	}

	switch {
	case matched:
		return ruleset.VerdictYes, statuses
	case unknown:
		return ruleset.VerdictUnknown, statuses
	default:
		return ruleset.VerdictNo, statuses
	}
}

func matchDomain(raw option.RawDefaultRule, target string) bool {
	for _, exact := range raw.Domain {
		if strings.EqualFold(exact, target) {
			return true
		}
	}
	for _, suffix := range raw.DomainSuffix {
		// sing-box treats a suffix without a leading dot as matching the name
		// itself as well as its subdomains (common/domain).
		lowered := strings.ToLower(suffix)
		if strings.HasSuffix(target, lowered) {
			return true
		}
		if !strings.HasPrefix(lowered, ".") && target == lowered {
			return true
		}
		if !strings.HasPrefix(lowered, ".") && strings.HasSuffix(target, "."+lowered) {
			return true
		}
	}
	for _, keyword := range raw.DomainKeyword {
		if keyword != "" && strings.Contains(target, strings.ToLower(keyword)) {
			return true
		}
	}
	for _, expression := range raw.DomainRegex {
		compiled, err := regexp.Compile(expression)
		if err != nil {
			// sing-box would have rejected the config, so this config is not
			// running as written; skipping beats guessing.
			continue
		}
		if compiled.MatchString(target) {
			return true
		}
	}
	return false
}

func inPrefixes(prefixes []string, address netip.Addr) bool {
	normalized := address.Unmap()
	for _, raw := range prefixes {
		if prefix, err := netip.ParsePrefix(raw); err == nil {
			if prefix.Contains(normalized) {
				return true
			}
			continue
		}
		if single, err := netip.ParseAddr(raw); err == nil && single.Unmap() == normalized {
			return true
		}
	}
	return false
}

func matchPortLists(port uint16, ports []uint16, ranges []string) bool {
	for _, candidate := range ports {
		if candidate == port {
			return true
		}
	}
	for _, raw := range ranges {
		start, end, ok := parseRange(raw)
		if ok && port >= start && port <= end {
			return true
		}
	}
	return false
}

// runtimeOnlyConditions names the conditions that need state living on the
// client's machine or in the running process.
func runtimeOnlyConditions(raw option.RawDefaultRule) []string {
	var conditions []string
	add := func(present bool, name string) {
		if present {
			conditions = append(conditions, name)
		}
	}
	add(len(raw.Client) > 0, "client")
	add(len(raw.AuthUser) > 0, "auth_user")
	add(len(raw.User) > 0, "user")
	add(len(raw.UserID) > 0, "user_id")
	add(len(raw.ProcessName) > 0, "process_name")
	add(len(raw.ProcessPath) > 0, "process_path")
	add(len(raw.ProcessPathRegex) > 0, "process_path_regex")
	add(len(raw.PackageName) > 0, "package_name")
	add(len(raw.NetworkType) > 0, "network_type")
	add(raw.NetworkIsExpensive, "network_is_expensive")
	add(raw.NetworkIsConstrained, "network_is_constrained")
	add(len(raw.WIFISSID) > 0, "wifi_ssid")
	add(len(raw.WIFIBSSID) > 0, "wifi_bssid")
	return conditions
}

func toState(verdict ruleset.Verdict) MatchState {
	switch verdict {
	case ruleset.VerdictYes:
		return MatchStateMatched
	case ruleset.VerdictNo:
		return MatchStateNotMatched
	default:
		return MatchStateUnevaluated
	}
}

func flip(verdict ruleset.Verdict, invert bool) ruleset.Verdict {
	if !invert {
		return verdict
	}
	switch verdict {
	case ruleset.VerdictYes:
		return ruleset.VerdictNo
	case ruleset.VerdictNo:
		return ruleset.VerdictYes
	default:
		return verdict
	}
}
