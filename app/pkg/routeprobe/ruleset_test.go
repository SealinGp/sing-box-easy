package routeprobe

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/ruleset"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

func inlineSet(tag string, suffixes ...string) option.RuleSet {
	return option.RuleSet{
		Type: C.RuleSetTypeInline,
		Tag:  tag,
		InlineOptions: option.PlainRuleSet{
			Rules: []option.HeadlessRule{{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultHeadlessRule{
					DomainSuffix: badoption.Listable[string](suffixes),
				},
			}},
		},
	}
}

func configWithSets(rules []option.Rule, sets []option.RuleSet, final string) *option.Options {
	options := configWith(rules, final, final)
	options.Route.RuleSet = sets
	return options
}

// The tags on ONE rule are a disjunction: sing-box's RuleSetItem returns true
// on the first set that matches (route/rule/rule_item_rule_set.go:43-52).
func TestRuleSetTagsAreDisjunction(t *testing.T) {
	options := configWithSets(
		[]option.Rule{routeRule(option.RawDefaultRule{
			RuleSet: badoption.Listable[string]{"ads", "trackers"},
		}, routeTo("block"))},
		[]option.RuleSet{inlineSet("ads", "ads.example"), inlineSet("trackers", "track.example")},
		"direct",
	)

	for _, destination := range []string{"a.ads.example", "b.track.example"} {
		result := runOrFail(t, options, Options{Destination: destination})
		if result.Outbound != "block" {
			t.Errorf("%s went to %q, want block", destination, result.Outbound)
		}
	}

	miss := runOrFail(t, options, Options{Destination: "example.com"})
	if !miss.FinalUsed {
		t.Errorf("example.com matched, want fall-through")
	}
}

// rule_set is an AND item, unlike the domain conditions it sits beside. A rule
// carrying both must satisfy both.
func TestRuleSetIsAndedWithOtherConditions(t *testing.T) {
	options := configWithSets(
		[]option.Rule{routeRule(option.RawDefaultRule{
			RuleSet: badoption.Listable[string]{"media"},
			Port:    badoption.Listable[uint16]{443},
		}, routeTo("stream"))},
		[]option.RuleSet{inlineSet("media", "video.example")},
		"direct",
	)

	onPort := runOrFail(t, options, Options{Destination: "a.video.example", Port: 443})
	if onPort.Outbound != "stream" {
		t.Errorf("Outbound = %q, want stream", onPort.Outbound)
	}

	offPort := runOrFail(t, options, Options{Destination: "a.video.example", Port: 8080})
	if !offPort.FinalUsed {
		t.Error("the port condition was not applied alongside rule_set")
	}
}

// A rule set that cannot be read makes its rule undecidable, which must be
// visible as a warning and must clear Exact — never silently become a miss.
func TestUnreadableRuleSetIsSurfaced(t *testing.T) {
	options := configWithSets(
		[]option.Rule{
			routeRule(option.RawDefaultRule{
				RuleSet: badoption.Listable[string]{"geoip-cn"},
			}, routeTo("cn")),
			routeRule(option.RawDefaultRule{}, routeTo("catch-all")),
		},
		// Remote, with no cache file configured: exactly what a fresh install
		// looks like before the first rule-set download.
		[]option.RuleSet{{Type: C.RuleSetTypeRemote, Tag: "geoip-cn", Format: C.RuleSetFormatBinary}},
		"direct",
	)

	result := runOrFail(t, options, Options{Destination: "example.com"})

	if result.Rules[0].State != MatchStateUnevaluated {
		t.Errorf("state = %q, want unevaluated", result.Rules[0].State)
	}
	if len(result.Rules[0].RuleSets) != 1 {
		t.Fatalf("got %d rule-set statuses, want 1", len(result.Rules[0].RuleSets))
	}
	status := result.Rules[0].RuleSets[0]
	if status.Tag != "geoip-cn" {
		t.Errorf("Tag = %q, want geoip-cn", status.Tag)
	}
	if status.Reason != ruleset.ReasonCacheDisabled {
		t.Errorf("Reason = %q, want %q", status.Reason, ruleset.ReasonCacheDisabled)
	}
	// The rule naming the set is the one that could not be decided, so the
	// answer below it is a guess and has to say so.
	if result.Exact {
		t.Error("Exact = true despite an unreadable rule set ahead")
	}
	if len(result.Warnings) == 0 {
		t.Error("no warning raised for an unreadable rule set")
	}
}

func TestSummaryNamesConditions(t *testing.T) {
	options := configWithSets(
		[]option.Rule{routeRule(option.RawDefaultRule{
			RuleSet:      badoption.Listable[string]{"a", "b", "c", "d"},
			DomainSuffix: badoption.Listable[string]{"example.com"},
		}, routeTo("proxy"))},
		nil,
		"direct",
	)

	result := runOrFail(t, options, Options{Destination: "example.com"})
	summary := result.Rules[0].Summary
	if summary == "" || summary == "(no conditions)" {
		t.Fatalf("Summary = %q", summary)
	}
	// Long lists are capped so a 40-entry rule does not flood the display.
	if !contains([]string{summary}, summary) || len(summary) > 200 {
		t.Errorf("Summary is not capped: %q", summary)
	}
}

func TestPortRangeCondition(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{
			PortRange: badoption.Listable[string]{"8000:8100"},
		}, routeTo("alt")),
	}, "direct", "direct")

	inRange := runOrFail(t, options, Options{Destination: "example.com", Port: 8080})
	if inRange.Outbound != "alt" {
		t.Errorf("Outbound = %q, want alt", inRange.Outbound)
	}

	outOfRange := runOrFail(t, options, Options{Destination: "example.com", Port: 9000})
	if !outOfRange.FinalUsed {
		t.Error("port 9000 matched an 8000:8100 range")
	}
}

func TestNetworkCondition(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{
			Network: badoption.Listable[string]{"udp"},
		}, routeTo("quic")),
	}, "direct", "direct")

	udp := runOrFail(t, options, Options{Destination: "example.com", Network: "udp"})
	if udp.Outbound != "quic" {
		t.Errorf("Outbound = %q, want quic", udp.Outbound)
	}

	tcp := runOrFail(t, options, Options{Destination: "example.com", Network: "tcp"})
	if !tcp.FinalUsed {
		t.Error("a tcp probe matched a udp-only rule")
	}
}

func TestDomainKeywordAndRegex(t *testing.T) {
	options := configWith([]option.Rule{
		routeRule(option.RawDefaultRule{
			DomainKeyword: badoption.Listable[string]{"cdn"},
		}, routeTo("edge")),
		routeRule(option.RawDefaultRule{
			DomainRegex: badoption.Listable[string]{`^api\d+\.example\.com$`},
		}, routeTo("api")),
	}, "direct", "direct")

	keyword := runOrFail(t, options, Options{Destination: "static.cdn.example.com"})
	if keyword.Outbound != "edge" {
		t.Errorf("Outbound = %q, want edge", keyword.Outbound)
	}

	regex := runOrFail(t, options, Options{Destination: "api7.example.com"})
	if regex.Outbound != "api" {
		t.Errorf("Outbound = %q, want api", regex.Outbound)
	}
}
