package config

import (
	"slices"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"
)

func endpoint(tag string) Outbound {
	return Outbound{Tag: tag, Type: "trojan", Options: &option.TrojanOutboundOptions{}}
}

func membersOf(t *testing.T, ob Outbound) []string {
	t.Helper()
	switch o := ob.Options.(type) {
	case *option.SelectorOutboundOptions:
		return o.Outbounds
	case *option.URLTestOutboundOptions:
		return o.Outbounds
	default:
		t.Fatalf("outbound %q is not a group: %T", ob.Tag, ob.Options)
		return nil
	}
}

func findOB(obs []Outbound, tag string) (Outbound, bool) {
	for _, o := range obs {
		if o.Tag == tag {
			return o, true
		}
	}
	return Outbound{}, false
}

// TestBuildGroupOutbounds_FreshBuild builds Filters + a Group from scratch and
// checks types, sorted members, and that endpoints are preserved.
func TestBuildGroupOutbounds_FreshBuild(t *testing.T) {
	existing := []Outbound{endpoint("HK-2"), endpoint("HK-1"), endpoint("JP-1")}

	filters := []FilterSpec{
		{Name: "Asia", OutboundType: "urltest", MemberTags: []string{"HK-2", "HK-1", "JP-1"}},
		{Name: "Other Nodes", OutboundType: "selector", MemberTags: []string{}}, // empty -> skipped
	}
	groups := []GroupSpec{
		{Name: "All", FilterNames: []string{"Asia", "Other Nodes"}}, // Other skipped -> only Asia
	}

	got := BuildGroupOutbounds(existing, filters, groups)

	// Endpoints preserved.
	for _, tag := range []string{"HK-1", "HK-2", "JP-1"} {
		if _, ok := findOB(got, tag); !ok {
			t.Errorf("endpoint %q dropped", tag)
		}
	}
	// Empty "Other Nodes" filter skipped.
	if _, ok := findOB(got, "Other Nodes"); ok {
		t.Error("empty filter 'Other Nodes' should be skipped")
	}
	// Asia urltest with sorted members.
	asia, ok := findOB(got, "Asia")
	if !ok || asia.Type != "urltest" {
		t.Fatalf("Asia missing or wrong type: %+v", asia)
	}
	if want := []string{"HK-1", "HK-2", "JP-1"}; !slices.Equal(membersOf(t, asia), want) {
		t.Errorf("Asia members = %v, want sorted %v", membersOf(t, asia), want)
	}
	// Group references only the live Asia filter.
	grp, ok := findOB(got, "All")
	if !ok || grp.Type != "selector" {
		t.Fatalf("group 'All' missing or wrong type")
	}
	if want := []string{"Asia"}; !slices.Equal(membersOf(t, grp), want) {
		t.Errorf("group members = %v, want %v (Other skipped)", membersOf(t, grp), want)
	}
}

// TestBuildGroupOutbounds_RebuildInPlace verifies a previously-generated group
// is rebuilt at its original position and empty ones are removed.
func TestBuildGroupOutbounds_RebuildInPlace(t *testing.T) {
	existing := []Outbound{
		{Tag: "Asia", Type: "urltest", Options: &option.URLTestOutboundOptions{Outbounds: []string{"OLD"}}},
		endpoint("HK-1"),
		{Tag: "Empty", Type: "urltest", Options: &option.URLTestOutboundOptions{Outbounds: []string{"GONE"}}},
		endpoint("JP-1"),
	}
	filters := []FilterSpec{
		{Name: "Asia", OutboundType: "urltest", MemberTags: []string{"HK-1", "JP-1"}},
		{Name: "Empty", OutboundType: "urltest", MemberTags: []string{}}, // now empty -> removed
	}

	got := BuildGroupOutbounds(existing, filters, nil)

	// Asia rebuilt in place (index 0) with fresh members.
	if got[0].Tag != "Asia" {
		t.Errorf("Asia should stay at index 0, got %q", got[0].Tag)
	}
	if want := []string{"HK-1", "JP-1"}; !slices.Equal(membersOf(t, got[0]), want) {
		t.Errorf("Asia rebuilt members = %v, want %v", membersOf(t, got[0]), want)
	}
	// Empty group removed.
	if _, ok := findOB(got, "Empty"); ok {
		t.Error("now-empty 'Empty' group should be removed")
	}
}

// TestBuildGroupOutbounds_URLTestSettings verifies urltest health-check fields
// (url/interval/tolerance) are attached to the generated urltest outbound.
func TestBuildGroupOutbounds_URLTestSettings(t *testing.T) {
	existing := []Outbound{endpoint("HK-1")}
	got := BuildGroupOutbounds(existing, []FilterSpec{
		{
			Name:          "Asia",
			OutboundType:  "urltest",
			MemberTags:    []string{"HK-1"},
			TestURL:       "http://www.gstatic.com/generate_204",
			TestInterval:  "10s",
			TestTolerance: 200,
		},
	}, nil)

	asia, ok := findOB(got, "Asia")
	if !ok {
		t.Fatal("Asia not built")
	}
	opts, ok := asia.Options.(*option.URLTestOutboundOptions)
	if !ok {
		t.Fatalf("Asia options type = %T, want *URLTestOutboundOptions", asia.Options)
	}
	if opts.URL != "http://www.gstatic.com/generate_204" {
		t.Errorf("url = %q", opts.URL)
	}
	if time.Duration(opts.Interval) != 10*time.Second {
		t.Errorf("interval = %v, want 10s", time.Duration(opts.Interval))
	}
	if opts.Tolerance != 200 {
		t.Errorf("tolerance = %d, want 200", opts.Tolerance)
	}
}

// TestBuildGroupOutbounds_Immutability ensures inputs are not mutated.
func TestBuildGroupOutbounds_Immutability(t *testing.T) {
	origMembers := []string{"OLD"}
	existing := []Outbound{
		{Tag: "Asia", Type: "urltest", Options: &option.URLTestOutboundOptions{Outbounds: origMembers}},
		endpoint("HK-1"),
	}
	_ = BuildGroupOutbounds(existing, []FilterSpec{
		{Name: "Asia", OutboundType: "urltest", MemberTags: []string{"HK-1"}},
	}, nil)

	opts := existing[0].Options.(*option.URLTestOutboundOptions)
	if !slices.Equal(opts.Outbounds, []string{"OLD"}) {
		t.Errorf("input group was mutated: %v", opts.Outbounds)
	}
}
