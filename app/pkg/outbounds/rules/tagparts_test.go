package noderules

import (
	"strings"
	"testing"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{
			"legacy host:port tag",
			"香港 09 s4.usghq.ps1ksydn.com:37219 | sub_1788165842",
			"香港 09",
		},
		{
			"fingerprint tag",
			"香港 09 a1b2c3d4 | sub_1788165842",
			"香港 09",
		},
		{
			"name containing pipes",
			"🇯🇵日本高速02|CTCU|0.5x cfyes.7770006.xyz:443 | sub_1778669376",
			"🇯🇵日本高速02|CTCU|0.5x",
		},
		{
			"bare host, no port",
			"🇩🇪 德国 example.com | sub_1",
			"🇩🇪 德国",
		},
		// Nothing recognizable is stripped: a hand-written tag has no machine
		// parts, and trimming its last word would narrow what matchers see.
		{"hand written tag", "relay_bwh_us1", "relay_bwh_us1"},
		{"plain name", "🇺🇸 美国 01", "🇺🇸 美国 01"},
		{"name ending in a number", "美国 高速 01", "美国 高速 01"},
		{"suffix only", "美国 01 | sub_1", "美国 01"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayName(tt.tag); got != tt.want {
				t.Errorf("displayName(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		haystack, needle string
		want             bool
	}{
		{"us", "us", true},
		{"us-01", "us", true},
		{"us_west", "us", true},
		{"hk01", "hk", true},
		{"01 us 02", "us", true},
		{"美国 us", "us", true},
		// The collisions that started this.
		{"s4.usghq.ps1ksydn.com:37219", "us", false},
		{"belarus 01", "us", false},
		{"cyprus 01", "us", false},
		{"singapore 01", "in", false},
		{"aws-link1.lxyun.xyz", "in", false},
		{"sweden 01", "de", false},
		{"macau 01", "au", false},
		{"", "us", false},
		{"us", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.haystack+"/"+tt.needle, func(t *testing.T) {
			if got := containsWord(tt.haystack, tt.needle); got != tt.want {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tt.haystack, tt.needle, got, tt.want)
			}
		})
	}
}

// The reported bug, end to end: a US/CA filter over the real tags that produced
// it. The negatives are the point — JP/SG/DE nodes from the same subscription
// were always correct, so a fix that just excluded that provider would be
// papering over the rule.
func TestCodeMatcherIgnoresServerHostname(t *testing.T) {
	ai := &Filter{
		ID:       "filter_ai",
		Name:     "🤖 AI",
		Matchers: []Matcher{{Type: MatcherCode, Value: "US"}, {Type: MatcherCode, Value: "CA"}},
	}
	tests := []struct {
		tag  string
		want bool
	}{
		{"香港 09 s4.usghq.ps1ksydn.com:37219 | sub_1788165842", false},
		{"台湾 02 s4.usghq.ps1ksydn.com:37232 | sub_1788165842", false},
		{"香港 09 a1b2c3d4 | sub_1788165842", false},
		{"美国 01 api-us-west1-a.treeslinks.com:37261 | sub_1788165842", true},
		{"🇺🇸 美国 04 gheianygh.owolist.cn:36004 | sub_1778669339", true},
		{"🇨🇦 加拿大 gheianygh.owolist.cn:37014 | sub_1778669339", true},
		{"日本 02 api-jp-01.treeslinks.com:37272 | sub_1788165842", false},
		{"新加坡 01 api-sg-09.treeslinks.com:37281 | sub_1788165842", false},
		{"德国 01 api-eu-west2-a.treeslinks.com:37291 | sub_1788165842", false},
		// A US node named in English still matches, boundaries and all.
		{"US-West 01 example.com:443 | sub_1", true},
		{"usa-01 example.com:443 | sub_1", true},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := filterClaims(ai, tt.tag, strings.ToLower(tt.tag))
			if got != tt.want {
				t.Errorf("filterClaims(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

// Keyword and emoji matchers keep seeing the WHOLE tag: operators write
// excludes that are full tags (server and subscription ID included), and an
// exclude that silently widened to every node of that name would quietly
// re-admit the node it was written to remove.
func TestKeywordMatchersStillSeeTheWholeTag(t *testing.T) {
	f := &Filter{
		ID:       "f",
		Matchers: []Matcher{{Type: MatcherCode, Value: "AU"}},
		Excludes: []Matcher{{
			Type:  MatcherKeyword,
			Value: "🇦🇺 澳大利亚 gheianygh.owolist.cn:37008 | sub_1778669339",
		}},
	}
	excluded := "🇦🇺 澳大利亚 gheianygh.owolist.cn:37008 | sub_1778669339"
	other := "🇦🇺 澳大利亚 other.example.com:37008 | sub_9999999999"

	if filterClaims(f, excluded, strings.ToLower(excluded)) {
		t.Error("the tag named by the exclude should stay excluded")
	}
	if !filterClaims(f, other, strings.ToLower(other)) {
		t.Error("a different AU node must not be caught by that exclude")
	}
}

// An exclude written against the pre-fingerprint tag format must keep excluding
// the same node after the format change. It matters more than it looks: an
// exclude that matches nothing fails silently, re-admitting the node the
// operator removed on purpose.
func TestLegacyExcludeStillMatchesAfterRetag(t *testing.T) {
	legacy := "🇦🇺 澳大利亚 gheianygh.owolist.cn:37008 | sub_1778669339"
	f := &Filter{
		ID:       "f",
		Matchers: []Matcher{{Type: MatcherCode, Value: "AU"}},
		Excludes: []Matcher{{Type: MatcherKeyword, Value: legacy}},
	}

	// The same node, re-tagged.
	fingerprint := legacyTagValue(legacy)
	if fingerprint == "" {
		t.Fatal("legacyTagValue returned empty for a legacy tag")
	}
	t.Logf("%q -> %q", legacy, fingerprint)

	if filterClaims(f, fingerprint, strings.ToLower(fingerprint)) {
		t.Error("re-tagged node should still be excluded")
	}
	// And the exclude must not widen to every AU node of that name.
	other := "🇦🇺 澳大利亚 deadbeef | sub_9999999999"
	if !filterClaims(f, other, strings.ToLower(other)) {
		t.Error("a different AU node must not be caught by that exclude")
	}
}

func TestLegacyTagValueIgnoresNonTags(t *testing.T) {
	for _, v := range []string{
		"香港",                    // a plain keyword
		"relay_bwh_us1",         // a hand-written tag
		"美国 deadbeef | sub_1",   // already fingerprinted
		"no separator host:443", // no ownership suffix
		"",
	} {
		if got := legacyTagValue(v); got != "" {
			t.Errorf("legacyTagValue(%q) = %q, want \"\"", v, got)
		}
	}
}
