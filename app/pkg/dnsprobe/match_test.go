package dnsprobe

import (
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// routeRule builds a default rule that routes to `server`.
func routeRule(server string, mutate func(*option.DefaultDNSRule)) option.DNSRule {
	rule := option.DefaultDNSRule{}
	rule.RouteOptions.Server = server
	if mutate != nil {
		mutate(&rule)
	}
	return option.DNSRule{Type: "default", DefaultOptions: rule}
}

func dnsWith(rules []option.DNSRule, final string) *option.DNSOptions {
	options := &option.DNSOptions{}
	options.Rules = rules
	options.Final = final
	return options
}

func TestAttributeExactDomainMatch(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_router", func(r *option.DefaultDNSRule) {
			r.Domain = badoption.Listable[string]{"tea.tparts.com"}
		}),
	}, "dns_final")

	got := Attribute(dns, "tea.tparts.com")

	if got.MatchedIndex != 0 {
		t.Errorf("MatchedIndex = %d, want 0", got.MatchedIndex)
	}
	if got.Server != "dns_router" {
		t.Errorf("Server = %q, want dns_router", got.Server)
	}
	if !got.Exact {
		t.Error("Exact = false, want true — nothing was unevaluable")
	}
	if got.FinalUsed {
		t.Error("FinalUsed = true, want false")
	}
}

// domain_suffix must use sing-box's own matcher semantics, not a hand-rolled
// string suffix, which is why the matcher is delegated rather than reimplemented.
func TestAttributeDomainSuffix(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_router", func(r *option.DefaultDNSRule) {
			r.DomainSuffix = badoption.Listable[string]{"owolist.cn"}
		}),
	}, "dns_final")

	cases := []struct {
		domain string
		want   int
	}{
		{"owolist.cn", 0},
		{"www.owolist.cn", 0},
		{"notowolist.cn", -1},
		{"owolist.cn.evil.com", -1},
	}

	for _, c := range cases {
		t.Run(c.domain, func(t *testing.T) {
			if got := Attribute(dns, c.domain).MatchedIndex; got != c.want {
				t.Errorf("MatchedIndex for %q = %d, want %d", c.domain, got, c.want)
			}
		})
	}
}

func TestAttributeFallsThroughToFinal(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_router", func(r *option.DefaultDNSRule) {
			r.Domain = badoption.Listable[string]{"other.com"}
		}),
	}, "dns_final")

	got := Attribute(dns, "example.com")

	if got.MatchedIndex != -1 {
		t.Errorf("MatchedIndex = %d, want -1", got.MatchedIndex)
	}
	if !got.FinalUsed || got.Server != "dns_final" {
		t.Errorf("FinalUsed=%v Server=%q, want true/dns_final", got.FinalUsed, got.Server)
	}
	if !got.Exact {
		t.Error("Exact = false, want true")
	}
}

// A rule_set rule cannot be decided offline. Anything after it is therefore a
// guess, and the result must say so rather than presenting a confident answer.
func TestAttributeRuleSetMakesLaterMatchInexact(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_google", func(r *option.DefaultDNSRule) {
			r.RuleSet = badoption.Listable[string]{"geosite-cn"}
		}),
		routeRule("dns_router", func(r *option.DefaultDNSRule) {
			r.DomainSuffix = badoption.Listable[string]{"example.com"}
		}),
	}, "dns_final")

	got := Attribute(dns, "www.example.com")

	if got.Rules[0].State != MatchStateUnevaluated {
		t.Errorf("rule[0].State = %q, want unevaluated", got.Rules[0].State)
	}
	if len(got.Rules[0].Unevaluated) == 0 || got.Rules[0].Unevaluated[0] != "rule_set" {
		t.Errorf("rule[0].Unevaluated = %v, want [rule_set]", got.Rules[0].Unevaluated)
	}
	if got.MatchedIndex != 1 {
		t.Errorf("MatchedIndex = %d, want 1", got.MatchedIndex)
	}
	if got.Exact {
		t.Error("Exact = true, want false — an earlier rule could have matched first")
	}
	if got.UnevaluatedBefore != 1 {
		t.Errorf("UnevaluatedBefore = %d, want 1", got.UnevaluatedBefore)
	}
}

// A failing domain condition rules the rule out even when another condition is
// unevaluable, because sing-box requires every condition to match.
func TestAttributeDomainMismatchBeatsUnevaluable(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_google", func(r *option.DefaultDNSRule) {
			r.RuleSet = badoption.Listable[string]{"geosite-cn"}
			r.DomainSuffix = badoption.Listable[string]{"other.com"}
		}),
	}, "dns_final")

	got := Attribute(dns, "example.com")

	if got.Rules[0].State != MatchStateNotMatched {
		t.Errorf("State = %q, want not_matched", got.Rules[0].State)
	}
	if !got.Exact {
		t.Error("Exact = false, want true — the rule was definitively excluded")
	}
}

func TestAttributePredefinedAction(t *testing.T) {
	rule := option.DefaultDNSRule{}
	rule.Domain = badoption.Listable[string]{"tea.tparts.com"}
	rule.Action = "predefined"
	dns := dnsWith([]option.DNSRule{{Type: "default", DefaultOptions: rule}}, "dns_final")

	got := Attribute(dns, "tea.tparts.com")

	if got.MatchedIndex != 0 {
		t.Fatalf("MatchedIndex = %d, want 0", got.MatchedIndex)
	}
	if got.Rules[0].Action != "predefined" {
		t.Errorf("Action = %q, want predefined", got.Rules[0].Action)
	}
}

// An omitted action means "route" in sing-box; the UI must not show a blank.
func TestAttributeDefaultsActionToRoute(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_router", func(r *option.DefaultDNSRule) {
			r.Domain = badoption.Listable[string]{"a.com"}
		}),
	}, "dns_final")

	if got := Attribute(dns, "a.com").Rules[0].Action; got != "route" {
		t.Errorf("Action = %q, want route", got)
	}
}

func TestAttributeKeywordAndRegex(t *testing.T) {
	dns := dnsWith([]option.DNSRule{
		routeRule("dns_a", func(r *option.DefaultDNSRule) {
			r.DomainKeyword = badoption.Listable[string]{"google"}
		}),
		routeRule("dns_b", func(r *option.DefaultDNSRule) {
			r.DomainRegex = badoption.Listable[string]{`^ads?\.`}
		}),
	}, "dns_final")

	if got := Attribute(dns, "www.google.com").MatchedIndex; got != 0 {
		t.Errorf("keyword MatchedIndex = %d, want 0", got)
	}
	if got := Attribute(dns, "ad.example.com").MatchedIndex; got != 1 {
		t.Errorf("regex MatchedIndex = %d, want 1", got)
	}
}

func TestAttributeNilDNS(t *testing.T) {
	got := Attribute(nil, "example.com")
	if got.MatchedIndex != -1 || len(got.Rules) != 0 {
		t.Errorf("got %+v, want empty attribution", got)
	}
}

func TestNormalizeQueryDomain(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"plain", "example.com", "example.com", false},
		{"uppercase and trailing dot", "Example.COM.", "example.com", false},
		{"whitespace", "  example.com  ", "example.com", false},
		{"empty", "", "", true},
		{"url", "https://example.com/path", "", true},
		{"host and port", "example.com:53", "", true},
		{"ipv4 literal", "1.1.1.1", "", true},
		{"single label", "localhost", "", true},
		{"empty label", "a..com", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeQueryDomain(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("NormalizeQueryDomain(%q) = %q, want error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
