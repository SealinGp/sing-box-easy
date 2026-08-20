// Package routeprobe answers "where would this destination go?" against a
// sing-box config, without sending any traffic and without a running sing-box.
//
// It is deliberately a PRE-flight tool. sing-box's own Clash API already
// reports where connections went once they exist, and dashboards render that
// well; what it cannot answer is the question asked immediately after editing a
// config — "will this do what I meant, and should I roll it back?" — because by
// then the traffic has already taken the wrong path.
//
// The prediction is therefore held to a standard the after-the-fact view does
// not need: every rule ahead of the decision must be fully evaluated for the
// answer to be exact, and when one is not, the result says so rather than
// quietly reporting the first rule it happened to be able to decide.
package routeprobe

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/ruleset"
	"github.com/sagernet/sing-box/option"
)

// maxDestinationLength is the DNS name limit, enforced so a probe cannot be
// used to push arbitrary payloads through the resolver.
const maxDestinationLength = 253

// DefaultPort is assumed when the caller does not name one. Route rules
// frequently key on the port, so a probe with no port at all would report
// "unknown" for rules that any real browser connection decides instantly.
const DefaultPort uint16 = 443

// DefaultNetwork matches what a browser opens.
const DefaultNetwork = "tcp"

// MatchState is the verdict for one route rule.
type MatchState string

const (
	// MatchStateMatched means every condition on the rule was evaluated and
	// all of them matched.
	MatchStateMatched MatchState = "matched"
	// MatchStateNotMatched means at least one evaluated condition failed, so
	// the rule cannot match whatever else could not be evaluated.
	MatchStateNotMatched MatchState = "not_matched"
	// MatchStateUnevaluated means the rule needs runtime state this process
	// does not have, and nothing else ruled it out.
	MatchStateUnevaluated MatchState = "unevaluated"
	// MatchStateSkipped means matching stopped before reaching this rule.
	// Rules after a terminal match are still listed — seeing what a rule was
	// shadowed BY is most of the value when a routing change misfires.
	MatchStateSkipped MatchState = "skipped"
)

// RuleSetStatus reports one rule set referenced by a rule, so a rule that could
// not be decided names the set responsible rather than saying only "rule_set".
type RuleSetStatus struct {
	Tag string `json:"tag"`
	// State is the set's own verdict: matched, not_matched or unevaluated.
	State MatchState `json:"state"`
	// Reason is populated when the set could not be read at all.
	Reason ruleset.Reason `json:"reason,omitempty"`
	Detail string         `json:"detail,omitempty"`
	// UpdatedAtUnix is when sing-box last downloaded the set, 0 when unknown.
	// A routing surprise is often just a stale set.
	UpdatedAtUnix int64 `json:"updated_at_unix,omitempty"`
}

// RuleEvaluation is one route rule's verdict, in config order.
type RuleEvaluation struct {
	Index int        `json:"index"`
	Type  string     `json:"type"`
	State MatchState `json:"state"`
	// Summary renders the rule's conditions for display.
	Summary string `json:"summary"`
	// Unevaluated names the conditions that could not be decided here.
	Unevaluated []string `json:"unevaluated,omitempty"`
	// Action is the rule's action, "route" when omitted — the default
	// sing-box assumes.
	Action string `json:"action"`
	// Outcome is the action's result: the outbound tag for a route, the
	// server for a resolve, and so on.
	Outcome string `json:"outcome,omitempty"`
	// Terminal reports whether this action would stop rule matching.
	Terminal bool `json:"terminal"`
	// Effect describes what a non-terminal rule changed for the rules below
	// it — the reason the ladder is a state machine and not a filter.
	Effect string `json:"effect,omitempty"`
	// RuleSets carries per-set detail when the rule references any.
	RuleSets []RuleSetStatus `json:"rule_sets,omitempty"`
}

// Result is the full answer for one destination.
type Result struct {
	// Destination is the input as normalized.
	Destination string `json:"destination"`
	// Domain and IP are the destination split by kind. A domain probe fills
	// IP only once resolution has supplied one.
	Domain string `json:"domain,omitempty"`
	IP     string `json:"ip,omitempty"`
	// IPSource says where IP came from: "input", "dns" or "" when unresolved.
	// A rule that matched on an address the panel resolved — rather than the
	// address sing-box would resolve — is a prediction about DNS as much as
	// about routing, and the UI has to be able to say so.
	IPSource string `json:"ip_source,omitempty"`
	// ResolveError explains a failed lookup; the probe continues without it.
	ResolveError string `json:"resolve_error,omitempty"`

	Port    uint16 `json:"port"`
	Network string `json:"network"`
	Inbound string `json:"inbound,omitempty"`

	Rules []RuleEvaluation `json:"rules"`
	// MatchedIndex is the rule that decided, or -1 when the destination falls
	// through to route.final.
	MatchedIndex int `json:"matched_index"`
	// Outbound is the tag traffic would leave through.
	Outbound string `json:"outbound"`
	// OutboundSource says which config key produced Outbound: "rule",
	// "route.final", "first_outbound" or "implicit_direct" — sing-box has all
	// three fallbacks and they are not interchangeable when debugging.
	OutboundSource string `json:"outbound_source"`
	// Action is the decisive action; "route" unless a rule rejected or
	// hijacked.
	Action string `json:"action"`
	// FinalUsed reports that no rule matched.
	FinalUsed bool `json:"final_used"`

	// Exact is true only when the verdict cannot be wrong: every rule ahead of
	// the decision was fully evaluated. One unevaluated rule earlier in the
	// list could have matched first, which makes everything below it a guess.
	Exact bool `json:"exact"`
	// UnevaluatedBefore counts the unevaluated rules ahead of the decision.
	UnevaluatedBefore int `json:"unevaluated_before"`
	// Warnings are machine keys the UI translates, e.g. a missing rule set.
	Warnings []string `json:"warnings,omitempty"`
}

// Resolver supplies an address for a domain. It is optional: without one, a
// domain probe simply cannot decide address-based rules and says so.
type Resolver func(domain string) (netip.Addr, error)

// Options controls one probe.
type Options struct {
	// Destination is a domain or an IP address.
	Destination string
	Port        uint16
	Network     string
	// Inbound is the tag traffic arrives on. Frequently decisive: the first
	// rule of a typical config keys on it.
	Inbound string
	// SourceIP is the client address.
	SourceIP string
	// ClashMode is the running instance's mode, when known. Configs commonly
	// carry `clash_mode: global` escape-hatch rules that are otherwise
	// undecidable.
	ClashMode string
	// Resolve turns a domain into an address so address rules can be decided.
	Resolve Resolver
}

// Run evaluates one destination against a config.
//
// It fails only on input that cannot be interpreted. Everything else — an
// unreadable rule set, a name that will not resolve — degrades into the result,
// where it is visible, rather than into an error that hides the rest.
func Run(options *option.Options, opts Options) (*Result, error) {
	destination := strings.TrimSpace(opts.Destination)
	if destination == "" {
		return nil, errors.New("destination is required")
	}
	if len(destination) > maxDestinationLength {
		return nil, fmt.Errorf("destination is longer than %d characters", maxDestinationLength)
	}

	network := strings.ToLower(strings.TrimSpace(opts.Network))
	switch network {
	case "":
		network = DefaultNetwork
	case "tcp", "udp":
	default:
		return nil, fmt.Errorf("unsupported network %q", opts.Network)
	}

	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}

	result := &Result{
		Destination:  destination,
		Port:         port,
		Network:      network,
		Inbound:      strings.TrimSpace(opts.Inbound),
		Rules:        []RuleEvaluation{},
		MatchedIndex: -1,
		Exact:        true,
	}

	target := ruleset.Target{Port: port, Network: network}

	// A literal address is matched as-is; anything else is a name, and its
	// address has to be resolved before ip_cidr and geoip rules mean anything.
	if address, err := netip.ParseAddr(destination); err == nil {
		result.IP = address.String()
		result.IPSource = "input"
		target.IP = address
	} else {
		result.Domain = normalizeDomain(destination)
		target.Domain = result.Domain
		if opts.Resolve != nil {
			resolved, resolveErr := opts.Resolve(result.Domain)
			if resolveErr != nil {
				result.ResolveError = resolveErr.Error()
			} else if resolved.IsValid() {
				result.IP = resolved.String()
				result.IPSource = "dns"
				target.IP = resolved
			}
		}
	}

	if sourceIP := strings.TrimSpace(opts.SourceIP); sourceIP != "" {
		address, err := netip.ParseAddr(sourceIP)
		if err != nil {
			return nil, fmt.Errorf("invalid source IP %q", opts.SourceIP)
		}
		target.SourceIP = address
	}

	loader := ruleset.NewLoader(nil, options)
	defer loader.Close()

	evaluator := &evaluator{
		loader:    loader,
		inbound:   result.Inbound,
		clashMode: strings.TrimSpace(opts.ClashMode),
		target:    target,
	}
	evaluator.walk(options, result)

	return result, nil
}

func normalizeDomain(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}
