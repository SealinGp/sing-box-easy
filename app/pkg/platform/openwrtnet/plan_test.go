package openwrtnet

import (
	"encoding/json"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func mustConfig(t *testing.T, raw string) *config.SingBoxConfig {
	t.Helper()
	var cfg config.SingBoxConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	return &cfg
}

func TestDerivePlan(t *testing.T) {
	t.Run("tun plus a hijack-dns inbound", func(t *testing.T) {
		// The shape this feature exists for: a TUN gateway whose DNS is
		// hijacked into sing-box on a loopback port.
		cfg := mustConfig(t, `{
			"inbounds": [
				{ "type": "direct", "tag": "dns-in", "listen": "127.0.0.1", "listen_port": 5333 },
				{ "type": "tun", "tag": "tun-in", "interface_name": "tun0",
				  "address": ["172.16.250.1/30"], "auto_route": true }
			],
			"outbounds": [ { "type": "direct", "tag": "direct" } ],
			"route": { "rules": [ { "inbound": "dns-in", "action": "hijack-dns" } ] }
		}`)

		got := DerivePlan(cfg)
		if got.TunInterface != "tun0" {
			t.Errorf("TunInterface = %q, want tun0", got.TunInterface)
		}
		if got.DNSAddress != "127.0.0.1" || got.DNSPort != 5333 {
			t.Errorf("DNS = %s:%d, want 127.0.0.1:5333", got.DNSAddress, got.DNSPort)
		}
		if !got.NeedsFirewallZone() || !got.NeedsDNSRedirect() {
			t.Errorf("plan %+v should need both zone and dns redirect", got)
		}
	})

	t.Run("tun without interface_name falls back to sing-box's default", func(t *testing.T) {
		cfg := mustConfig(t, `{
			"inbounds": [ { "type": "tun", "tag": "tun-in", "address": ["172.16.250.1/30"] } ],
			"outbounds": [ { "type": "direct", "tag": "direct" } ]
		}`)
		if got := DerivePlan(cfg); got.TunInterface != defaultTunInterface {
			t.Errorf("TunInterface = %q, want %q", got.TunInterface, defaultTunInterface)
		}
	})

	t.Run("hijack rule with an inbound list", func(t *testing.T) {
		cfg := mustConfig(t, `{
			"inbounds": [ { "type": "direct", "tag": "dns-in", "listen": "127.0.0.1", "listen_port": 5353 } ],
			"outbounds": [ { "type": "direct", "tag": "direct" } ],
			"route": { "rules": [ { "inbound": ["other", "dns-in"], "action": "hijack-dns" } ] }
		}`)
		got := DerivePlan(cfg)
		if got.DNSPort != 5353 {
			t.Errorf("DNSPort = %d, want 5353", got.DNSPort)
		}
	})

	t.Run("a DNS inbound with no hijack rule is not ours to redirect", func(t *testing.T) {
		// Without hijack-dns the inbound is a plain forwarder, not sing-box's
		// resolver — pointing dnsmasq at it would be wrong.
		cfg := mustConfig(t, `{
			"inbounds": [ { "type": "direct", "tag": "dns-in", "listen": "127.0.0.1", "listen_port": 5333 } ],
			"outbounds": [ { "type": "direct", "tag": "direct" } ]
		}`)
		if got := DerivePlan(cfg); got.NeedsDNSRedirect() {
			t.Errorf("NeedsDNSRedirect() = true without a hijack-dns rule (%+v)", got)
		}
	})

	t.Run("no tun, no dns", func(t *testing.T) {
		cfg := mustConfig(t, `{
			"inbounds": [ { "type": "mixed", "tag": "mixed-in", "listen": "::", "listen_port": 5330 } ],
			"outbounds": [ { "type": "direct", "tag": "direct" } ]
		}`)
		got := DerivePlan(cfg)
		if got.NeedsFirewallZone() || got.NeedsDNSRedirect() {
			t.Errorf("plan %+v should need nothing for a proxy-only config", got)
		}
	})

	t.Run("nil config is inert", func(t *testing.T) {
		if got := DerivePlan(nil); got.NeedsFirewallZone() || got.NeedsDNSRedirect() {
			t.Errorf("DerivePlan(nil) = %+v, want an empty plan", got)
		}
	})
}

func TestPlanDNSUpstream(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
		want string
	}{
		{
			name: "loopback",
			plan: Plan{DNSAddress: "127.0.0.1", DNSPort: 5333, hijack: true},
			want: "127.0.0.1#5333",
		},
		{
			// A wildcard listen is not a usable upstream address for dnsmasq;
			// dial loopback instead of "0.0.0.0#5333".
			name: "wildcard listen becomes loopback",
			plan: Plan{DNSAddress: "0.0.0.0", DNSPort: 5333, hijack: true},
			want: "127.0.0.1#5333",
		},
		{
			name: "ipv6 wildcard becomes loopback",
			plan: Plan{DNSAddress: "::", DNSPort: 5333, hijack: true},
			want: "127.0.0.1#5333",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.DNSUpstream(); got != tt.want {
				t.Errorf("DNSUpstream() = %q, want %q", got, tt.want)
			}
		})
	}
}
