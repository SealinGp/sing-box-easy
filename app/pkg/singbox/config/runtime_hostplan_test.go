package config

import (
	"os"
	"path/filepath"
	"testing"
)

// The OpenWrt host integration derives its plan from dns.servers and
// route.rules. GetRuntimeConfig must therefore decode them; a partial parse
// silently yields "no DNS redirect needed" and dnsmasq is never pointed at
// sing-box.
func TestGetRuntimeConfigDecodesDNSAndRouteForHostPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{
  "dns":{
    "servers":[{"type":"hosts","tag":"dns_lan","predefined":{"nas.tparts.com":"192.168.9.207"}}]
  },
  "inbounds":[
    {"type":"direct","tag":"dns-in","listen":"0.0.0.0","listen_port":5333,"network":"udp"},
    {"type":"tun","tag":"tun-in","interface_name":"tun0","auto_route":true}
  ],
  "route":{
    "rules":[{"inbound":"dns-in","action":"hijack-dns"}]
  }
}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewManager(configPath, "sing-box", "").GetRuntimeConfig()
	if err != nil {
		t.Fatalf("GetRuntimeConfig() error = %v", err)
	}
	if len(cfg.Route.Rules) != 1 {
		t.Fatalf("route.rules not decoded: got %d, want 1", len(cfg.Route.Rules))
	}
	if len(cfg.DNS.Servers) != 1 {
		t.Fatalf("dns.servers not decoded: got %d, want 1", len(cfg.DNS.Servers))
	}
}
