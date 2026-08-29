package openwrtnet

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// zoneName is the firewall zone this package owns. Everything Revert removes
// is identified by this name, so a zone the operator created by hand is never
// touched.
const zoneName = "sing-box-easy"

// lanZone is the source zone forwarded into the tunnel. LAN clients reach the
// proxy through it; without the forwarding, fw4's forward chain (policy drop)
// silently discards their traffic.
const lanZone = "lan"

// Runner executes a host command. Swappable so the lifecycle can be tested
// without a router.
type Runner func(name string, args ...string) (string, error)

// Manager applies and reverts the OpenWrt-side network integration.
//
// Every operation is a no-op when disabled, so callers need not branch on the
// platform.
type Manager struct {
	enabled   bool
	statePath string
	run       Runner
}

// NewManager returns a manager. enabled should be true only on OpenWrt;
// statePath is where the pre-existing host settings are recorded so they can be
// restored later.
func NewManager(enabled bool, statePath string) *Manager {
	return &Manager{
		enabled:   enabled,
		statePath: statePath,
		run:       execRun,
	}
}

func execRun(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %s: %w: %s",
			name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Apply puts the host into the state the plan requires, recording whatever it
// replaces. It is idempotent: re-applying an already-applied plan changes
// nothing and, importantly, does not overwrite the recorded original.
func (m *Manager) Apply(plan Plan) error {
	if !m.enabled {
		return nil
	}
	if !plan.NeedsFirewallZone() && !plan.NeedsDNSRedirect() {
		return nil
	}

	st, err := m.loadState()
	if err != nil {
		return err
	}

	if plan.NeedsFirewallZone() {
		if err := m.ensureZone(plan.TunInterface, &st); err != nil {
			return err
		}
	}

	if plan.NeedsDNSRedirect() {
		// The exemption is staged before the redirect so the two land in the
		// same dnsmasq restart. There is never a window where dnsmasq
		// forwards to sing-box while still filtering the private addresses
		// sing-box answers with.
		//
		// It is gated on the redirect for the same reason: rebind protection
		// only strips answers that arrive from an upstream server, and
		// sing-box is not one until dnsmasq points at it.
		exempted, err := m.stageRebindSync(plan.LANDomains, &st)
		if err != nil {
			return err
		}
		redirected, err := m.stageDNSRedirect(plan.DNSUpstream(), &st)
		if err != nil {
			return err
		}
		if exempted || redirected {
			if err := m.commitDNSMasq(); err != nil {
				return err
			}
		}
	}

	return m.saveState(st)
}

// Revert restores whatever Apply replaced and forgets the state.
//
// Errors are collected rather than returned early: a failure to remove the
// firewall zone must not leave dnsmasq pointing at a port that is about to
// close, which would take the LAN's DNS down with the proxy.
func (m *Manager) Revert() error {
	if !m.enabled {
		return nil
	}

	st, err := m.loadState()
	if err != nil {
		return err
	}
	if !st.ZoneCreated && !st.DNSChanged && len(st.RebindDomains) == 0 {
		return nil
	}

	var problems []string
	if st.DNSChanged || len(st.RebindDomains) > 0 {
		if err := m.restoreDNS(st); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if st.ZoneCreated {
		if err := m.removeZone(); err != nil {
			problems = append(problems, err.Error())
		}
	}

	// Clear the state either way: retrying a partial revert on the next stop
	// would re-delete sections that are already gone.
	if err := m.clearState(); err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) > 0 {
		return fmt.Errorf("revert incomplete: %s", strings.Join(problems, "; "))
	}
	return nil
}

// ensureZone creates the firewall zone for the tun device, plus a lan->zone
// forwarding, unless the zone already exists.
func (m *Manager) ensureZone(tunInterface string, st *state) error {
	show, err := m.run("uci", "show", "firewall")
	if err != nil {
		return fmt.Errorf("failed to read the firewall config: %w", err)
	}
	if _, exists := findAnonSection(show, "firewall", "zone", "name", zoneName); exists {
		st.ZoneCreated = true // ours to remove on revert
		return nil
	}

	// masq is deliberately off. sing-box reads packets off the tun fd and
	// opens its own outbound connections, so masquerading would only rewrite
	// the source to the tun address and destroy the client IP that route rules
	// match on. mtu_fix clamps MSS, which matters against a tun MTU well above
	// the real path.
	steps := [][]string{
		{"add", "firewall", "zone"},
		{"set", "firewall.@zone[-1].name=" + zoneName},
		{"set", "firewall.@zone[-1].input=ACCEPT"},
		{"set", "firewall.@zone[-1].output=ACCEPT"},
		{"set", "firewall.@zone[-1].forward=ACCEPT"},
		{"set", "firewall.@zone[-1].masq=0"},
		{"set", "firewall.@zone[-1].mtu_fix=1"},
		{"add_list", "firewall.@zone[-1].device=" + tunInterface},
		{"add", "firewall", "forwarding"},
		{"set", "firewall.@forwarding[-1].src=" + lanZone},
		{"set", "firewall.@forwarding[-1].dest=" + zoneName},
		{"commit", "firewall"},
	}
	for _, args := range steps {
		if _, err := m.run("uci", args...); err != nil {
			return fmt.Errorf("failed to create the firewall zone: %w", err)
		}
	}
	if _, err := m.run("/etc/init.d/firewall", "reload"); err != nil {
		return fmt.Errorf("failed to reload the firewall: %w", err)
	}

	st.ZoneCreated = true
	logger.Info("OpenWrt firewall zone created for sing-box",
		zap.String("zone", zoneName), zap.String("device", tunInterface))
	return nil
}

func (m *Manager) removeZone() error {
	show, err := m.run("uci", "show", "firewall")
	if err != nil {
		return fmt.Errorf("failed to read the firewall config: %w", err)
	}

	// Forwarding first: it references the zone.
	if section, ok := findAnonSection(show, "firewall", "forwarding", "dest", zoneName); ok {
		if _, err := m.run("uci", "delete", section); err != nil {
			return fmt.Errorf("failed to remove the firewall forwarding: %w", err)
		}
		// Deleting shifts the anonymous indices, so re-read before the zone.
		if show, err = m.run("uci", "show", "firewall"); err != nil {
			return fmt.Errorf("failed to re-read the firewall config: %w", err)
		}
	}
	if section, ok := findAnonSection(show, "firewall", "zone", "name", zoneName); ok {
		if _, err := m.run("uci", "delete", section); err != nil {
			return fmt.Errorf("failed to remove the firewall zone: %w", err)
		}
	}

	if _, err := m.run("uci", "commit", "firewall"); err != nil {
		return fmt.Errorf("failed to commit the firewall config: %w", err)
	}
	if _, err := m.run("/etc/init.d/firewall", "reload"); err != nil {
		return fmt.Errorf("failed to reload the firewall: %w", err)
	}

	logger.Info("OpenWrt firewall zone removed", zap.String("zone", zoneName))
	return nil
}

// stageDNSRedirect points dnsmasq at sing-box, recording the previous
// upstreams. Nothing is committed: the caller batches every dhcp edit into a
// single commit and restart.
func (m *Manager) stageDNSRedirect(upstream string, st *state) (bool, error) {
	current, err := m.run("uci", "-q", "get", "dhcp.@dnsmasq[0].server")
	// A missing option exits non-zero; that is "no upstream configured", not a
	// failure, and the empty result is what we want to record.
	if err != nil {
		current = ""
	}
	servers := parseUCIList(current)

	if isAlreadyRedirected(servers, upstream) {
		st.DNSChanged = true
		return false, nil
	}

	// Only capture the original once. Re-applying must not record our own
	// value as the thing to restore.
	if !st.DNSChanged {
		st.PriorServers = servers
	}

	if err := m.stageDNSServers([]string{upstream}); err != nil {
		return false, err
	}
	st.DNSChanged = true
	logger.Info("dnsmasq redirected to sing-box",
		zap.String("upstream", upstream), zap.Strings("previous", st.PriorServers))
	return true, nil
}

// restoreDNS undoes both dhcp edits in a single commit and restart, so the
// router never runs with one half applied — in particular never forwards to
// sing-box with the rebind protection already unexempted, the state that
// silently blanks every LAN name.
func (m *Manager) restoreDNS(st state) error {
	dirty, removeErr := m.stageRebindRemove(st.RebindDomains)

	if st.DNSChanged {
		if err := m.stageDNSServers(st.PriorServers); err != nil {
			return err
		}
		dirty = true
	}
	if dirty {
		if err := m.commitDNSMasq(); err != nil {
			return err
		}
	}
	if removeErr != nil {
		return removeErr
	}
	if st.DNSChanged {
		logger.Info("dnsmasq upstream restored", zap.Strings("servers", st.PriorServers))
	}
	return nil
}

// stageDNSServers replaces the dnsmasq upstream list. An empty list clears the
// option entirely, which is how dnsmasq falls back to resolv.conf.
func (m *Manager) stageDNSServers(servers []string) error {
	// Ignore the delete error: the option is legitimately absent when nothing
	// was configured.
	_, _ = m.run("uci", "-q", "delete", "dhcp.@dnsmasq[0].server")

	for _, s := range servers {
		// add_list, not set: the dnsmasq init script iterates `server` as a
		// UCI list. Writing it as a plain option makes the generated
		// dnsmasq.conf carry no server= line at all, and every query is
		// REFUSED.
		if _, err := m.run("uci", "add_list", "dhcp.@dnsmasq[0].server="+s); err != nil {
			return fmt.Errorf("failed to set the dnsmasq upstream: %w", err)
		}
	}
	return nil
}

// commitDNSMasq flushes the staged dhcp edits and reloads dnsmasq once.
func (m *Manager) commitDNSMasq() error {
	if _, err := m.run("uci", "commit", "dhcp"); err != nil {
		return fmt.Errorf("failed to commit the dhcp config: %w", err)
	}
	if _, err := m.run("/etc/init.d/dnsmasq", "restart"); err != nil {
		return fmt.Errorf("failed to restart dnsmasq: %w", err)
	}
	return nil
}

func isAlreadyRedirected(servers []string, upstream string) bool {
	return len(servers) == 1 && servers[0] == upstream
}
