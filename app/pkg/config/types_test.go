package config

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

// TestGetOutboundServerKey_TypedOptions verifies that the server-key helpers
// work when Options is a typed sing-box option struct — the shape produced by
// the protocol parsers AND by the typed JSON registry when loading config.json.
//
// Before the fix, the helpers only matched `map[string]interface{}`, so typed
// structs silently returned "" — collapsing the subscription diff into a no-op
// (added/updated/deleted all zero regardless of input).
func TestGetOutboundServerKey_TypedOptions(t *testing.T) {
	tests := []struct {
		name        string
		outbound    Outbound
		wantKey     string
		wantServer  string
	}{
		{
			name: "vmess typed struct (parser output)",
			outbound: Outbound{
				Type: "vmess",
				Tag:  "test-vmess",
				Options: option.VMessOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "1.2.3.4",
						ServerPort: 443,
					},
					UUID: "00000000-0000-0000-0000-000000000000",
				},
			},
			wantKey:    "1.2.3.4:443",
			wantServer: "1.2.3.4",
		},
		{
			name: "shadowsocks typed struct",
			outbound: Outbound{
				Type: "shadowsocks",
				Tag:  "test-ss",
				Options: option.ShadowsocksOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "ss.example.com",
						ServerPort: 8388,
					},
					Method:   "aes-128-gcm",
					Password: "secret",
				},
			},
			wantKey:    "ss.example.com:8388",
			wantServer: "ss.example.com",
		},
		{
			name: "generic map (legacy callers)",
			outbound: Outbound{
				Type: "vmess",
				Options: map[string]interface{}{
					"server":      "10.0.0.1",
					"server_port": float64(1080),
				},
			},
			wantKey:    "10.0.0.1:1080",
			wantServer: "10.0.0.1",
		},
		{
			name: "nil options",
			outbound: Outbound{
				Type:    "direct",
				Options: nil,
			},
			wantKey:    "",
			wantServer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutboundServerKey(tt.outbound); got != tt.wantKey {
				t.Errorf("GetOutboundServerKey() = %q, want %q", got, tt.wantKey)
			}
			if got := GetOutboundServer(tt.outbound); got != tt.wantServer {
				t.Errorf("GetOutboundServer() = %q, want %q", got, tt.wantServer)
			}
		})
	}
}

func TestFingerprintEndpointKey(t *testing.T) {
	// Stable: the same endpoint must always produce the same token, or every
	// refresh would re-tag every node.
	first := FingerprintEndpointKey("s4.example.com:37219")
	if first != FingerprintEndpointKey("s4.example.com:37219") {
		t.Error("fingerprint is not stable across calls")
	}
	if len(first) != endpointFingerprintLen {
		t.Errorf("fingerprint %q has length %d, want %d", first, len(first), endpointFingerprintLen)
	}
	for _, r := range first {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Errorf("fingerprint %q is not lowercase hex", first)
			break
		}
	}
	// Distinct endpoints must not collide — ports included, since a provider's
	// nodes commonly differ only by port.
	if FingerprintEndpointKey("s4.example.com:37219") == FingerprintEndpointKey("s4.example.com:37220") {
		t.Error("endpoints differing only by port produced the same fingerprint")
	}
	if FingerprintEndpointKey("") != "" {
		t.Error("an empty key must produce an empty fingerprint, not a hash of nothing")
	}
}

func TestGenerateFingerprintedTag(t *testing.T) {
	ob := Outbound{
		Tag:     "香港 09",
		Type:    "trojan",
		Options: map[string]any{"server": "s4.example.com", "server_port": 37219},
	}
	got := GenerateFingerprintedTag("香港 09", ob)
	want := "香港 09 " + FingerprintEndpointKey("s4.example.com:37219")
	if got != want {
		t.Errorf("GenerateFingerprintedTag = %q, want %q", got, want)
	}
	// It must be shorter than the readable form it replaces — that is the
	// point of hashing rather than spelling the endpoint out.
	if readable := GenerateUniqueTag("香港 09", ob); len(got) >= len(readable) {
		t.Errorf("fingerprinted tag %q is not shorter than %q", got, readable)
	}

	// No endpoint (a group or a pseudo-outbound): the name passes through.
	bare := Outbound{Tag: "block", Type: "block"}
	if got := GenerateFingerprintedTag("block", bare); got != "block" {
		t.Errorf("GenerateFingerprintedTag(no endpoint) = %q, want %q", got, "block")
	}
}

func TestOutboundTagCandidates(t *testing.T) {
	ob := Outbound{
		Tag:     "香港 09",
		Type:    "trojan",
		Options: map[string]any{"server": "s4.example.com", "server_port": 37219},
	}

	got := OutboundTagCandidates("香港 09", ob)
	if len(got) != 2 {
		t.Fatalf("candidates = %v, want the minted tag plus the legacy form", got)
	}
	// The tag actually minted is always first.
	if want := GenerateFingerprintedTag("香港 09", ob); got[0] != want {
		t.Errorf("candidates[0] = %q, want %q", got[0], want)
	}
	// The legacy form is what an older build wrote, so re-adding a node added
	// then is still recognized as a duplicate instead of making a second copy.
	if want := GenerateUniqueTag("香港 09", ob); got[1] != want {
		t.Errorf("candidates[1] = %q, want %q", got[1], want)
	}

	// No endpoint: there is only one possible shape, and offering the name
	// twice would just be noise.
	bare := Outbound{Tag: "block", Type: "block"}
	if got := OutboundTagCandidates("block", bare); len(got) != 1 || got[0] != "block" {
		t.Errorf("candidates for an endpoint-less outbound = %v, want [block]", got)
	}
}

func TestFirstExistingTag(t *testing.T) {
	ob := Outbound{
		Tag:     "香港 09",
		Type:    "trojan",
		Options: map[string]any{"server": "s4.example.com", "server_port": 37219},
	}
	candidates := OutboundTagCandidates("香港 09", ob)

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
