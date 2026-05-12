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
