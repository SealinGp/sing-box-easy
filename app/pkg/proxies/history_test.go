package proxies

import (
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

func TestBuildViewKeepsNodeHistorySeparateFromGroups(t *testing.T) {
	nodeHistory := []clashapi.DelaySample{{Time: time.Now(), Delay: 90}}
	groupHistory := []clashapi.DelaySample{{Time: time.Now(), Delay: 999}}
	view := BuildView([]clashapi.Proxy{
		{Name: "selector", Type: "Selector", Now: "node", All: []string{"node", "nested", "missing"}},
		{Name: "nested", Type: "URLTest", Now: "node", All: []string{"node"}, History: groupHistory},
		{Name: "node", Type: "VLESS", Delay: 90, History: nodeHistory},
	}, false)
	members := view.Groups[0].Members
	if len(members[0].History) != 1 || members[0].History[0].Delay != 90 {
		t.Fatalf("node history lost: %+v", members[0])
	}
	if len(members[1].History) != 0 || len(members[2].History) != 0 {
		t.Fatalf("group/missing node has misleading history: %+v", members)
	}
}
