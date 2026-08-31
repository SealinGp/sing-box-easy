package subscription

import (
	"strconv"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
)

// opts builds an outbound options map with a server endpoint.
func opts(server string, port int) map[string]interface{} {
	return map[string]interface{}{"server": server, "server_port": port}
}

func ob(tag, server string, port int) config.Outbound {
	return config.Outbound{Tag: tag, Type: "trojan", Options: opts(server, port)}
}

func sn(name, server string, port int) *node.SubNode {
	return &node.SubNode{Tag: name, Type: "trojan", Options: opts(server, port)}
}

const testSubID = "sub_test"

// p suffixes a unique tag with the test subscription's namespace.
func p(uniqueTag string) string {
	return uniqueTag + subscriptionTagSuffix(testSubID)
}

// fp is the endpoint fingerprint the minted tags carry, resolved through the
// same helper the updater uses so the tests never hard-code a hash.
func fp(server string, port int) string {
	return config.FingerprintEndpointKey(config.GetOutboundServerKey(ob("x", server, port)))
}

// minted is the tag a feed node ends up with: "<name> <fingerprint> | <subID>".
func minted(name, server string, port int) string {
	return p(name + " " + fp(server, port))
}

// TestDiffNodesPrefixOwnership verifies the subscription-ID prefix is the sole
// ownership signal: already-namespaced nodes are matched/deleted by prefix, and
// many distinct nodes can share one server:port (a relay/CDN endpoint) without
// collapsing onto one tag.
func TestDiffNodesPrefixOwnership(t *testing.T) {
	au := &AutoUpdater{}
	sub := &Subscription{ID: testSubID}

	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob(minted("A", "relay", 443), "relay", 443),   // matches feed exactly → unchanged
		ob(p("Gone relay:443"), "relay", 443),         // prefixed but not in feed → deleted
		ob("Other other.host:443", "other.host", 443), // unrelated host, no prefix → untouched
	}

	newNodes := []*node.SubNode{
		sn("A", "relay", 443),   // -> "sub_test | A relay:443" (unchanged)
		sn("New", "relay", 443), // -> "sub_test | New relay:443" (added)
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, newNodes)

	if len(toDelete) != 1 {
		t.Fatalf("toDelete = %v, want exactly {%s}", toDelete, p("Gone relay:443"))
	}
	if _, ok := toDelete[p("Gone relay:443")]; !ok {
		t.Errorf("expected %q in toDelete, got %v", p("Gone relay:443"), toDelete)
	}

	if len(toUpdate) != 0 {
		t.Fatalf("toUpdate = %v, want empty (A is unchanged)", toUpdate)
	}

	if len(toAdd) != 1 || toAdd[0].Tag != minted("New", "relay", 443) {
		t.Fatalf("toAdd = %v, want exactly [%s]", collectTags(toAdd), minted("New", "relay", 443))
	}

	if _, ok := toDelete[minted("A", "relay", 443)]; ok {
		t.Error("prefixed 'A' should be unchanged, not deleted")
	}
	if _, ok := toDelete["Other other.host:443"]; ok {
		t.Error("unrelated host must not be deleted by this subscription's update")
	}
}

// TestDiffNodesLegacyMigration verifies un-prefixed outbounds (added before
// tag-prefixing or added manually) whose server is in the feed are re-tagged
// into the subscription namespace, preserving group memberships via a rename.
func TestDiffNodesLegacyMigration(t *testing.T) {
	au := &AutoUpdater{}
	sub := &Subscription{ID: testSubID}

	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob("Legacy relay:443", "relay", 443), // legacy with suffix → renamed into namespace
		ob("Bare", "relay", 443),             // legacy bare name → renamed into namespace
		ob("Stale relay:443", "relay", 443),  // legacy, host in feed but no match → deleted
	}

	newNodes := []*node.SubNode{
		sn("Legacy", "relay", 443), // -> "sub_test | Legacy relay:443"
		sn("Bare", "relay", 443),   // -> "sub_test | Bare relay:443"
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, newNodes)

	if len(toUpdate) != 2 {
		t.Fatalf("toUpdate = %v, want 2 migrations", toUpdate)
	}
	if got, want := toUpdate["Legacy relay:443"], minted("Legacy", "relay", 443); got.Tag != want {
		t.Errorf("legacy 'Legacy relay:443' -> %q, want %q", got.Tag, want)
	}
	if got, want := toUpdate["Bare"], minted("Bare", "relay", 443); got.Tag != want {
		t.Errorf("legacy 'Bare' -> %q, want %q", got.Tag, want)
	}

	if len(toDelete) != 1 {
		t.Fatalf("toDelete = %v, want exactly {Stale relay:443}", toDelete)
	}
	if _, ok := toDelete["Stale relay:443"]; !ok {
		t.Errorf("expected 'Stale relay:443' in toDelete, got %v", toDelete)
	}

	if len(toAdd) != 0 {
		t.Fatalf("toAdd = %v, want empty (both feed nodes matched legacy outbounds)", collectTags(toAdd))
	}
}

// TestDiffNodesNameChange verifies that a provider renaming a node is detected
// via the prefix: the old prefixed tag (absent from the feed) is deleted and the
// new prefixed tag is added.
func TestDiffNodesNameChange(t *testing.T) {
	au := &AutoUpdater{}
	sub := &Subscription{ID: testSubID}

	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob(minted("OldName", "relay", 443), "relay", 443),
	}

	newNodes := []*node.SubNode{
		sn("NewName", "relay", 443), // same endpoint, renamed -> "sub_test | NewName relay:443"
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, newNodes)

	if len(toUpdate) != 0 {
		t.Fatalf("toUpdate = %v, want empty (a rename is delete+add, not update)", toUpdate)
	}
	if _, ok := toDelete[minted("OldName", "relay", 443)]; !ok || len(toDelete) != 1 {
		t.Fatalf("toDelete = %v, want exactly {%s}", toDelete, minted("OldName", "relay", 443))
	}
	if len(toAdd) != 1 || toAdd[0].Tag != minted("NewName", "relay", 443) {
		t.Fatalf("toAdd = %v, want exactly [%s]", collectTags(toAdd), minted("NewName", "relay", 443))
	}
}

// The tag format change must land as a RENAME, not as delete-plus-add. A
// delete would strip the node from every selector/urltest that references it
// and re-add it bare, so a config that worked before the upgrade would come
// back with half-empty groups.
func TestDiffNodesMigratesLegacyEndpointTagsAsRenames(t *testing.T) {
	au := &AutoUpdater{}
	sub := &Subscription{ID: testSubID}

	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob(p("香港 09 s4.example.com:37219"), "s4.example.com", 37219),
		ob(p("美国 01 api.example.com:37261"), "api.example.com", 37261),
	}

	newNodes := []*node.SubNode{
		sn("香港 09", "s4.example.com", 37219),
		sn("美国 01", "api.example.com", 37261),
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, newNodes)

	if len(toDelete) != 0 {
		t.Fatalf("toDelete = %v, want empty — a format change is not a deletion", toDelete)
	}
	if len(toAdd) != 0 {
		t.Fatalf("toAdd = %v, want empty — the nodes already exist", collectTags(toAdd))
	}
	if len(toUpdate) != 2 {
		t.Fatalf("toUpdate = %v, want 2 renames", toUpdate)
	}
	for _, tc := range []struct {
		name, server string
		port         int
	}{
		{"香港 09", "s4.example.com", 37219},
		{"美国 01", "api.example.com", 37261},
	} {
		old := p(tc.name + " " + tc.server + ":" + itoa(tc.port))
		got, ok := toUpdate[old]
		if !ok {
			t.Errorf("no rename recorded for %q", old)
			continue
		}
		if want := minted(tc.name, tc.server, tc.port); got.Tag != want {
			t.Errorf("%q renamed to %q, want %q", old, got.Tag, want)
		}
	}
}

// A tag already in the current format must not be re-migrated (which would
// churn every refresh).
func TestFingerprintLegacyTagIgnoresCurrentFormat(t *testing.T) {
	current := minted("香港 09", "s4.example.com", 37219)
	if got := fingerprintLegacyTag(current, testSubID); got != "" {
		t.Errorf("fingerprintLegacyTag(%q) = %q, want \"\"", current, got)
	}
	// Nor a tag belonging to a different subscription.
	other := "香港 09 s4.example.com:37219 | sub_other"
	if got := fingerprintLegacyTag(other, testSubID); got != "" {
		t.Errorf("fingerprintLegacyTag(%q) = %q, want \"\"", other, got)
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
