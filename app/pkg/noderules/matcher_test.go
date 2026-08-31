package noderules

import (
	"slices"
	"testing"
)

func kw(v string) Matcher   { return Matcher{Type: MatcherKeyword, Value: v} }
func code(v string) Matcher { return Matcher{Type: MatcherCode, Value: v} }
func emo(v string) Matcher  { return Matcher{Type: MatcherEmoji, Value: v} }

func filter(id, name string, prio int, ms ...Matcher) *Filter {
	return &Filter{ID: id, Name: name, Priority: prio, Matchers: ms, OutboundType: OutboundTypeURLTest}
}

func fallback() *Filter {
	return &Filter{ID: FallbackFilterID, Name: FallbackFilterName, IsFallback: true, Priority: FallbackPriority}
}

// TestAssignFilters_MultiMatch verifies an endpoint joins EVERY matching Filter
// (the YouTube/Asia overlap case), not just the first.
func TestAssignFilters_MultiMatch(t *testing.T) {
	asia := filter("f_asia", "Asia", 10, code("HK"), code("JP"), code("KR"))
	youtube := filter("f_yt", "YouTube", 20, kw("Streaming"), code("HK"))
	other := fallback()

	tags := []string{
		"🇭🇰 HK Streaming 1.2.3.4:443 | sub_a", // matches Asia (HK) AND YouTube (HK + Streaming)
		"日本 Tokyo 5.6.7.8:443 | sub_a",        // matches Asia (JP) only
		"US West 9.9.9.9:443 | sub_a",         // matches nothing -> Other
	}

	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{asia, youtube, other})

	if want := []string{tags[0], tags[1]}; !slices.Equal(membership["f_asia"], want) {
		t.Errorf("Asia members = %v, want %v", membership["f_asia"], want)
	}
	if want := []string{tags[0]}; !slices.Equal(membership["f_yt"], want) {
		t.Errorf("YouTube members = %v, want %v", membership["f_yt"], want)
	}
	if want := []string{tags[2]}; !slices.Equal(membership[FallbackFilterID], want) {
		t.Errorf("Other members = %v, want %v", membership[FallbackFilterID], want)
	}
	if want := []string{tags[2]}; !slices.Equal(others, want) {
		t.Errorf("others = %v, want %v", others, want)
	}
}

// TestAssignFilters_Excludes verifies the deny-list: a node matched by a
// matcher (code "US") is kept OUT of the Filter when it also matches an exclude
// (keyword "relay_bwh_us1"), and — matching no other Filter — falls through to
// the fallback.
func TestAssignFilters_Excludes(t *testing.T) {
	us := filter("f_us", "US", 10, code("US"))
	us.Excludes = []Matcher{kw("relay_bwh_us1")}
	other := fallback()

	tags := []string{
		"relay_bwh_us1",              // matches US code, but excluded -> Other
		"🇺🇸 美国 04 host:36004 | sub_1", // matches US, not excluded -> US
		"relay_bwh_us2",             // matches US, not excluded -> US
	}
	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{us, other})

	if want := []string{tags[1], tags[2]}; !slices.Equal(membership["f_us"], want) {
		t.Errorf("US members = %v, want %v", membership["f_us"], want)
	}
	if want := []string{"relay_bwh_us1"}; !slices.Equal(membership[FallbackFilterID], want) {
		t.Errorf("fallback members = %v, want %v", membership[FallbackFilterID], want)
	}
	if want := []string{"relay_bwh_us1"}; !slices.Equal(others, want) {
		t.Errorf("others = %v, want %v", others, want)
	}
}

// TestAssignFilters_CodeSynonyms verifies a single `code` matcher catches
// Chinese, English, bare-code, and emoji spellings.
func TestAssignFilters_CodeSynonyms(t *testing.T) {
	hk := filter("f_hk", "HK", 10, code("HK"))
	other := fallback()

	tags := []string{
		"香港 01 | sub_a",
		"Hong Kong 02 | sub_a",
		"HK-03 | sub_a",
		"🇭🇰 04 | sub_a",
		"东京 05 | sub_a", // not HK -> Other
	}
	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{hk, other})

	if got := len(membership["f_hk"]); got != 4 {
		t.Errorf("HK matched %d tags, want 4 (%v)", got, membership["f_hk"])
	}
	if want := []string{"东京 05 | sub_a"}; !slices.Equal(others, want) {
		t.Errorf("others = %v, want %v", others, want)
	}
}

// TestAssignFilters_KeywordCollision documents that a bare keyword is a literal
// substring: "Korea" also matches "North Korea". Curated `code` matchers are the
// recommended way to avoid this; the test pins the documented behavior.
func TestAssignFilters_KeywordCollision(t *testing.T) {
	korea := filter("f_kr", "Korea", 10, kw("Korea"))
	other := fallback()

	tags := []string{"Seoul South Korea | sub_a", "Pyongyang North Korea | sub_a"}
	membership, _ := AssignFilters(NodePool{Endpoints: tags}, []*Filter{korea, other})

	if got := len(membership["f_kr"]); got != 2 {
		t.Errorf("bare keyword 'Korea' matched %d, want 2 (substring incl. North Korea)", got)
	}
}

// TestAssignFilters_EmojiMatcher verifies a literal emoji-flag matcher.
func TestAssignFilters_EmojiMatcher(t *testing.T) {
	hk := filter("f_hk", "HK", 10, emo("🇭🇰"))
	other := fallback()

	tags := []string{"🇭🇰 Node 1 | sub_a", "Plain JP | sub_a"}
	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{hk, other})

	if want := []string{"🇭🇰 Node 1 | sub_a"}; !slices.Equal(membership["f_hk"], want) {
		t.Errorf("HK emoji members = %v, want %v", membership["f_hk"], want)
	}
	if want := []string{"Plain JP | sub_a"}; !slices.Equal(others, want) {
		t.Errorf("others = %v, want %v", others, want)
	}
}

// TestAssignFilters_FallbackOnlyForZeroMatch ensures the fallback never absorbs
// a tag that already matched a real Filter.
func TestAssignFilters_FallbackOnlyForZeroMatch(t *testing.T) {
	jp := filter("f_jp", "Japan", 10, code("JP"))
	other := fallback()

	tags := []string{"日本 01 | sub_a", "Mars 02 | sub_a"}
	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{jp, other})

	if _, inOther := membership[FallbackFilterID]; len(membership[FallbackFilterID]) != 1 || !inOther {
		t.Errorf("fallback members = %v, want exactly [Mars]", membership[FallbackFilterID])
	}
	if !slices.Equal(others, []string{"Mars 02 | sub_a"}) {
		t.Errorf("others = %v, want [Mars]", others)
	}
}

// TestAssignFilters_NoFallback verifies graceful behavior when no fallback is
// configured (unmatched tags appear only in `others`).
func TestAssignFilters_NoFallback(t *testing.T) {
	jp := filter("f_jp", "Japan", 10, code("JP"))
	tags := []string{"日本 01 | sub_a", "Mars 02 | sub_a"}
	membership, others := AssignFilters(NodePool{Endpoints: tags}, []*Filter{jp})

	if _, ok := membership[FallbackFilterID]; ok {
		t.Errorf("no fallback configured, but fallback key present: %v", membership)
	}
	if !slices.Equal(others, []string{"Mars 02 | sub_a"}) {
		t.Errorf("others = %v, want [Mars]", others)
	}
}

// TestAssignFilters_OptInClaimedOnlyByExplicitMatch verifies the `direct`
// outbound case: an opt-in tag joins a Filter that names it, and is neither
// swallowed by the fallback nor reported as unmatched when nothing claims it.
func TestAssignFilters_OptInClaimedOnlyByExplicitMatch(t *testing.T) {
	bypass := filter("f_bypass", "Bypass", 10, kw("direct"))
	other := fallback()

	pool := NodePool{
		Endpoints: []string{"🇯🇵 JP 1.1.1.1:443"},
		OptIn:     []string{"direct", "direct-cn"},
	}

	membership, others := AssignFilters(pool, []*Filter{bypass, other})

	if want := []string{"direct", "direct-cn"}; !slices.Equal(membership["f_bypass"], want) {
		t.Errorf("Bypass members = %v, want %v", membership["f_bypass"], want)
	}
	if want := []string{pool.Endpoints[0]}; !slices.Equal(membership[FallbackFilterID], want) {
		t.Errorf("fallback members = %v, want %v (opt-in tags must not fall through)", membership[FallbackFilterID], want)
	}
	if want := []string{pool.Endpoints[0]}; !slices.Equal(others, want) {
		t.Errorf("others = %v, want %v", others, want)
	}
}

// TestAssignFilters_OptInNeverFallsBack pins that an unclaimed opt-in tag joins
// nothing at all — a urltest that acquired `direct` would elect it every probe.
func TestAssignFilters_OptInNeverFallsBack(t *testing.T) {
	other := fallback()

	membership, others := AssignFilters(NodePool{OptIn: []string{"direct"}}, []*Filter{other})

	if len(membership[FallbackFilterID]) != 0 {
		t.Errorf("fallback members = %v, want empty", membership[FallbackFilterID])
	}
	if len(others) != 0 {
		t.Errorf("others = %v, want empty", others)
	}
}
