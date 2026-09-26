package dns

// rule_set and query_type attribution, against raw config JSON.

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// inlineSetLoader builds a loader over one inline rule set, which decodes in
// process and therefore exercises tier 1 without touching a cache file.
func inlineSetLoader(tag string, suffixes []string) *ruleset.Loader {
	return ruleset.NewLoader(nil, &option.Options{
		Route: &option.RouteOptions{RuleSet: []option.RuleSet{{
			Type: C.RuleSetTypeInline,
			Tag:  tag,
			InlineOptions: option.PlainRuleSet{Rules: []option.HeadlessRule{{
				Type:           C.RuleTypeDefault,
				DefaultOptions: option.DefaultHeadlessRule{DomainSuffix: suffixes},
			}}},
		}}},
	})
}

const ruleSetSection = `{"rules":[
  {"rule_set":["geosite-google"],"action":"route","server":"proxy-dns"}
],"final":"local-dns"}`

func TestAttributeDecidesRuleSet(t *testing.T) {
	// Before the loader was wired in, this rule reported `unevaluated` and
	// took the whole prediction below it with it — on a real config, most
	// rules carry nothing but a rule_set.
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	got := attribute(t, ruleSetSection, Query{Domain: "www.google.com", Sets: loader})

	if len(got.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(got.Rules))
	}
	if got.Rules[0].State != MatchStateMatched {
		t.Fatalf("state = %q, want matched", got.Rules[0].State)
	}
	if got.MatchedIndex != 0 || got.Server != "proxy-dns" {
		t.Fatalf("matched=%d server=%q", got.MatchedIndex, got.Server)
	}
	if !got.Exact {
		t.Fatal("a fully decided walk must be exact")
	}
	if len(got.Rules[0].RuleSets) != 1 || got.Rules[0].RuleSets[0].Tag != "geosite-google" {
		t.Fatalf("rule_sets = %+v, want the tag reported", got.Rules[0].RuleSets)
	}
}

func TestAttributeRuleSetMissIsNotAMatch(t *testing.T) {
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	got := attribute(t, ruleSetSection, Query{Domain: "www.baidu.com", Sets: loader})

	if got.Rules[0].State != MatchStateNotMatched {
		t.Fatalf("state = %q, want not_matched", got.Rules[0].State)
	}
	if !got.FinalUsed || got.Server != "local-dns" {
		t.Fatalf("expected a fall-through to dns.final, got %+v", got)
	}
	if !got.Exact {
		t.Fatal("a decided miss must not make the walk inexact")
	}
}

func TestAttributeUnreadableSetStaysUnevaluated(t *testing.T) {
	// No loader at all: the rule must remain undecidable rather than
	// collapsing into a miss, which would fabricate a confident prediction.
	got := attribute(t, ruleSetSection, Query{Domain: "www.google.com"})

	if got.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got.Rules[0].State)
	}
	if got.Exact {
		t.Fatal("an undecidable rule ahead of the decision must clear Exact")
	}
}

func TestAttributeUnknownTagIsUndecidableNotAMiss(t *testing.T) {
	// A tag no rule set declares is a config error. sing-box would refuse to
	// start, so this build must not pretend the rule simply did not match.
	loader := inlineSetLoader("geosite-google", []string{"google.com"})
	defer loader.Close()

	got := attribute(t, `{"rules":[
	  {"rule_set":["geosite-absent"],"action":"route","server":"proxy-dns"}
	],"final":"local-dns"}`, Query{Domain: "www.google.com", Sets: loader})

	if got.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got.Rules[0].State)
	}
	if got.Rules[0].RuleSets[0].Reason != ruleset.ReasonUnknownTag {
		t.Fatalf("reason = %q, want unknown_tag", got.Rules[0].RuleSets[0].Reason)
	}
}

const queryTypeSection = `{"rules":[
  {"query_type":["AAAA","HTTPS"],"action":"route","server":"block-dns"}
],"final":"local-dns"}`

func TestAttributeDecidesQueryType(t *testing.T) {
	// The probe has always known which record type it asked for and threw it
	// away. An AAAA-suppression rule — the whole basis of an IPv6 split — was
	// therefore undecidable on every probe.
	matched := attribute(t, queryTypeSection, Query{Domain: "www.google.com", Type: "AAAA"})
	if matched.Rules[0].State != MatchStateMatched {
		t.Fatalf("AAAA state = %q, want matched", matched.Rules[0].State)
	}

	missed := attribute(t, queryTypeSection, Query{Domain: "www.google.com", Type: "A"})
	if missed.Rules[0].State != MatchStateNotMatched {
		t.Fatalf("A state = %q, want not_matched", missed.Rules[0].State)
	}
	if !missed.FinalUsed {
		t.Fatal("an A query must fall through the AAAA rule")
	}
}

func TestAttributeAcceptsNumericQueryType(t *testing.T) {
	// sing-box accepts both spellings; a config written with numbers must not
	// silently stop matching.
	got := attribute(t, `{"rules":[
	  {"query_type":[28],"action":"route","server":"block-dns"}
	],"final":"local-dns"}`, Query{Domain: "www.google.com", Type: "AAAA"})

	if got.Rules[0].State != MatchStateMatched {
		t.Fatalf("state = %q, want matched (28 == AAAA)", got.Rules[0].State)
	}
}

func TestAttributeWithoutTypeLeavesQueryTypeUndecided(t *testing.T) {
	got := attribute(t, queryTypeSection, Query{Domain: "www.google.com"})
	if got.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got.Rules[0].State)
	}
	if !contains(got.Rules[0].Unevaluated, "query_type") {
		t.Fatalf("unevaluated = %v, want query_type named", got.Rules[0].Unevaluated)
	}
}

func TestAttributeSummaryNamesQueryType(t *testing.T) {
	// A rule decided ENTIRELY by its query_type must not render as
	// "(no conditions)" — a matched rung with no visible reason for matching.
	got := attribute(t, queryTypeSection, Query{Domain: "a.test", Type: "AAAA"})
	if got.Rules[0].Summary == "(no conditions)" {
		t.Fatal("summary lost the query_type condition")
	}
}
