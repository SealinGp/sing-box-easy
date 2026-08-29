package openwrtnet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHost records every command and answers the queries Apply/Revert make.
type fakeHost struct {
	calls            []string
	uciShow          string // response to `uci show firewall`
	dnsServer        string // response to `uci -q get dhcp.@dnsmasq[0].server`
	rebindDomain     string // response to `uci -q get dhcp.@dnsmasq[0].rebind_domain`
	rebindProtection string // response to `uci -q get ...rebind_protection`; "" is absent
	failOn           string // substring of a command that should fail
}

func (f *fakeHost) run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)
	if f.failOn != "" && strings.Contains(cmd, f.failOn) {
		return "", os.ErrPermission
	}
	switch {
	case strings.Contains(cmd, "show firewall"):
		return f.uciShow, nil
	case strings.Contains(cmd, "get dhcp.@dnsmasq[0].server"):
		return f.dnsServer, nil
	case strings.Contains(cmd, "get dhcp.@dnsmasq[0].rebind_domain"):
		return f.rebindDomain, nil
	case strings.Contains(cmd, "get dhcp.@dnsmasq[0].rebind_protection"):
		// An absent option exits non-zero on a real box, and OpenWrt's init
		// script then defaults the protection on.
		if f.rebindProtection == "" {
			return "", os.ErrNotExist
		}
		return f.rebindProtection, nil
	}
	return "", nil
}

func (f *fakeHost) ran(substr string) bool {
	for _, c := range f.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func newTestManager(t *testing.T, h *fakeHost) *Manager {
	t.Helper()
	m := NewManager(true, filepath.Join(t.TempDir(), "state.json"))
	m.run = h.run
	return m
}

var fullPlan = Plan{TunInterface: "tun0", DNSAddress: "127.0.0.1", DNSPort: 5333, hijack: true}

func TestApplyDisabledIsNoOp(t *testing.T) {
	h := &fakeHost{}
	m := NewManager(false, filepath.Join(t.TempDir(), "state.json"))
	m.run = h.run

	if err := m.Apply(fullPlan); err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if len(h.calls) != 0 {
		t.Errorf("ran %v on a non-OpenWrt host, want nothing", h.calls)
	}
}

func TestApplyCreatesZoneAndRedirectsDNS(t *testing.T) {
	h := &fakeHost{uciShow: "firewall.@zone[0].name='lan'\n", dnsServer: "127.0.0.1#7874\n"}
	m := newTestManager(t, h)

	if err := m.Apply(fullPlan); err != nil {
		t.Fatalf("Apply() = %v", err)
	}

	for _, want := range []string{
		"add firewall zone",
		"firewall.@zone[-1].name=" + zoneName,
		"firewall.@zone[-1].device=tun0",
		"add firewall forwarding",
		"commit firewall",
		"add_list dhcp.@dnsmasq[0].server=127.0.0.1#5333",
		"commit dhcp",
	} {
		if !h.ran(want) {
			t.Errorf("missing command containing %q\ngot: %v", want, h.calls)
		}
	}

	// The prior upstream has to be recorded or Revert cannot put it back.
	st, err := m.loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if !st.DNSChanged || len(st.PriorServers) != 1 || st.PriorServers[0] != "127.0.0.1#7874" {
		t.Errorf("state = %+v, want the previous upstream recorded", st)
	}
	if !st.ZoneCreated {
		t.Error("state.ZoneCreated = false")
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	// Zone already present: a second Apply must not add a duplicate.
	h := &fakeHost{
		uciShow:   "firewall.@zone[2].name='" + zoneName + "'\n",
		dnsServer: "127.0.0.1#5333\n",
	}
	m := newTestManager(t, h)

	if err := m.Apply(fullPlan); err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if h.ran("add firewall zone") {
		t.Errorf("created a second zone\ngot: %v", h.calls)
	}
}

func TestApplyDoesNotRecordItselfAsThePriorUpstream(t *testing.T) {
	// Re-applying while already pointed at sing-box must not overwrite the
	// saved original with our own value — that would make Revert restore the
	// dead port instead of the operator's real resolver.
	h := &fakeHost{uciShow: "", dnsServer: "127.0.0.1#7874\n"}
	m := newTestManager(t, h)
	if err := m.Apply(fullPlan); err != nil {
		t.Fatal(err)
	}

	h2 := &fakeHost{uciShow: "firewall.@zone[0].name='" + zoneName + "'\n", dnsServer: "127.0.0.1#5333\n"}
	m.run = h2.run
	if err := m.Apply(fullPlan); err != nil {
		t.Fatal(err)
	}

	st, _ := m.loadState()
	if len(st.PriorServers) != 1 || st.PriorServers[0] != "127.0.0.1#7874" {
		t.Errorf("PriorServers = %v, want the original 127.0.0.1#7874 preserved", st.PriorServers)
	}
}

func TestRevertRestoresPreviousState(t *testing.T) {
	h := &fakeHost{uciShow: "firewall.@zone[0].name='lan'\n", dnsServer: "223.5.5.5 119.29.29.29\n"}
	m := newTestManager(t, h)
	if err := m.Apply(fullPlan); err != nil {
		t.Fatal(err)
	}

	h2 := &fakeHost{
		uciShow: "firewall.@zone[0].name='lan'\n" +
			"firewall.@zone[1].name='" + zoneName + "'\n" +
			"firewall.@forwarding[0].dest='" + zoneName + "'\n",
	}
	m.run = h2.run
	if err := m.Revert(); err != nil {
		t.Fatalf("Revert() = %v", err)
	}

	for _, want := range []string{
		"delete firewall.@forwarding[0]",
		"delete firewall.@zone[1]",
		"add_list dhcp.@dnsmasq[0].server=223.5.5.5",
		"add_list dhcp.@dnsmasq[0].server=119.29.29.29",
		"commit dhcp",
	} {
		if !h2.ran(want) {
			t.Errorf("missing %q\ngot: %v", want, h2.calls)
		}
	}

	// State is consumed, so a second Revert does nothing.
	h3 := &fakeHost{}
	m.run = h3.run
	if err := m.Revert(); err != nil {
		t.Fatalf("second Revert() = %v", err)
	}
	if len(h3.calls) != 0 {
		t.Errorf("second Revert ran %v, want nothing", h3.calls)
	}
}

func TestRevertWithNoPriorUpstreamClearsTheOption(t *testing.T) {
	// dnsmasq had no explicit upstream before us (it used resolv.conf), so
	// Revert must delete the option rather than invent one.
	h := &fakeHost{dnsServer: "\n"}
	m := newTestManager(t, h)
	if err := m.Apply(fullPlan); err != nil {
		t.Fatal(err)
	}

	h2 := &fakeHost{}
	m.run = h2.run
	if err := m.Revert(); err != nil {
		t.Fatal(err)
	}
	if h2.ran("add_list dhcp.@dnsmasq[0].server=") {
		t.Errorf("restored an upstream that never existed\ngot: %v", h2.calls)
	}
	if !h2.ran("delete dhcp.@dnsmasq[0].server") {
		t.Errorf("did not clear the option\ngot: %v", h2.calls)
	}
}

func TestApplyOnlyWhatThePlanNeeds(t *testing.T) {
	h := &fakeHost{}
	m := newTestManager(t, h)

	// Proxy-only config: no tun, no hijacked DNS.
	if err := m.Apply(Plan{}); err != nil {
		t.Fatalf("Apply() = %v", err)
	}
	if len(h.calls) != 0 {
		t.Errorf("ran %v for an empty plan, want nothing", h.calls)
	}
}
