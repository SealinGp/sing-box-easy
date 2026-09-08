package ruleset

import (
	"net/netip"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/domain"
	"github.com/sagernet/sing/common/json/badoption"
)

func compileOrFail(t *testing.T, rule option.HeadlessRule) matcher {
	t.Helper()
	compiled, err := compile(rule)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return compiled
}

func defaultRule(options option.DefaultHeadlessRule) option.HeadlessRule {
	return option.HeadlessRule{Type: "default", DefaultOptions: options}
}

func verdictName(v Verdict) string {
	switch v {
	case VerdictYes:
		return "YES"
	case VerdictNo:
		return "NO"
	default:
		return "UNKNOWN"
	}
}

func assertVerdict(t *testing.T, got, want Verdict, format string, args ...any) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %s, want %s", sprintf(format, args...), verdictName(got), verdictName(want))
	}
}

func TestDomainConditions(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Domain:       badoption.Listable[string]{"example.com"},
		DomainSuffix: badoption.Listable[string]{".cdn.net"},
	}))

	cases := []struct {
		domain string
		want   Verdict
	}{
		{"example.com", VerdictYes},
		{"a.cdn.net", VerdictYes},
		// `domain` is exact: a subdomain of an exact entry does not match.
		{"www.example.com", VerdictNo},
		{"other.org", VerdictNo},
		// No domain supplied: the rule tests one, so the answer is unknown
		// rather than a confident miss.
		{"", VerdictUnknown},
	}
	for _, tc := range cases {
		assertVerdict(t, rule.match(Target{Domain: tc.domain}), tc.want, "domain %q", tc.domain)
	}
}

// TestDomainOrIPCIDRShareOneGroup pins the least obvious rule in the engine:
// domain* and ip_cidr both write metadata.DestinationAddressMatch
// (route/rule/rule_abstract.go:79-97), so a rule carrying both is a
// DISJUNCTION. Reading the struct as a conjunction — the natural assumption —
// makes every such rule far too narrow.
func TestDomainOrIPCIDRShareOneGroup(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Domain: badoption.Listable[string]{"example.com"},
		IPCIDR: badoption.Listable[string]{"10.0.0.0/8"},
	}))

	// Domain hits, address does not.
	assertVerdict(t, rule.match(Target{Domain: "example.com", IP: netip.MustParseAddr("8.8.8.8")}),
		VerdictYes, "domain hit, ip miss")
	// Address hits, domain does not.
	assertVerdict(t, rule.match(Target{Domain: "other.org", IP: netip.MustParseAddr("10.1.2.3")}),
		VerdictYes, "ip hit, domain miss")
	// Neither.
	assertVerdict(t, rule.match(Target{Domain: "other.org", IP: netip.MustParseAddr("8.8.8.8")}),
		VerdictNo, "both miss")
}

// TestInvertedLogicalRule reproduces the shape every AdGuard-derived rule set
// uses — AND(NOT exceptions, blocklist) — which is how anti-ad is published.
//
// This is a regression test for a real bug: the group helpers used to apply
// `invert` themselves and then hand the result back to match(), which applied
// it again. An inverted rule whose domain missed reported a SATISFIED group and
// fell through, so anti-ad answered "no" for doubleclick.net.
func TestInvertedLogicalRule(t *testing.T) {
	blocked := option.DefaultHeadlessRule{
		DomainSuffix: badoption.Listable[string]{"doubleclick.net", "ads.example"},
	}
	exception := option.DefaultHeadlessRule{
		Domain: badoption.Listable[string]{"allowed.ads.example"},
		Invert: true,
	}

	rule := compileOrFail(t, option.HeadlessRule{
		Type: "logical",
		LogicalOptions: option.LogicalHeadlessRule{
			Mode:  "and",
			Rules: []option.HeadlessRule{defaultRule(exception), defaultRule(blocked)},
		},
	})

	cases := []struct {
		domain string
		want   Verdict
	}{
		{"x.doubleclick.net", VerdictYes},
		// On the exception list, so the inverted operand fails the AND.
		{"allowed.ads.example", VerdictNo},
		{"example.com", VerdictNo},
	}
	for _, tc := range cases {
		assertVerdict(t, rule.match(Target{Domain: tc.domain}), tc.want, "adguard-shaped %q", tc.domain)
	}
}

func TestInvertedDefaultRule(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Domain: badoption.Listable[string]{"example.com"},
		Invert: true,
	}))

	assertVerdict(t, rule.match(Target{Domain: "example.com"}), VerdictNo, "inverted hit")
	assertVerdict(t, rule.match(Target{Domain: "other.org"}), VerdictYes, "inverted miss")
	// Unknown survives inversion: not knowing is not the opposite of knowing.
	assertVerdict(t, rule.match(Target{}), VerdictUnknown, "inverted unknown")
}

func TestLogicalOrTriState(t *testing.T) {
	// One operand decidable and false, one undecidable.
	rule := compileOrFail(t, option.HeadlessRule{
		Type: "logical",
		LogicalOptions: option.LogicalHeadlessRule{
			Mode: "or",
			Rules: []option.HeadlessRule{
				defaultRule(option.DefaultHeadlessRule{Domain: badoption.Listable[string]{"example.com"}}),
				defaultRule(option.DefaultHeadlessRule{ProcessName: badoption.Listable[string]{"curl"}}),
			},
		},
	})

	// A definite hit beats the unknown operand.
	assertVerdict(t, rule.match(Target{Domain: "example.com"}), VerdictYes, "or with hit")
	// Nothing hit, but the process operand could still have.
	assertVerdict(t, rule.match(Target{Domain: "other.org"}), VerdictUnknown, "or with unknown")
}

func TestUnevaluableConditionBlocksAMatch(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Domain:      badoption.Listable[string]{"example.com"},
		ProcessName: badoption.Listable[string]{"firefox"},
	}))

	// Everything visible matches, but sing-box also requires the process name,
	// which this process cannot see.
	assertVerdict(t, rule.match(Target{Domain: "example.com"}), VerdictUnknown, "domain hit, process unknown")
	// A decidable failure still ends it: no need to know the process.
	assertVerdict(t, rule.match(Target{Domain: "other.org"}), VerdictNo, "domain miss short-circuits")
}

// A rule with no conditions matches everything (route/rule/rule_abstract.go:55).
func TestEmptyRuleMatchesEverything(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{}))
	assertVerdict(t, rule.match(Target{}), VerdictYes, "empty rule")
	assertVerdict(t, rule.match(Target{Domain: "anything.example"}), VerdictYes, "empty rule with domain")
}

func TestPortConditions(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Port:      badoption.Listable[uint16]{443},
		PortRange: badoption.Listable[string]{"8000:8080"},
	}))

	assertVerdict(t, rule.match(Target{Port: 443}), VerdictYes, "exact port")
	assertVerdict(t, rule.match(Target{Port: 8080}), VerdictYes, "range upper bound")
	assertVerdict(t, rule.match(Target{Port: 80}), VerdictNo, "outside")
	assertVerdict(t, rule.match(Target{}), VerdictUnknown, "no port supplied")
}

func TestNetworkCondition(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		Network: badoption.Listable[string]{"udp"},
	}))

	assertVerdict(t, rule.match(Target{Network: "udp"}), VerdictYes, "udp")
	assertVerdict(t, rule.match(Target{Network: "tcp"}), VerdictNo, "tcp")
	assertVerdict(t, rule.match(Target{}), VerdictUnknown, "network unset")
}

// A binary .srs carries a prebuilt matcher rather than the string lists, so the
// compiler has to read both shapes (route/rule/rule_headless.go:48-58).
func TestPrebuiltDomainMatcher(t *testing.T) {
	rule := compileOrFail(t, defaultRule(option.DefaultHeadlessRule{
		DomainMatcher: domain.NewMatcher([]string{"example.com"}, []string{".cdn.net"}, false),
	}))

	assertVerdict(t, rule.match(Target{Domain: "example.com"}), VerdictYes, "prebuilt exact")
	assertVerdict(t, rule.match(Target{Domain: "a.cdn.net"}), VerdictYes, "prebuilt suffix")
	assertVerdict(t, rule.match(Target{Domain: "other.org"}), VerdictNo, "prebuilt miss")
}

func TestParsePortRange(t *testing.T) {
	cases := []struct {
		raw   string
		start uint16
		end   uint16
		fails bool
	}{
		{raw: "1000:2000", start: 1000, end: 2000},
		{raw: ":1024", start: 0, end: 1024},
		{raw: "1024:", start: 1024, end: 65535},
		{raw: "not-a-range", fails: true},
	}
	for _, tc := range cases {
		parsed, err := parsePortRange(tc.raw)
		if tc.fails {
			if err == nil {
				t.Errorf("parsePortRange(%q): expected an error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePortRange(%q): %v", tc.raw, err)
			continue
		}
		if parsed.start != tc.start || parsed.end != tc.end {
			t.Errorf("parsePortRange(%q) = %d:%d, want %d:%d", tc.raw, parsed.start, parsed.end, tc.start, tc.end)
		}
	}
}
