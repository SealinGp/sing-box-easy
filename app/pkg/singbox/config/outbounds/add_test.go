package outbounds

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds/nodetag"
)

func TestFirstExistingTag(t *testing.T) {
	ob := config.Outbound{
		Tag:     "香港 09",
		Type:    "trojan",
		Options: map[string]any{"server": "s4.example.com", "server_port": 37219},
	}
	candidates := nodetag.OutboundTagCandidates("香港 09", ob)

	// Nothing stored yet → add it.
	if _, found := firstExistingTag(map[string]bool{}, candidates); found {
		t.Error("an empty config must not report the node as existing")
	}
	// Stored under the current shape → skip.
	if got, found := firstExistingTag(map[string]bool{candidates[0]: true}, candidates); !found || got != candidates[0] {
		t.Errorf("firstExistingTag = (%q, %v), want (%q, true)", got, found, candidates[0])
	}
	// Stored under the pre-fingerprint shape → still a skip, reported under the
	// name it is actually stored as so the response is not a fiction.
	if got, found := firstExistingTag(map[string]bool{candidates[1]: true}, candidates); !found || got != candidates[1] {
		t.Errorf("firstExistingTag = (%q, %v), want (%q, true)", got, found, candidates[1])
	}
	// A different node that merely shares the display name is NOT a duplicate.
	if _, found := firstExistingTag(map[string]bool{"香港 09": true}, candidates); found {
		t.Error("a bare display name must not count as the same node")
	}
}
