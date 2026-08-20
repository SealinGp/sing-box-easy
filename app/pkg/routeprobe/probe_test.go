package routeprobe

import (
	"errors"
	"net/netip"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

func routeRule(raw option.RawDefaultRule, action option.RuleAction) option.Rule {
	return option.Rule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultRule{
			RawDefaultRule: raw,
			RuleAction:     action,
		},
	}
}

func routeTo(outbound string) option.RuleAction {
	return option.RuleAction{
		Action:       C.RuleActionTypeRoute,
		RouteOptions: option.RouteActionOptions{Outbound: outbound},
	}
}

func configWith(rules []option.Rule, final string, outbounds ...string) *option.Options {
	options := &option.Options{
		Route: &option.RouteOptions{Rules: rules, Final: final},
	}
	for _, tag := range outbounds {
		options.Outbounds = append(options.Outbounds, option.Outbound{Tag: tag, Type: C.TypeDirect})
	}
	return options
}

func runOrFail(t *testing.T, options *option.Options, opts Options) *Result {
	t.Helper()
	result, err := Run(options, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return result
}

func TestFirstMatchWins(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}}, routeTo("proxy-a")),
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}}, routeTo("proxy-b")),
	}, "direct", "direct")

	result := runOrFail(t, options, Options{Destination: "www.example.com"})

	if result.Outbound != "proxy-a" {
		t.Errorf("Outbound = %q, want proxy-a", result.Outbound)
	}
	if result.MatchedIndex != 0 {
		t.Errorf("MatchedIndex = %d, want 0", result.MatchedIndex)
	}
	if !result.Exact {
		t.Error("Exact = false, want true")
	}
	// The shadowed rule must still be reported: "what stole my traffic" is
	// answered by seeing rule 1 sitting below the winner, marked skipped.
	if len(result.Rules) != 2 {
		t.Fatalf("got %d evaluations, want 2", len(result.Rules))
	}
	if result.Rules[1].State != MatchStateSkipped {
		t.Errorf("shadowed rule state = %q, want %q", result.Rules[1].State, MatchStateSkipped)
	}
}

// Only route, reject and hijack-dns stop matching (route/route.go:478-484). A
// `sniff` rule matching must NOT end the walk.
func TestNonTerminalActionsContinue(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{}, option.RuleAction{Action: C.RuleActionTypeSniff}),
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}}, routeTo("proxy")),
	}, "direct", "direct")

	result := runOrFail(t, options, Options{Destination: "www.example.com"})

	if result.Outbound != "proxy" {
		t.Errorf("Outbound = %q, want proxy — a sniff rule must not terminate", result.Outbound)
	}
	if result.Rules[0].State != MatchStateMatched {
		t.Errorf("sniff rule state = %q, want matched", result.Rules[0].State)
	}
	if result.Rules[0].Terminal {
		t.Error("sniff reported as terminal")
	}
}

func TestFallThroughSources(t *testing.T) {
	rules := []option.Rule{
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"nomatch.invalid"}}, routeTo("proxy")),
	}

	cases := []struct {
		name     string
		options  *option.Options
		outbound string
		source   string
	}{
		{
			name:     "route.final names it",
			options:  configWith(rules, "my-final", "first", "my-final"),
			outbound: "my-final",
			source:   "route.final",
		},
		{
			// With no final, the FIRST outbound becomes the default
			// (adapter/outbound/manager.go:289).
			name:     "no final falls to the first outbound",
			options:  configWith(rules, "", "first", "second"),
			outbound: "first",
			source:   "first_outbound",
		},
		{
			// With no outbounds at all sing-box synthesises a direct one
			// (box.go:317). Reporting "final" here sends someone hunting for
			// a config key that does not exist.
			name:     "no outbounds at all",
			options:  configWith(rules, ""),
			outbound: "direct",
			source:   "implicit_direct",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runOrFail(t, tc.options, Options{Destination: "example.com"})
			if !result.FinalUsed {
				t.Error("FinalUsed = false")
			}
			if result.Outbound != tc.outbound {
				t.Errorf("Outbound = %q, want %q", result.Outbound, tc.outbound)
			}
			if result.OutboundSource != tc.source {
				t.Errorf("OutboundSource = %q, want %q", result.OutboundSource, tc.source)
			}
		})
	}
}

// ip_is_private joins the destination-address group, so it is OR'd with the
// domain matchers rather than AND'd (route/rule/rule_default.go:152-157 both
// write DestinationAddressMatch).
func TestPrivateAddressRule(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{IPIsPrivate: true}, routeTo("lan")),
	}, "proxy", "proxy")

	private := runOrFail(t, options, Options{Destination: "192.168.9.50"})
	if private.Outbound != "lan" {
		t.Errorf("private address went to %q, want lan", private.Outbound)
	}
	if private.IPSource != "input" {
		t.Errorf("IPSource = %q, want input", private.IPSource)
	}

	public := runOrFail(t, options, Options{Destination: "8.8.8.8"})
	if public.Outbound != "proxy" {
		t.Errorf("public address went to %q, want proxy", public.Outbound)
	}
}

func TestInboundCondition(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{
			Inbound: badoption.Listable[string]{"dns-in"},
		}, option.RuleAction{Action: C.RuleActionTypeHijackDNS}),
	}, "direct", "direct")

	matched := runOrFail(t, options, Options{Destination: "example.com", Inbound: "dns-in"})
	if matched.Action != C.RuleActionTypeHijackDNS {
		t.Errorf("Action = %q, want hijack-dns", matched.Action)
	}

	other := runOrFail(t, options, Options{Destination: "example.com", Inbound: "tun-in"})
	if !other.FinalUsed {
		t.Error("a different inbound should not match the rule")
	}

	// With no inbound supplied the rule cannot be decided, and everything
	// below it becomes a guess rather than a fact.
	unknown := runOrFail(t, options, Options{Destination: "example.com"})
	if unknown.Exact {
		t.Error("Exact = true despite an undecidable inbound rule ahead")
	}
	if unknown.Rules[0].State != MatchStateUnevaluated {
		t.Errorf("state = %q, want unevaluated", unknown.Rules[0].State)
	}
}

// An unevaluated rule ahead of the decision makes the whole verdict a guess:
// it could have matched first.
func TestUnevaluatedRuleAheadClearsExact(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{ClashMode: "global"}, routeTo("everything")),
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}}, routeTo("proxy")),
	}, "direct", "direct")

	guess := runOrFail(t, options, Options{Destination: "example.com"})
	if guess.Exact {
		t.Error("Exact = true with a clash_mode rule ahead")
	}
	if guess.UnevaluatedBefore != 1 {
		t.Errorf("UnevaluatedBefore = %d, want 1", guess.UnevaluatedBefore)
	}
	if guess.Outbound != "proxy" {
		t.Errorf("Outbound = %q, want proxy", guess.Outbound)
	}

	// Supplying the mode makes the same probe exact — and changes the answer.
	known := runOrFail(t, options, Options{Destination: "example.com", ClashMode: "global"})
	if !known.Exact {
		t.Error("Exact = false once clash_mode is known")
	}
	if known.Outbound != "everything" {
		t.Errorf("Outbound = %q, want everything", known.Outbound)
	}
}

func TestDomainRuleNeedsAddressForIPConditions(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{
			IPCIDR: badoption.Listable[string]{"198.51.100.0/24"},
		}, routeTo("special")),
	}, "direct", "direct")

	// No resolver: the address is unknown, so the rule is undecidable.
	unresolved := runOrFail(t, options, Options{Destination: "example.com"})
	if unresolved.Rules[0].State != MatchStateUnevaluated {
		t.Errorf("state = %q, want unevaluated without an address", unresolved.Rules[0].State)
	}

	// With one, it decides.
	resolved := runOrFail(t, options, Options{
		Destination: "example.com",
		Resolve:     func(string) (netip.Addr, error) { return netip.MustParseAddr("198.51.100.7"), nil },
	})
	if resolved.Outbound != "special" {
		t.Errorf("Outbound = %q, want special", resolved.Outbound)
	}
	if resolved.IPSource != "dns" {
		t.Errorf("IPSource = %q, want dns", resolved.IPSource)
	}
}

func TestResolveFailureIsReportedNotFatal(t *testing.T) {
	options := configWith(nil, "direct", "direct")
	result := runOrFail(t, options, Options{
		Destination: "example.com",
		Resolve:     func(string) (netip.Addr, error) { return netip.Addr{}, errors.New("no such host") },
	})
	if result.ResolveError == "" {
		t.Error("ResolveError is empty")
	}
	if result.Outbound != "direct" {
		t.Errorf("Outbound = %q, want direct — a failed lookup must not abort the walk", result.Outbound)
	}
}

func TestPortConditions(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{Port: badoption.Listable[uint16]{853}}, routeTo("dns-proxy")),
	}, "direct", "direct")

	onPort := runOrFail(t, options, Options{Destination: "example.com", Port: 853})
	if onPort.Outbound != "dns-proxy" {
		t.Errorf("Outbound = %q, want dns-proxy", onPort.Outbound)
	}

	// The default port stands in for a browser connection rather than leaving
	// every port rule undecidable.
	defaulted := runOrFail(t, options, Options{Destination: "example.com"})
	if defaulted.Port != DefaultPort {
		t.Errorf("Port = %d, want %d", defaulted.Port, DefaultPort)
	}
	if !defaulted.FinalUsed {
		t.Error("port 443 should not match a port-853 rule")
	}
}

func TestRejectActionOutcome(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"ads.example"}},
			option.RuleAction{
				Action:        C.RuleActionTypeReject,
				RejectOptions: option.RejectActionOptions{Method: "drop"},
			}),
	}, "direct", "direct")

	result := runOrFail(t, options, Options{Destination: "x.ads.example"})
	if result.Action != C.RuleActionTypeReject {
		t.Errorf("Action = %q, want reject", result.Action)
	}
	if result.Outbound != "drop" {
		t.Errorf("Outcome = %q, want drop", result.Outbound)
	}
}

// An omitted action means "route" (sing-box's documented default), and the
// panel deliberately does not write it back into the config — so it has to be
// applied on every read.
func TestOmittedActionDefaultsToRoute(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}},
			option.RuleAction{RouteOptions: option.RouteActionOptions{Outbound: "proxy"}}),
	}, "direct", "direct")

	result := runOrFail(t, options, Options{Destination: "example.com"})
	if result.Outbound != "proxy" {
		t.Errorf("Outbound = %q, want proxy", result.Outbound)
	}
	if result.Rules[0].Action != C.RuleActionTypeRoute {
		t.Errorf("Action = %q, want route", result.Rules[0].Action)
	}
}

func TestInputValidation(t *testing.T) {
	options := configWith(nil, "direct", "direct")

	cases := []struct {
		name string
		opts Options
	}{
		{"empty destination", Options{}},
		{"unsupported network", Options{Destination: "example.com", Network: "sctp"}},
		{"invalid source ip", Options{Destination: "example.com", SourceIP: "not-an-ip"}},
		{"overlong destination", Options{Destination: string(make([]byte, maxDestinationLength+1))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Run(options, tc.opts); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestLogicalRuleTriState(t *testing.T) {
	options := configWith([]option.Rule{{
		Type: C.RuleTypeLogical,
		LogicalOptions: option.LogicalRule{
			RawLogicalRule: option.RawLogicalRule{
				Mode: C.LogicalTypeAnd,
				Rules: []option.Rule{
					routeRule(option.RawDefaultRule{DomainSuffix: badoption.Listable[string]{"example.com"}}, option.RuleAction{}),
					routeRule(option.RawDefaultRule{Port: badoption.Listable[uint16]{443}}, option.RuleAction{}),
				},
			},
			RuleAction: routeTo("both"),
		},
	}}, "direct", "direct")

	both := runOrFail(t, options, Options{Destination: "example.com", Port: 443})
	if both.Outbound != "both" {
		t.Errorf("Outbound = %q, want both", both.Outbound)
	}

	onlyOne := runOrFail(t, options, Options{Destination: "example.com", Port: 80})
	if !onlyOne.FinalUsed {
		t.Error("AND rule matched with only one operand satisfied")
	}
}
