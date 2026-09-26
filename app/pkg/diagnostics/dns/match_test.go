package dns

// The original attribution tests, rewritten against raw config JSON.
//
// They used to build `option.DNSOptions` values. That input shape was dropped
// because re-encoding it to reach the walk is lossy in upstream sing-box —
// `DNSRuleAction.MarshalJSON` omits the route options whenever `Action` is
// unset, which is the default spelling, so {"domain":"x","server":"s"} marshals
// back as {"domain":"x"}. Written as JSON these tests also describe the shape a
// config is actually stored in, which is what the walk now reads.

import (
	"testing"
)

func TestAttributeExactDomainMatch(t *testing.T) {
	got := attribute(t, `{"rules":[
	  {"domain":["tea.tparts.com"],"server":"dns_router"}
	],"final":"dns_final"}`, Query{Domain: "tea.tparts.com"})

	if got.MatchedIndex != 0 {
		t.Errorf("MatchedIndex = %d, want 0", got.MatchedIndex)
	}
	if got.Server != "dns_router" {
		t.Errorf("Server = %q, want dns_router", got.Server)
	}
	if got.FinalUsed {
		t.Errorf("FinalUsed = true, want false")
	}
	if !got.Exact {
		t.Errorf("Exact = false, want true")
	}
}

func TestAttributeDomainSuffix(t *testing.T) {
	section := `{"rules":[
	  {"domain_suffix":["owolist.cn"],"server":"dns_router"}
	],"final":"dns_final"}`

	cases := []struct {
		domain string
		want   int
	}{
		{"owolist.cn", 0},
		{"www.owolist.cn", 0},
		// The suffix is a DOMAIN suffix, not a string suffix: sing-box matches
		// the name itself and anything under it, never a different name that
		// merely ends in the same characters.
		{"notowolist.cn", -1},
		{"example.com", -1},
	}

	for _, c := range cases {
		t.Run(c.domain, func(t *testing.T) {
			if got := attribute(t, section, Query{Domain: c.domain}).MatchedIndex; got != c.want {
				t.Errorf("MatchedIndex for %q = %d, want %d", c.domain, got, c.want)
			}
		})
	}
}

func TestAttributeFallsThroughToFinal(t *testing.T) {
	got := attribute(t, `{"rules":[
	  {"domain":["tea.tparts.com"],"server":"dns_router"}
	],"final":"dns_final"}`, Query{Domain: "example.com"})

	if got.MatchedIndex != -1 || !got.FinalUsed {
		t.Errorf("got %+v, want a fall-through", got)
	}
	if got.Server != "dns_final" {
		t.Errorf("Server = %q, want dns_final", got.Server)
	}
	if !got.Exact {
		t.Errorf("Exact = false, want true")
	}
}

func TestAttributeRuleSetMakesLaterMatchInexact(t *testing.T) {
	// A rule_set that cannot be consulted sits ahead of the decision, so the
	// rule below it might never have been reached.
	got := attribute(t, `{"rules":[
	  {"rule_set":["geosite-cn"],"server":"dns_local"},
	  {"domain_suffix":["example.com"],"server":"dns_router"}
	],"final":"dns_final"}`, Query{Domain: "www.example.com"})

	if got.Rules[0].State != MatchStateUnevaluated {
		t.Errorf("rule 0 state = %q, want unevaluated", got.Rules[0].State)
	}
	if got.MatchedIndex != 1 {
		t.Errorf("MatchedIndex = %d, want 1", got.MatchedIndex)
	}
	if got.Exact {
		t.Errorf("Exact = true, want false")
	}
	if got.UnevaluatedBefore != 1 {
		t.Errorf("UnevaluatedBefore = %d, want 1", got.UnevaluatedBefore)
	}
}

func TestAttributeDomainMismatchBeatsUnevaluable(t *testing.T) {
	// Conditions are AND'd, so a decided domain miss rules the rule out even
	// though its rule_set could not be read. Reporting "unevaluated" here
	// would make every such walk inexact for no reason.
	got := attribute(t, `{"rules":[
	  {"domain":["other.com"],"rule_set":["geosite-cn"],"server":"dns_local"}
	],"final":"dns_final"}`, Query{Domain: "example.com"})

	if got.Rules[0].State != MatchStateNotMatched {
		t.Errorf("state = %q, want not_matched", got.Rules[0].State)
	}
	if !got.Exact {
		t.Errorf("Exact = false, want true")
	}
}

func TestAttributePredefinedAction(t *testing.T) {
	got := attribute(t, `{"rules":[
	  {"domain":["tea.tparts.com"],"action":"predefined","rcode":"NOERROR"}
	],"final":"dns_final"}`, Query{Domain: "tea.tparts.com"})

	if got.MatchedIndex != 0 {
		t.Errorf("MatchedIndex = %d, want 0", got.MatchedIndex)
	}
	if got.Rules[0].Action != "predefined" {
		t.Errorf("Action = %q, want predefined", got.Rules[0].Action)
	}
	if !got.Rules[0].Terminal {
		t.Error("predefined must terminate the walk")
	}
}

func TestAttributeDefaultsActionToRoute(t *testing.T) {
	got := attribute(t, `{"rules":[
	  {"domain":["a.com"],"server":"s"}
	],"final":"f"}`, Query{Domain: "a.com"})

	if got.Rules[0].Action != "route" {
		t.Errorf("Action = %q, want route", got.Rules[0].Action)
	}
}

func TestAttributeKeywordAndRegex(t *testing.T) {
	section := `{"rules":[
	  {"domain_keyword":["google"],"server":"s1"},
	  {"domain_regex":["^ad\\..*"],"server":"s2"}
	],"final":"f"}`

	if got := attribute(t, section, Query{Domain: "www.google.com"}).MatchedIndex; got != 0 {
		t.Errorf("keyword MatchedIndex = %d, want 0", got)
	}
	if got := attribute(t, section, Query{Domain: "ad.example.com"}).MatchedIndex; got != 1 {
		t.Errorf("regex MatchedIndex = %d, want 1", got)
	}
}

func TestAttributeEmptySection(t *testing.T) {
	got := AttributeRaw(nil, Query{Domain: "example.com"})
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
