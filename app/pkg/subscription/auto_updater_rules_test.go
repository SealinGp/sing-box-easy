package subscription

import (
	"os"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/noderules"
	"github.com/sagernet/sing-box/option"
)

// TestMain initializes the global logger so code paths that log (e.g. the
// node-rules rebuild) don't panic on a nil logger during tests.
func TestMain(m *testing.M) {
	logger.InitDefault()
	os.Exit(m.Run())
}

// fakeRules is an in-memory NodeRulesProvider for testing the rebuild path
// without a database.
type fakeRules struct {
	filters []*noderules.Filter
	groups  []*noderules.Group
}

func (f *fakeRules) ListFilters() ([]*noderules.Filter, error) { return f.filters, nil }
func (f *fakeRules) ListGroups() ([]*noderules.Group, error)   { return f.groups, nil }

func ep(tag string) config.Outbound {
	return config.Outbound{Tag: tag, Type: "trojan", Options: &option.TrojanOutboundOptions{}}
}

func groupMembers(t *testing.T, obs []config.Outbound, tag string) []string {
	t.Helper()
	for _, o := range obs {
		if o.Tag != tag {
			continue
		}
		switch opt := o.Options.(type) {
		case *option.SelectorOutboundOptions:
			return opt.Outbounds
		case *option.URLTestOutboundOptions:
			return opt.Outbounds
		}
		t.Fatalf("%q is not a group: %T", tag, o.Options)
	}
	return nil
}

func hasTag(obs []config.Outbound, tag string) bool {
	for _, o := range obs {
		if o.Tag == tag {
			return true
		}
	}
	return false
}

// TestRebuildNodeRules_AssignsAndBuilds verifies a full rebuild: endpoints are
// assigned to Filters (multi-match), the fallback collects leftovers, a Group
// references its Filters, and a user-authored selector is preserved.
func TestRebuildNodeRules_AssignsAndBuilds(t *testing.T) {
	au := &AutoUpdater{nodeRules: &fakeRules{
		filters: []*noderules.Filter{
			{ID: "f_asia", Name: "Asia", Priority: 10, OutboundType: "urltest", Matchers: []noderules.Matcher{
				{Type: noderules.MatcherCode, Value: "HK"}, {Type: noderules.MatcherCode, Value: "JP"},
			}},
			{ID: "f_stream", Name: "Streaming", Priority: 20, OutboundType: "selector", Matchers: []noderules.Matcher{
				{Type: noderules.MatcherKeyword, Value: "Streaming"},
			}},
			{ID: noderules.FallbackFilterID, Name: noderules.FallbackFilterName, IsFallback: true, Priority: noderules.FallbackPriority, OutboundType: "urltest"},
		},
		groups: []*noderules.Group{
			{ID: "g_all", Name: "All Regions", FilterIDs: []string{"f_asia", "f_stream", noderules.FallbackFilterID}},
		},
	}}

	outbounds := []config.Outbound{
		// user-authored selector must survive untouched.
		{Tag: "PROXY", Type: "selector", Options: &option.SelectorOutboundOptions{Outbounds: []string{"All Regions"}}},
		ep("🇭🇰 HK Streaming | sub_a"), // Asia (HK) + Streaming
		ep("日本 Tokyo | sub_a"),        // Asia (JP)
		ep("US West | sub_a"),         // -> Other
	}

	got, err := au.rebuildNodeRules(outbounds, "sub_a")
	if err != nil {
		t.Fatalf("rebuild error: %v", err)
	}

	// Asia urltest has both HK+JP nodes (sorted).
	if m := groupMembers(t, got, "Asia"); len(m) != 2 {
		t.Errorf("Asia members = %v, want 2", m)
	}
	// Streaming selector has the HK-Streaming node (multi-match).
	if m := groupMembers(t, got, "Streaming"); len(m) != 1 {
		t.Errorf("Streaming members = %v, want 1", m)
	}
	// Other collects the unmatched US node.
	if m := groupMembers(t, got, noderules.FallbackFilterName); len(m) != 1 || m[0] != "US West | sub_a" {
		t.Errorf("Other members = %v, want [US West | sub_a]", m)
	}
	// Group references all three live filters.
	if m := groupMembers(t, got, "All Regions"); len(m) != 3 {
		t.Errorf("group members = %v, want 3", m)
	}
	// User selector preserved.
	if !hasTag(got, "PROXY") {
		t.Error("user-authored 'PROXY' selector was dropped")
	}
}

// TestRebuildNodeRules_Idempotent verifies running the rebuild twice yields an
// identical outbound set (deterministic membership).
func TestRebuildNodeRules_Idempotent(t *testing.T) {
	provider := &fakeRules{
		filters: []*noderules.Filter{
			{ID: "f_asia", Name: "Asia", Priority: 10, OutboundType: "urltest", Matchers: []noderules.Matcher{{Type: noderules.MatcherCode, Value: "HK"}}},
			{ID: noderules.FallbackFilterID, Name: noderules.FallbackFilterName, IsFallback: true, Priority: noderules.FallbackPriority, OutboundType: "urltest"},
		},
	}
	au := &AutoUpdater{nodeRules: provider}
	outbounds := []config.Outbound{ep("🇭🇰 HK-1 | sub_a"), ep("🇭🇰 HK-2 | sub_a"), ep("US | sub_a")}

	first, err := au.rebuildNodeRules(outbounds, "sub_a")
	if err != nil {
		t.Fatalf("first rebuild: %v", err)
	}
	second, err := au.rebuildNodeRules(first, "sub_a")
	if err != nil {
		t.Fatalf("second rebuild: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("non-idempotent: first=%d second=%d outbounds", len(first), len(second))
	}
	for _, tag := range []string{"Asia", noderules.FallbackFilterName} {
		a, b := groupMembers(t, first, tag), groupMembers(t, second, tag)
		if len(a) != len(b) {
			t.Errorf("%q membership changed across rebuilds: %v vs %v", tag, a, b)
		}
	}
}

// TestRebuildNodeRules_NilProviderNoop verifies a nil rules provider leaves the
// outbounds unchanged (legacy path owns additions).
func TestRebuildNodeRules_NilProviderNoop(t *testing.T) {
	au := &AutoUpdater{}
	outbounds := []config.Outbound{ep("HK-1 | sub_a")}
	got, err := au.rebuildNodeRules(outbounds, "sub_a")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("nil provider should be a no-op, got %d outbounds", len(got))
	}
}
