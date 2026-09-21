package dnsprobe

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// inlineSetLoader builds a loader over one inline rule set, which decodes in
// process and therefore exercises tier 1 without touching a cache file.
func inlineSetLoader(tag string, suffixes []string) *ruleset.Loader {
	return ruleset.NewLoader(nil, &option.Options{
		Route: &option.RouteOptions{RuleSet: []option.RuleSet{{
			Type: C.RuleSetTypeInline,
			Tag:  tag,
			InlineOptions: option.PlainRuleSet{Rules: []option.HeadlessRule{{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultHeadlessRule{DomainSuffix: suffixes},
			}}},
		}}},
	})
}

func ruleSetRule(tag, server string) option.DNSRule {
	return option.DNSRule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{RuleSet: []string{tag}},
			DNSRuleAction: option.DNSRuleAction{
				Action:       C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{Server: server},
			},
		},
	}
}

func TestAttributeQueryDecidesRuleSet(t *testing.T) {
	// Before the loader was wired in, this rule reported `unevaluated` and
	// took the whole prediction below it with it — on a real config, most
	// rules carry nothing but a rule_set.
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{ruleSetRule("geosite-google", "proxy-dns")},
			Final: "local-dns",
		},
	}

	result := AttributeQuery(dns, Query{Domain: "www.google.com", Sets: loader})

	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if got := result.Rules[0].State; got != MatchStateMatched {
		t.Fatalf("state = %q, want matched", got)
	}
	if result.MatchedIndex != 0 || result.Server != "proxy-dns" {
		t.Fatalf("matched=%d server=%q", result.MatchedIndex, result.Server)
	}
	if !result.Exact {
		t.Fatal("a fully decided walk must be exact")
	}
	if len(result.Rules[0].RuleSets) != 1 || result.Rules[0].RuleSets[0].Tag != "geosite-google" {
		t.Fatalf("rule_sets = %+v, want the tag reported", result.Rules[0].RuleSets)
	}
}

func TestAttributeQueryRuleSetMissIsNotAMatch(t *testing.T) {
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{ruleSetRule("geosite-google", "proxy-dns")},
			Final: "local-dns",
		},
	}

	result := AttributeQuery(dns, Query{Domain: "www.baidu.com", Sets: loader})

	if got := result.Rules[0].State; got != MatchStateNotMatched {
		t.Fatalf("state = %q, want not_matched", got)
	}
	if !result.FinalUsed || result.Server != "local-dns" {
		t.Fatalf("expected a fall-through to dns.final, got %+v", result)
	}
	if !result.Exact {
		t.Fatal("a decided miss must not make the walk inexact")
	}
}

func TestAttributeQueryUnreadableSetStaysUnevaluated(t *testing.T) {
	// No loader at all: the rule must remain undecidable rather than
	// collapsing into a miss, which would fabricate a confident prediction.
	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{ruleSetRule("geosite-google", "proxy-dns")},
			Final: "local-dns",
		},
	}

	result := AttributeQuery(dns, Query{Domain: "www.google.com"})

	if got := result.Rules[0].State; got != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got)
	}
	if result.Exact {
		t.Fatal("an undecidable rule ahead of the decision must clear Exact")
	}
}

func TestAttributeQueryUnknownTagIsUndecidableNotAMiss(t *testing.T) {
	// A tag no rule set declares is a config error. sing-box would refuse to
	// start, so this build must not pretend the rule simply did not match.
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{ruleSetRule("geosite-absent", "proxy-dns")},
			Final: "local-dns",
		},
	}

	result := AttributeQuery(dns, Query{Domain: "www.google.com", Sets: loader})

	if got := result.Rules[0].State; got != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got)
	}
	if got := result.Rules[0].RuleSets[0].Reason; got != ruleset.ReasonUnknownTag {
		t.Fatalf("reason = %q, want unknown_tag", got)
	}
}

func queryTypeRule(types []string, server string) option.DNSRule {
	listable := make(badoption.Listable[option.DNSQueryType], 0, len(types))
	for _, name := range types {
		var qType option.DNSQueryType
		if err := qType.UnmarshalJSON([]byte(`"` + name + `"`)); err != nil {
			panic(err)
		}
		listable = append(listable, qType)
	}
	return option.DNSRule{
		Type: C.RuleTypeDefault,
		DefaultOptions: option.DefaultDNSRule{
			RawDefaultDNSRule: option.RawDefaultDNSRule{QueryType: listable},
			DNSRuleAction: option.DNSRuleAction{
				Action:       C.RuleActionTypeRoute,
				RouteOptions: option.DNSRouteActionOptions{Server: server},
			},
		},
	}
}

func TestAttributeQueryDecidesQueryType(t *testing.T) {
	// The probe has always known which record type it asked for and threw it
	// away. An AAAA-suppression rule — the whole basis of an IPv6 split — was
	// therefore undecidable on every probe.
	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{queryTypeRule([]string{"AAAA", "HTTPS"}, "block-dns")},
			Final: "local-dns",
		},
	}

	matched := AttributeQuery(dns, Query{Domain: "www.google.com", Type: "AAAA"})
	if got := matched.Rules[0].State; got != MatchStateMatched {
		t.Fatalf("AAAA state = %q, want matched", got)
	}

	missed := AttributeQuery(dns, Query{Domain: "www.google.com", Type: "A"})
	if got := missed.Rules[0].State; got != MatchStateNotMatched {
		t.Fatalf("A state = %q, want not_matched", got)
	}
	if !missed.FinalUsed {
		t.Fatal("an A query must fall through the AAAA rule")
	}
}

func TestAttributeQueryWithoutTypeLeavesQueryTypeUndecided(t *testing.T) {
	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{queryTypeRule([]string{"AAAA"}, "block-dns")},
			Final: "local-dns",
		},
	}

	result := AttributeQuery(dns, Query{Domain: "www.google.com"})
	if got := result.Rules[0].State; got != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got)
	}
}

func TestAttributeStillWorksWithoutAQuery(t *testing.T) {
	// The old two-argument entry point stays, so existing callers are
	// unchanged and a probe with no loader behaves exactly as before.
	dns := &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{ruleSetRule("geosite-google", "proxy-dns")},
			Final: "local-dns",
		},
	}
	result := Attribute(dns, "www.google.com")
	if result.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", result.Rules[0].State)
	}
}
