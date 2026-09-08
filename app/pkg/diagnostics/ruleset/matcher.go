package ruleset

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/domain"

	"go4.org/netipx"
)

// Verdict is a tri-state match result.
//
// The third state is the point of this package. "This rule set does not match"
// and "this rule set could not be consulted" lead to opposite conclusions about
// where a connection goes, and a two-state matcher is forced to guess.
type Verdict uint8

const (
	// VerdictNo means the rule was fully evaluated and did not match.
	VerdictNo Verdict = iota
	// VerdictYes means the rule was fully evaluated and matched.
	VerdictYes
	// VerdictUnknown means the rule carries a condition this process cannot
	// evaluate (a process name, the WiFi SSID) and nothing else ruled it out.
	VerdictUnknown
)

// Target is the connection being simulated.
//
// Zero values mean "not supplied", and any condition that would test an unset
// field yields VerdictUnknown rather than a confident miss — a rule that keys
// on source_ip_cidr cannot be decided for a caller who did not say where the
// traffic comes from.
type Target struct {
	// Domain is the destination name, already lowercased and dot-stripped.
	Domain string
	// IP is the destination address. For a domain target this is the resolved
	// address, which is what sing-box matches ip_cidr against once a resolve
	// action or a sniffed connection has produced it.
	IP         netip.Addr
	Port       uint16
	Network    string
	SourceIP   netip.Addr
	SourcePort uint16
}

// matcher evaluates one compiled headless rule.
type matcher interface {
	match(target Target) Verdict
}

// defaultMatcher mirrors route/rule.abstractDefaultRule.Match.
//
// The grouping is not cosmetic and is easy to get wrong:
//
//   - Within a group the items are OR'd.
//   - Between groups, and against `items`, they are AND'd.
//   - domain* and ip_cidr share ONE group flag upstream
//     (rule_abstract.go:79-97 both write metadata.DestinationAddressMatch),
//     so a rule carrying both `domain_suffix` and `ip_cidr` matches when
//     EITHER hits. Reading the field list as a conjunction — the obvious
//     assumption — gets this backwards.
//   - A rule with no items at all matches everything (rule_abstract.go:55).
type defaultMatcher struct {
	invert bool

	// destination address group: domain matchers and ip_cidr, OR'd together.
	domainMatcher  *domain.Matcher
	adGuardMatcher *domain.AdGuardMatcher
	domainKeyword  []string
	domainRegex    []*regexp.Regexp
	ipSet          *netipx.IPSet
	hasDestAddress bool

	// destination port group
	ports     []uint16
	portRange []portRange
	hasPort   bool

	// source groups
	sourceIPSet     *netipx.IPSet
	hasSourceIP     bool
	sourcePorts     []uint16
	sourcePortRange []portRange
	hasSourcePort   bool

	// AND'd individually
	networks []string

	// unevaluable names the conditions that need runtime state this process
	// does not have. Their presence downgrades a would-be match to Unknown.
	unevaluable []string

	empty bool
}

type portRange struct{ start, end uint16 }

func (m *defaultMatcher) match(target Target) Verdict {
	if m.empty {
		return m.flip(VerdictYes)
	}

	// `items` first: these are hard AND conditions, and a failure here ends it
	// regardless of the address groups.
	if len(m.networks) > 0 {
		if target.Network == "" {
			return VerdictUnknown
		}
		if !containsFold(m.networks, target.Network) {
			return m.flip(VerdictNo)
		}
	}

	// Each group answers only "was I satisfied?" — never the rule's verdict.
	// `invert` is applied exactly once, at the end. Folding it into the groups
	// instead inverts twice on the miss path: an inverted rule whose domain
	// group missed would report a satisfied group and fall through to the next
	// check rather than returning.
	for _, group := range []func(Target) Verdict{
		m.matchDestinationAddress,
		m.matchDestinationPort,
		m.matchSourceAddress,
		m.matchSourcePort,
	} {
		switch group(target) {
		case VerdictUnknown:
			return VerdictUnknown
		case VerdictNo:
			return m.flip(VerdictNo)
		}
	}

	// Everything decidable matched. If the rule also carries a condition we
	// cannot see, we cannot claim the match — sing-box requires ALL of them.
	if len(m.unevaluable) > 0 {
		return VerdictUnknown
	}
	return m.flip(VerdictYes)
}

// matchDestinationAddress evaluates the domain matchers and ip_cidr as one
// OR'd group, per the shared DestinationAddressMatch flag upstream.
//
// Like every group helper it returns whether the GROUP was satisfied, not the
// rule's verdict: `invert` belongs to the rule and is applied once by match().
func (m *defaultMatcher) matchDestinationAddress(target Target) Verdict {
	if !m.hasDestAddress {
		return VerdictYes
	}

	// Track whether any member of the group could not be decided: a miss on
	// the decidable half is only a real miss if the rest was decidable too.
	undecided := false

	if m.domainMatcher != nil || m.adGuardMatcher != nil || len(m.domainKeyword) > 0 || len(m.domainRegex) > 0 {
		if target.Domain == "" {
			undecided = true
		} else {
			if m.domainMatcher != nil && m.domainMatcher.Match(target.Domain) {
				return VerdictYes
			}
			if m.adGuardMatcher != nil && m.adGuardMatcher.Match(target.Domain) {
				return VerdictYes
			}
			for _, keyword := range m.domainKeyword {
				if keyword != "" && strings.Contains(target.Domain, keyword) {
					return VerdictYes
				}
			}
			for _, expression := range m.domainRegex {
				if expression.MatchString(target.Domain) {
					return VerdictYes
				}
			}
		}
	}

	if m.ipSet != nil {
		if !target.IP.IsValid() {
			undecided = true
		} else if m.ipSet.Contains(target.IP.Unmap()) {
			return VerdictYes
		}
	}

	if undecided {
		return VerdictUnknown
	}
	return VerdictNo
}

func (m *defaultMatcher) matchDestinationPort(target Target) Verdict {
	if !m.hasPort {
		return VerdictYes
	}
	if target.Port == 0 {
		return VerdictUnknown
	}
	if matchPort(target.Port, m.ports, m.portRange) {
		return VerdictYes
	}
	return VerdictNo
}

func (m *defaultMatcher) matchSourceAddress(target Target) Verdict {
	if !m.hasSourceIP {
		return VerdictYes
	}
	if !target.SourceIP.IsValid() {
		return VerdictUnknown
	}
	if m.sourceIPSet != nil && m.sourceIPSet.Contains(target.SourceIP.Unmap()) {
		return VerdictYes
	}
	return VerdictNo
}

func (m *defaultMatcher) matchSourcePort(target Target) Verdict {
	if !m.hasSourcePort {
		return VerdictYes
	}
	if target.SourcePort == 0 {
		return VerdictUnknown
	}
	if matchPort(target.SourcePort, m.sourcePorts, m.sourcePortRange) {
		return VerdictYes
	}
	return VerdictNo
}

// flip applies `invert`. Unknown is never flipped: not knowing the answer is
// not the same as knowing the opposite.
func (m *defaultMatcher) flip(verdict Verdict) Verdict {
	if !m.invert {
		return verdict
	}
	switch verdict {
	case VerdictYes:
		return VerdictNo
	case VerdictNo:
		return VerdictYes
	default:
		return verdict
	}
}

// logicalMatcher mirrors route/rule.LogicalRule with tri-state operands.
type logicalMatcher struct {
	mode   string
	rules  []matcher
	invert bool
}

func (m *logicalMatcher) match(target Target) Verdict {
	and := strings.EqualFold(m.mode, "and")
	unknown := false

	for _, rule := range m.rules {
		switch rule.match(target) {
		case VerdictUnknown:
			unknown = true
		case VerdictYes:
			if !and {
				return flipVerdict(VerdictYes, m.invert)
			}
		case VerdictNo:
			if and {
				return flipVerdict(VerdictNo, m.invert)
			}
		}
	}

	// A definite short-circuit above beats an unknown; reaching here means the
	// unknown operands are the only thing standing between us and an answer.
	if unknown {
		return VerdictUnknown
	}
	if and {
		return flipVerdict(VerdictYes, m.invert)
	}
	return flipVerdict(VerdictNo, m.invert)
}

func flipVerdict(verdict Verdict, invert bool) Verdict {
	if !invert {
		return verdict
	}
	switch verdict {
	case VerdictYes:
		return VerdictNo
	case VerdictNo:
		return VerdictYes
	default:
		return verdict
	}
}

func matchPort(port uint16, ports []uint16, ranges []portRange) bool {
	for _, candidate := range ports {
		if candidate == port {
			return true
		}
	}
	for _, r := range ranges {
		if port >= r.start && port <= r.end {
			return true
		}
	}
	return false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

// compile turns an option rule into a matcher.
func compile(rule option.HeadlessRule) (matcher, error) {
	switch rule.Type {
	case "", "default":
		return compileDefault(rule.DefaultOptions)
	case "logical":
		compiled := &logicalMatcher{mode: rule.LogicalOptions.Mode, invert: rule.LogicalOptions.Invert}
		for _, sub := range rule.LogicalOptions.Rules {
			subMatcher, err := compile(sub)
			if err != nil {
				return nil, err
			}
			compiled.rules = append(compiled.rules, subMatcher)
		}
		return compiled, nil
	default:
		return nil, errUnknownRuleType(rule.Type)
	}
}

func compileDefault(options option.DefaultHeadlessRule) (matcher, error) {
	compiled := &defaultMatcher{invert: options.Invert}

	// Domain matchers. The pre-built matcher on the struct is what a binary
	// .srs carries — srs.Read populates DomainMatcher rather than the string
	// lists — so both shapes must be handled, exactly as NewDefaultHeadlessRule
	// does (rule_headless.go:48-58).
	if len(options.Domain) > 0 || len(options.DomainSuffix) > 0 {
		compiled.domainMatcher = domain.NewMatcher(options.Domain, options.DomainSuffix, false)
		compiled.hasDestAddress = true
	} else if options.DomainMatcher != nil {
		compiled.domainMatcher = options.DomainMatcher
		compiled.hasDestAddress = true
	}
	if len(options.AdGuardDomain) > 0 {
		compiled.adGuardMatcher = domain.NewAdGuardMatcher(options.AdGuardDomain)
		compiled.hasDestAddress = true
	} else if options.AdGuardDomainMatcher != nil {
		compiled.adGuardMatcher = options.AdGuardDomainMatcher
		compiled.hasDestAddress = true
	}
	if len(options.DomainKeyword) > 0 {
		compiled.domainKeyword = options.DomainKeyword
		compiled.hasDestAddress = true
	}
	for _, expression := range options.DomainRegex {
		compiledExpression, err := regexp.Compile(expression)
		if err != nil {
			return nil, err
		}
		compiled.domainRegex = append(compiled.domainRegex, compiledExpression)
		compiled.hasDestAddress = true
	}

	if len(options.IPCIDR) > 0 {
		set, err := buildIPSet(options.IPCIDR)
		if err != nil {
			return nil, err
		}
		compiled.ipSet = set
		compiled.hasDestAddress = true
	} else if options.IPSet != nil {
		compiled.ipSet = options.IPSet
		compiled.hasDestAddress = true
	}

	if len(options.SourceIPCIDR) > 0 {
		set, err := buildIPSet(options.SourceIPCIDR)
		if err != nil {
			return nil, err
		}
		compiled.sourceIPSet = set
		compiled.hasSourceIP = true
	} else if options.SourceIPSet != nil {
		compiled.sourceIPSet = options.SourceIPSet
		compiled.hasSourceIP = true
	}

	if len(options.Port) > 0 {
		compiled.ports = options.Port
		compiled.hasPort = true
	}
	for _, raw := range options.PortRange {
		parsed, err := parsePortRange(raw)
		if err != nil {
			return nil, err
		}
		compiled.portRange = append(compiled.portRange, parsed)
		compiled.hasPort = true
	}
	if len(options.SourcePort) > 0 {
		compiled.sourcePorts = options.SourcePort
		compiled.hasSourcePort = true
	}
	for _, raw := range options.SourcePortRange {
		parsed, err := parsePortRange(raw)
		if err != nil {
			return nil, err
		}
		compiled.sourcePortRange = append(compiled.sourcePortRange, parsed)
		compiled.hasSourcePort = true
	}

	if len(options.Network) > 0 {
		compiled.networks = options.Network
	}

	// Conditions that need state only a running sing-box on the client's
	// machine has. query_type is listed because a rule set can be shared
	// between the DNS and route rule trees; in a route context sing-box would
	// compare it against an unset value, and reporting that as a confident
	// miss would be a guess dressed as a fact.
	addIf := func(present bool, name string) {
		if present {
			compiled.unevaluable = append(compiled.unevaluable, name)
		}
	}
	addIf(len(options.ProcessName) > 0, "process_name")
	addIf(len(options.ProcessPath) > 0, "process_path")
	addIf(len(options.ProcessPathRegex) > 0, "process_path_regex")
	addIf(len(options.PackageName) > 0, "package_name")
	addIf(len(options.NetworkType) > 0, "network_type")
	addIf(options.NetworkIsExpensive, "network_is_expensive")
	addIf(options.NetworkIsConstrained, "network_is_constrained")
	addIf(len(options.WIFISSID) > 0, "wifi_ssid")
	addIf(len(options.WIFIBSSID) > 0, "wifi_bssid")
	addIf(len(options.QueryType) > 0, "query_type")

	compiled.empty = !compiled.hasDestAddress && !compiled.hasPort &&
		!compiled.hasSourceIP && !compiled.hasSourcePort &&
		len(compiled.networks) == 0 && len(compiled.unevaluable) == 0

	return compiled, nil
}

func buildIPSet(prefixes []string) (*netipx.IPSet, error) {
	var builder netipx.IPSetBuilder
	for _, raw := range prefixes {
		prefix, err := netip.ParsePrefix(raw)
		if err == nil {
			builder.AddPrefix(prefix)
			continue
		}
		address, addrErr := netip.ParseAddr(raw)
		if addrErr != nil {
			return nil, err
		}
		builder.Add(address)
	}
	return builder.IPSet()
}

func parsePortRange(raw string) (portRange, error) {
	// sing-box accepts "1000:2000", and an open end on either side.
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return portRange{}, errBadPortRange(raw)
	}
	result := portRange{start: 0, end: 65535}
	if parts[0] != "" {
		value, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return portRange{}, err
		}
		result.start = uint16(value)
	}
	if parts[1] != "" {
		value, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return portRange{}, err
		}
		result.end = uint16(value)
	}
	return result, nil
}
