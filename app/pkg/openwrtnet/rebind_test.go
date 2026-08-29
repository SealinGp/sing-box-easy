package openwrtnet

import (
	"path/filepath"
	"strings"
	"testing"
)

// lanHostsConfig is the shape this feature exists for: a `hosts` DNS server
// mapping LAN names to RFC1918 addresses, reached through a domain rule.
const lanHostsConfig = `{
	"dns": {
		"servers": [
			{ "type": "udp", "tag": "dns-china", "server": "223.5.5.5" },
			{ "type": "hosts", "tag": "dns_lan", "predefined": {
				"home.tparts.com": "192.168.9.207",
				"nas.tparts.com": "192.168.9.207",
				"keel.tparts.com": "192.168.9.27"
			} }
		],
		"rules": [ { "domain": ["home.tparts.com"], "server": "dns_lan" } ]
	},
	"inbounds": [
		{ "type": "direct", "tag": "dns-in", "listen": "0.0.0.0", "listen_port": 5333 },
		{ "type": "tun", "tag": "tun-in", "interface_name": "tun0", "address": ["172.16.250.1/30"] }
	],
	"outbounds": [ { "type": "direct", "tag": "direct" } ],
	"route": { "rules": [ { "inbound": "dns-in", "action": "hijack-dns" } ] }
}`

func TestDerivePlanCollectsLANDomains(t *testing.T) {
	plan := DerivePlan(mustConfig(t, lanHostsConfig))

	if !plan.NeedsRebindAllow() {
		t.Fatalf("plan %+v should need a rebind exemption", plan)
	}
	want := []string{"home.tparts.com", "keel.tparts.com", "nas.tparts.com"}
	if strings.Join(plan.LANDomains, ",") != strings.Join(want, ",") {
		t.Errorf("LANDomains = %v, want %v (sorted, deduped)", plan.LANDomains, want)
	}
}

func TestDerivePlanIgnoresPublicHostsEntries(t *testing.T) {
	// dnsmasq only filters private addresses. A hosts entry pinning a public
	// IP passes rebind protection untouched and must not widen the exemption.
	cfg := mustConfig(t, `{
		"dns": { "servers": [ { "type": "hosts", "tag": "pin", "predefined": {
			"cdn.example.com": "1.2.3.4"
		} } ] },
		"inbounds": [ { "type": "tun", "tag": "tun-in", "address": ["172.16.250.1/30"] } ],
		"outbounds": [ { "type": "direct", "tag": "direct" } ]
	}`)
	if plan := DerivePlan(cfg); plan.NeedsRebindAllow() {
		t.Errorf("LANDomains = %v, want none for a public address", plan.LANDomains)
	}
}

func TestDerivePlanReadsListValuedHostsEntries(t *testing.T) {
	// sing-box's predefined values are Listable: one address marshals as a
	// bare string, several as an array. Both shapes must be understood.
	cfg := mustConfig(t, `{
		"dns": { "servers": [ { "type": "hosts", "tag": "dns_lan", "predefined": {
			"dual.tparts.com": ["203.0.113.9", "192.168.9.207"]
		} } ] },
		"inbounds": [ { "type": "tun", "tag": "tun-in", "address": ["172.16.250.1/30"] } ],
		"outbounds": [ { "type": "direct", "tag": "direct" } ]
	}`)
	plan := DerivePlan(cfg)
	if len(plan.LANDomains) != 1 || plan.LANDomains[0] != "dual.tparts.com" {
		t.Errorf("LANDomains = %v, want [dual.tparts.com]", plan.LANDomains)
	}
}

func TestApplyExemptsLANDomainsFromRebindProtection(t *testing.T) {
	h := &fakeHost{dnsServer: "223.5.5.5\n"}
	m := newTestManager(t, h)
	plan := DerivePlan(mustConfig(t, lanHostsConfig))

	if err := m.Apply(plan); err != nil {
		t.Fatalf("Apply() = %v", err)
	}

	for _, want := range []string{
		"add_list dhcp.@dnsmasq[0].rebind_domain=home.tparts.com",
		"add_list dhcp.@dnsmasq[0].rebind_domain=keel.tparts.com",
		"add_list dhcp.@dnsmasq[0].rebind_domain=nas.tparts.com",
		"commit dhcp",
	} {
		if !h.ran(want) {
			t.Errorf("missing %q\ngot: %v", want, h.calls)
		}
	}

	st, err := m.loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if len(st.RebindDomains) != 3 {
		t.Errorf("state.RebindDomains = %v, want the three names recorded", st.RebindDomains)
	}
}

func TestApplyRestartsDNSMasqOnce(t *testing.T) {
	// The redirect and the exemption are two dhcp edits, but a LAN that loses
	// DNS twice on every start is a regression the operator will notice.
	h := &fakeHost{dnsServer: "223.5.5.5\n"}
	m := newTestManager(t, h)

	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatalf("Apply() = %v", err)
	}

	restarts := 0
	for _, c := range h.calls {
		if strings.Contains(c, "/etc/init.d/dnsmasq restart") {
			restarts++
		}
	}
	if restarts != 1 {
		t.Errorf("restarted dnsmasq %d times, want 1\ngot: %v", restarts, h.calls)
	}
}

func TestApplySkipsExemptionWhenProtectionIsOff(t *testing.T) {
	// Nothing is filtering the answers, so an exemption would be a leftover
	// with no reason behind it.
	h := &fakeHost{dnsServer: "223.5.5.5\n", rebindProtection: "0\n"}
	m := newTestManager(t, h)

	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if h.ran("rebind_domain=") {
		t.Errorf("wrote an exemption with protection disabled\ngot: %v", h.calls)
	}
}

func TestApplyLeavesOperatorRebindDomainsAlone(t *testing.T) {
	// A name the operator exempted by hand is already covered, and must not be
	// recorded as ours — Revert would then delete somebody else's entry.
	h := &fakeHost{dnsServer: "223.5.5.5\n", rebindDomain: "home.tparts.com\n"}
	m := newTestManager(t, h)

	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if h.ran("rebind_domain=home.tparts.com") {
		t.Errorf("re-added an existing exemption\ngot: %v", h.calls)
	}

	st, _ := m.loadState()
	for _, d := range st.RebindDomains {
		if d == "home.tparts.com" {
			t.Errorf("claimed the operator's entry as ours: %v", st.RebindDomains)
		}
	}
}

func TestRevertRemovesOnlyTheExemptionsWeAdded(t *testing.T) {
	h := &fakeHost{dnsServer: "223.5.5.5\n", rebindDomain: "corp.example.com\n"}
	m := newTestManager(t, h)
	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatal(err)
	}

	h2 := &fakeHost{rebindDomain: "corp.example.com home.tparts.com keel.tparts.com nas.tparts.com\n"}
	m.run = h2.run
	if err := m.Revert(); err != nil {
		t.Fatalf("Revert() = %v", err)
	}

	if !h2.ran("del_list dhcp.@dnsmasq[0].rebind_domain=home.tparts.com") {
		t.Errorf("did not remove our exemption\ngot: %v", h2.calls)
	}
	if h2.ran("rebind_domain=corp.example.com") {
		t.Errorf("removed the operator's exemption\ngot: %v", h2.calls)
	}
}

func TestRebindExemptionIsIdempotent(t *testing.T) {
	h := &fakeHost{dnsServer: "223.5.5.5\n"}
	m := newTestManager(t, h)
	plan := DerivePlan(mustConfig(t, lanHostsConfig))
	if err := m.Apply(plan); err != nil {
		t.Fatal(err)
	}

	// Second start: the entries are on the box now.
	h2 := &fakeHost{
		dnsServer:    "127.0.0.1#5333\n",
		rebindDomain: "home.tparts.com keel.tparts.com nas.tparts.com\n",
	}
	m.run = h2.run
	if err := m.Apply(plan); err != nil {
		t.Fatal(err)
	}
	if h2.ran("add_list dhcp.@dnsmasq[0].rebind_domain") {
		t.Errorf("re-added existing exemptions\ngot: %v", h2.calls)
	}
	if h2.ran("/etc/init.d/dnsmasq restart") {
		t.Errorf("restarted dnsmasq with nothing to change\ngot: %v", h2.calls)
	}

	st, _ := m.loadState()
	if len(st.RebindDomains) != 3 {
		t.Errorf("RebindDomains = %v, want the three still recorded", st.RebindDomains)
	}
}

func TestApplyDisabledWritesNoExemption(t *testing.T) {
	h := &fakeHost{}
	m := NewManager(false, filepath.Join(t.TempDir(), "state.json"))
	m.run = h.run
	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatal(err)
	}
	if len(h.calls) != 0 {
		t.Errorf("ran %v on a non-OpenWrt host, want nothing", h.calls)
	}
}

func TestApplyPrunesExemptionsDroppedFromTheConfig(t *testing.T) {
	// The operator deletes a name from the hosts server. Its exemption has no
	// reason to exist any more, and waiting for a Revert that may never come
	// would leave a hole in the rebind protection indefinitely.
	h := &fakeHost{dnsServer: "223.5.5.5\n"}
	m := newTestManager(t, h)
	if err := m.Apply(DerivePlan(mustConfig(t, lanHostsConfig))); err != nil {
		t.Fatal(err)
	}

	shrunk := mustConfig(t, `{
		"dns": { "servers": [ { "type": "hosts", "tag": "dns_lan", "predefined": {
			"home.tparts.com": "192.168.9.207"
		} } ] },
		"inbounds": [
			{ "type": "direct", "tag": "dns-in", "listen": "0.0.0.0", "listen_port": 5333 },
			{ "type": "tun", "tag": "tun-in", "interface_name": "tun0", "address": ["172.16.250.1/30"] }
		],
		"outbounds": [ { "type": "direct", "tag": "direct" } ],
		"route": { "rules": [ { "inbound": "dns-in", "action": "hijack-dns" } ] }
	}`)

	h2 := &fakeHost{
		dnsServer:    "127.0.0.1#5333\n",
		rebindDomain: "home.tparts.com keel.tparts.com nas.tparts.com\n",
	}
	m.run = h2.run
	if err := m.Apply(DerivePlan(shrunk)); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"del_list dhcp.@dnsmasq[0].rebind_domain=keel.tparts.com",
		"del_list dhcp.@dnsmasq[0].rebind_domain=nas.tparts.com",
	} {
		if !h2.ran(want) {
			t.Errorf("missing %q\ngot: %v", want, h2.calls)
		}
	}
	if h2.ran("del_list dhcp.@dnsmasq[0].rebind_domain=home.tparts.com") {
		t.Errorf("pruned a name the config still maps\ngot: %v", h2.calls)
	}

	st, _ := m.loadState()
	if len(st.RebindDomains) != 1 || st.RebindDomains[0] != "home.tparts.com" {
		t.Errorf("RebindDomains = %v, want only home.tparts.com", st.RebindDomains)
	}
}
