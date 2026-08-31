package sublink

import (
	"encoding/json"
	"os"
	"testing"
)

// Manual: SBE_LIVE_SUB=<url> go test ./app/pkg/sublink -run LiveSubscription -v
func TestLiveSubscription(t *testing.T) {
	url := os.Getenv("SBE_LIVE_SUB")
	if url == "" {
		t.Skip("SBE_LIVE_SUB not set")
	}
	l := new(SubLink)
	nodes, meta, err := l.ListNodesWithMeta([]string{url})
	if err != nil {
		t.Fatalf("ListNodesWithMeta: %v", err)
	}
	t.Logf("userinfo: %q", meta.Userinfo)
	t.Logf("site url: %q", meta.SiteURL)
	t.Logf("nodes: %d", len(nodes))
	counts := map[string]int{}
	for _, n := range nodes {
		counts[n.Type]++
	}
	t.Logf("types: %v", counts)
	if len(nodes) > 0 {
		b, _ := json.Marshal(nodes[0])
		t.Logf("first: %s", b)
	}
}
