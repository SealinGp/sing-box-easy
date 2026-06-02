package subscription

import (
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

// TestDiffNodesPrefixOwnership verifies the subscription-ID prefix is the sole
// ownership signal: already-namespaced nodes are matched/deleted by prefix, and
// many distinct nodes can share one server:port (a relay/CDN endpoint) without
// collapsing onto one tag.
func TestDiffNodesPrefixOwnership(t *testing.T) {
	au := &AutoUpdater{}
	sub := &Subscription{ID: testSubID}

	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob(p("A relay:443"), "relay", 443),            // matches feed exactly → unchanged
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

	if len(toAdd) != 1 || toAdd[0].Tag != p("New relay:443") {
		t.Fatalf("toAdd = %v, want exactly [%s]", collectTags(toAdd), p("New relay:443"))
	}

	if _, ok := toDelete[p("A relay:443")]; ok {
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
	if got := toUpdate["Legacy relay:443"]; got.Tag != p("Legacy relay:443") {
		t.Errorf("legacy 'Legacy relay:443' -> %q, want %q", got.Tag, p("Legacy relay:443"))
	}
	if got := toUpdate["Bare"]; got.Tag != p("Bare relay:443") {
		t.Errorf("legacy 'Bare' -> %q, want %q", got.Tag, p("Bare relay:443"))
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
		ob(p("OldName relay:443"), "relay", 443),
	}

	newNodes := []*node.SubNode{
		sn("NewName", "relay", 443), // same endpoint, renamed -> "sub_test | NewName relay:443"
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, sub, newNodes)

	if len(toUpdate) != 0 {
		t.Fatalf("toUpdate = %v, want empty (a rename is delete+add, not update)", toUpdate)
	}
	if _, ok := toDelete[p("OldName relay:443")]; !ok || len(toDelete) != 1 {
		t.Fatalf("toDelete = %v, want exactly {%s}", toDelete, p("OldName relay:443"))
	}
	if len(toAdd) != 1 || toAdd[0].Tag != p("NewName relay:443") {
		t.Fatalf("toAdd = %v, want exactly [%s]", collectTags(toAdd), p("NewName relay:443"))
	}
}
