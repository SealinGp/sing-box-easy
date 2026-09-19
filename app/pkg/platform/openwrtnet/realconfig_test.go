package openwrtnet

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

// End-to-end over the shape a real OpenWrt gateway runs: a direct inbound named
// by a hijack-dns route rule must produce a DNS redirect plan.
func TestDerivePlanFromRuntimeConfigNeedsDNSRedirect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{
  "dns":{"servers":[{"type":"hosts","tag":"dns_lan","predefined":{"nas.tparts.com":"192.168.9.207"}}]},
  "inbounds":[
    {"type":"direct","tag":"dns-in","listen":"0.0.0.0","listen_port":5333,"network":"udp"},
    {"type":"tun","tag":"tun-in","interface_name":"tun0","auto_route":true}
  ],
  "route":{"rules":[{"inbound":"dns-in","action":"hijack-dns"}]}
}`), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.NewManager(path, "sing-box", "").GetRuntimeConfig()
	if err != nil {
		t.Fatalf("GetRuntimeConfig() error = %v", err)
	}
	plan := DerivePlan(cfg)
	t.Logf("plan: tun=%q dns=%s hijack=%v lan=%v",
		plan.TunInterface, plan.DNSUpstream(), plan.NeedsDNSRedirect(), plan.LANDomains)

	if plan.TunInterface != "tun0" {
		t.Errorf("TunInterface = %q, want tun0", plan.TunInterface)
	}
	if !plan.NeedsDNSRedirect() {
		t.Error("NeedsDNSRedirect() = false, want true")
	}
	if got := plan.DNSUpstream(); got != "127.0.0.1#5333" {
		t.Errorf("DNSUpstream() = %q, want 127.0.0.1#5333", got)
	}
	if !plan.NeedsRebindAllow() {
		t.Error("NeedsRebindAllow() = false, want true")
	}
}
