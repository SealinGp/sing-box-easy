package config

import (
	"slices"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func makeSelector(tag string, members []string, def string) Outbound {
	return Outbound{
		Tag:  tag,
		Type: "selector",
		Options: &option.SelectorOutboundOptions{
			Outbounds: members,
			Default:   def,
		},
	}
}

func makeURLTest(tag string, members []string) Outbound {
	return Outbound{
		Tag:  tag,
		Type: "urltest",
		Options: &option.URLTestOutboundOptions{
			Outbounds: members,
		},
	}
}

func selectorMembers(t *testing.T, ob Outbound) []string {
	t.Helper()
	opts, ok := ob.Options.(*option.SelectorOutboundOptions)
	if !ok {
		t.Fatalf("expected SelectorOutboundOptions, got %T", ob.Options)
	}
	return opts.Outbounds
}

func selectorDefault(t *testing.T, ob Outbound) string {
	t.Helper()
	opts, ok := ob.Options.(*option.SelectorOutboundOptions)
	if !ok {
		t.Fatalf("expected SelectorOutboundOptions, got %T", ob.Options)
	}
	return opts.Default
}

func urlTestMembers(t *testing.T, ob Outbound) []string {
	t.Helper()
	opts, ok := ob.Options.(*option.URLTestOutboundOptions)
	if !ok {
		t.Fatalf("expected URLTestOutboundOptions, got %T", ob.Options)
	}
	return opts.Outbounds
}

// TestPruneGroupReferences_RemovesDeletedTagsFromSelector covers the bug we
// were tracking: a subscription delete left the deleted node tag inside
// selector/urltest "outbounds" lists, even though `sing-box check` passed.
func TestPruneGroupReferences_RemovesDeletedTagsFromSelector(t *testing.T) {
	outbounds := []Outbound{
		makeSelector("manual", []string{"node-a", "node-b", "node-c"}, "node-b"),
		makeURLTest("auto", []string{"node-a", "node-b"}),
	}

	deleted := map[string]struct{}{
		"node-b": {},
	}

	got := PruneGroupReferences(outbounds, deleted, nil, nil)

	if want := []string{"node-a", "node-c"}; !slices.Equal(selectorMembers(t, got[0]), want) {
		t.Errorf("selector members = %v, want %v", selectorMembers(t, got[0]), want)
	}
	// default pointed at deleted tag -> cleared
	if def := selectorDefault(t, got[0]); def != "" {
		t.Errorf("selector default = %q, want empty (was a deleted tag)", def)
	}
	if want := []string{"node-a"}; !slices.Equal(urlTestMembers(t, got[1]), want) {
		t.Errorf("urltest members = %v, want %v", urlTestMembers(t, got[1]), want)
	}
}

// TestPruneGroupReferences_DoesNotMutateInput verifies the immutability
// contract: callers can keep referencing the original outbound slice and
// option structs after the helper returns.
func TestPruneGroupReferences_DoesNotMutateInput(t *testing.T) {
	original := makeSelector("manual", []string{"node-a", "node-b"}, "node-b")
	outbounds := []Outbound{original}

	_ = PruneGroupReferences(outbounds, map[string]struct{}{"node-b": {}}, nil, nil)

	opts := original.Options.(*option.SelectorOutboundOptions)
	if !slices.Equal(opts.Outbounds, []string{"node-a", "node-b"}) {
		t.Errorf("input selector outbounds were mutated: %v", opts.Outbounds)
	}
	if opts.Default != "node-b" {
		t.Errorf("input selector default was mutated: %q", opts.Default)
	}
}

// TestPruneGroupReferences_RenameRewrites covers the subscription update case
// where a node's server endpoint is unchanged but the human-facing tag was
// changed by the upstream provider.
func TestPruneGroupReferences_RenameRewrites(t *testing.T) {
	outbounds := []Outbound{
		makeSelector("manual", []string{"old-tag", "node-other"}, "old-tag"),
		makeURLTest("auto", []string{"old-tag"}),
	}

	rename := map[string]string{"old-tag": "new-tag"}

	got := PruneGroupReferences(outbounds, nil, rename, nil)

	if want := []string{"new-tag", "node-other"}; !slices.Equal(selectorMembers(t, got[0]), want) {
		t.Errorf("selector members = %v, want %v", selectorMembers(t, got[0]), want)
	}
	if def := selectorDefault(t, got[0]); def != "new-tag" {
		t.Errorf("selector default = %q, want %q", def, "new-tag")
	}
	if want := []string{"new-tag"}; !slices.Equal(urlTestMembers(t, got[1]), want) {
		t.Errorf("urltest members = %v, want %v", urlTestMembers(t, got[1]), want)
	}
}

// TestPruneGroupReferences_RenameThenDelete covers the precedence rule: rename
// is applied first, then delete. If `old` is renamed to `new` and `new` is
// deleted, the reference is removed.
func TestPruneGroupReferences_RenameThenDelete(t *testing.T) {
	outbounds := []Outbound{
		makeSelector("manual", []string{"old", "keep"}, "old"),
	}

	got := PruneGroupReferences(outbounds,
		map[string]struct{}{"new": {}},
		map[string]string{"old": "new"},
		nil,
	)

	if want := []string{"keep"}; !slices.Equal(selectorMembers(t, got[0]), want) {
		t.Errorf("selector members = %v, want %v", selectorMembers(t, got[0]), want)
	}
	if def := selectorDefault(t, got[0]); def != "" {
		t.Errorf("selector default = %q, want empty (renamed-then-deleted)", def)
	}
}

// TestPruneGroupReferences_NoopWhenEmpty preserves the original slice when
// there's nothing to do — keeps callers cheap on the happy path.
func TestPruneGroupReferences_NoopWhenEmpty(t *testing.T) {
	outbounds := []Outbound{
		makeSelector("manual", []string{"a", "b"}, "a"),
	}
	got := PruneGroupReferences(outbounds, nil, nil, nil)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	// Should reuse the same option pointer when nothing changed.
	if got[0].Options != outbounds[0].Options {
		t.Errorf("expected option pointer reuse on no-op")
	}
}

// TestPruneGroupReferences_DropsDuplicatesAfterRename guards against a rename
// collision producing a duplicate tag in the group members list.
func TestPruneGroupReferences_DropsDuplicatesAfterRename(t *testing.T) {
	outbounds := []Outbound{
		makeSelector("manual", []string{"old", "new"}, "new"),
	}
	got := PruneGroupReferences(outbounds, nil, map[string]string{"old": "new"}, nil)

	if want := []string{"new"}; !slices.Equal(selectorMembers(t, got[0]), want) {
		t.Errorf("selector members = %v, want %v (rename collision)", selectorMembers(t, got[0]), want)
	}
}

// TestPruneGroupReferences_LeavesNonGroupOutboundsAlone is a sanity check that
// the helper doesn't touch protocol outbounds (vmess, trojan, etc.).
func TestPruneGroupReferences_LeavesNonGroupOutboundsAlone(t *testing.T) {
	other := Outbound{
		Tag:     "node-a",
		Type:    "vmess",
		Options: &option.VMessOutboundOptions{},
	}
	outbounds := []Outbound{
		other,
		makeSelector("manual", []string{"node-a", "gone"}, ""),
	}

	got := PruneGroupReferences(outbounds, map[string]struct{}{"gone": {}}, nil, nil)

	if got[0].Options != other.Options {
		t.Errorf("non-group outbound options were replaced")
	}
}

// TestPruneGroupReferences_AddsNewNodesToCollectionsOnly verifies that
// freshly-added node tags are appended to node *collections* (flat aggregates
// of final exit nodes) but NOT to node *groups* (curated lists of other
// selectors, identified by the "分组"/"group" name marker).
func TestPruneGroupReferences_AddsNewNodesToCollectionsOnly(t *testing.T) {
	outbounds := []Outbound{
		// Collection: flat list of nodes → should receive the new nodes.
		makeURLTest("♻️ 自动选择", []string{"node-a"}),
		// Collection: another flat list → should also receive them.
		makeSelector("🌍 其他节点", []string{"node-a"}, "node-a"),
		// Group: curates other selectors → must NOT receive raw nodes.
		makeSelector("🎬 流媒体分组", []string{"♻️ 自动选择", "🌍 其他节点"}, "♻️ 自动选择"),
	}

	addTags := []string{"node-new1", "node-new2", "node-a"} // node-a already present → deduped

	got := PruneGroupReferences(outbounds, nil, nil, addTags)

	if want := []string{"node-a", "node-new1", "node-new2"}; !slices.Equal(urlTestMembers(t, got[0]), want) {
		t.Errorf("collection urltest members = %v, want %v", urlTestMembers(t, got[0]), want)
	}
	if want := []string{"node-a", "node-new1", "node-new2"}; !slices.Equal(selectorMembers(t, got[1]), want) {
		t.Errorf("collection selector members = %v, want %v", selectorMembers(t, got[1]), want)
	}
	// The group keeps exactly its curated members — no raw nodes added.
	if want := []string{"♻️ 自动选择", "🌍 其他节点"}; !slices.Equal(selectorMembers(t, got[2]), want) {
		t.Errorf("group members = %v, want %v (groups must not receive raw nodes)", selectorMembers(t, got[2]), want)
	}
}
