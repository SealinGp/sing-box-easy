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

// TestDiffNodesSharedEndpoint is the regression test for the duplicate-tag bug:
// many distinct nodes can share one server:port (a relay/CDN endpoint). The diff
// must key on the unique tag, not server:port, so co-located nodes are handled
// independently instead of collapsing onto one (which produced 19 identical tags
// and failed `sing-box check`).
func TestDiffNodesSharedEndpoint(t *testing.T) {
	au := &AutoUpdater{}

	// Three existing outbounds behind the SAME endpoint relay:443, plus one
	// unrelated node on a different host that must stay untouched.
	cfg := &config.SingBoxConfig{}
	cfg.Outbounds = []config.Outbound{
		ob("A relay:443", "relay", 443),               // matches feed exactly → unchanged
		ob("Legacy", "relay", 443),                    // legacy (no suffix) → renamed in place
		ob("Gone relay:443", "relay", 443),            // not in feed → deleted
		ob("Other other.host:443", "other.host", 443), // different host → untouched
	}

	newNodes := []*node.SubNode{
		sn("A", "relay", 443),      // -> "A relay:443" (unchanged)
		sn("Legacy", "relay", 443), // -> "Legacy relay:443" (rename of "Legacy")
		sn("New", "relay", 443),    // -> "New relay:443" (added)
	}

	toDelete, toAdd, toUpdate := au.diffNodes(cfg, newNodes)

	// Exactly the stale node is deleted.
	if len(toDelete) != 1 {
		t.Fatalf("toDelete = %v, want exactly {Gone relay:443}", toDelete)
	}
	if _, ok := toDelete["Gone relay:443"]; !ok {
		t.Errorf("expected 'Gone relay:443' in toDelete, got %v", toDelete)
	}

	// The legacy node is renamed in place (keyed by its existing tag).
	if len(toUpdate) != 1 {
		t.Fatalf("toUpdate = %v, want exactly {Legacy -> Legacy relay:443}", toUpdate)
	}
	updated, ok := toUpdate["Legacy"]
	if !ok {
		t.Errorf("expected 'Legacy' key in toUpdate, got %v", toUpdate)
	} else if updated.Tag != "Legacy relay:443" {
		t.Errorf("legacy rename target tag = %q, want 'Legacy relay:443'", updated.Tag)
	}

	// Only the genuinely-new node is added.
	if len(toAdd) != 1 || toAdd[0].Tag != "New relay:443" {
		t.Fatalf("toAdd = %v, want exactly [New relay:443]", collectTags(toAdd))
	}

	// "A relay:443" must NOT be in any change set (matched + unchanged), and the
	// unrelated host must never be considered. This is what the old server:port
	// keying got wrong (it collapsed all relay:443 nodes together).
	if _, ok := toDelete["A relay:443"]; ok {
		t.Error("'A relay:443' should be unchanged, not deleted")
	}
	if _, ok := toUpdate["A relay:443"]; ok {
		t.Error("'A relay:443' should be unchanged, not updated")
	}
	if _, ok := toDelete["Other other.host:443"]; ok {
		t.Error("unrelated host must not be deleted by this subscription's update")
	}
}
