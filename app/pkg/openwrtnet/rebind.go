package openwrtnet

import (
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// dnsmasq's DNS rebind protection (`stop-dns-rebind`) drops every RFC1918
// address that arrives from an upstream server. Once the DNS redirect lands,
// *all* of sing-box's answers are upstream answers — including the LAN
// mappings a `hosts` DNS server exists to serve. The query then comes back
// NOERROR with zero records and the authoritative flag set, which looks like a
// broken sing-box rule and is not: sing-box resolved the name correctly and
// dnsmasq threw the answer away on the way out.
//
// dnsmasq's `rebind_domain` (--rebind-domain-ok) exempts individual names, so
// the protection keeps applying to everything else. Only names sing-box
// actually maps to a private address are exempted, and only entries this
// package added are removed again on Revert — an exemption the operator wrote
// by hand outlives us.

// rebindProtectionEnabled reports whether dnsmasq is filtering private
// addresses out of upstream replies.
//
// An absent option means enabled: OpenWrt's dnsmasq init reads it with
// `config_get_bool ... 1`, so the protection is on unless explicitly turned
// off. Reading absence as "off" would skip the exemption on a stock router,
// which is precisely the box that needs it.
func (m *Manager) rebindProtectionEnabled() bool {
	out, err := m.run("uci", "-q", "get", "dhcp.@dnsmasq[0].rebind_protection")
	if err != nil {
		return true
	}
	switch strings.TrimSpace(out) {
	case "0", "off", "false", "no", "disabled":
		return false
	}
	return true
}

// currentRebindDomains reads the exemptions already on the box.
func (m *Manager) currentRebindDomains() map[string]bool {
	// A missing option exits non-zero; that is "nothing exempted", not a
	// failure worth aborting the start for.
	out, err := m.run("uci", "-q", "get", "dhcp.@dnsmasq[0].rebind_domain")
	if err != nil {
		return map[string]bool{}
	}
	present := make(map[string]bool)
	for _, d := range parseUCIList(out) {
		present[normalizeDomain(d)] = true
	}
	return present
}

// stageRebindSync brings the exemptions on the box in line with the plan,
// without committing. It reports whether anything was staged, so the caller
// can restart dnsmasq once for every dhcp edit rather than once per edit.
//
// It both adds and prunes: a name the operator deletes from the hosts server
// should stop being exempt at the next restart, not linger as a hole in the
// rebind protection until someone happens to stop sing-box.
func (m *Manager) stageRebindSync(domains []string, st *state) (bool, error) {
	if len(domains) == 0 && len(st.RebindDomains) == 0 {
		return false, nil
	}
	if len(domains) > 0 && !m.rebindProtectionEnabled() {
		return false, nil
	}

	want := make(map[string]bool, len(domains))
	for _, d := range domains {
		want[d] = true
	}
	present := m.currentRebindDomains()

	var added []string
	for _, domain := range domains {
		// Whatever is already on the box stays there untouched — whether the
		// operator wrote it or a previous Apply did. Only names this call
		// actually writes are recorded, which is what stops Revert from
		// deleting an exemption it did not create.
		if present[domain] {
			continue
		}
		if _, err := m.run("uci", "add_list", "dhcp.@dnsmasq[0].rebind_domain="+domain); err != nil {
			return false, fmt.Errorf("failed to exempt %s from the DNS rebind protection: %w", domain, err)
		}
		added = append(added, domain)
	}

	// Prune only from what we recorded. A name the plan never mentioned and we
	// never wrote is somebody else's business.
	var stale []string
	var keep []string
	for _, domain := range st.RebindDomains {
		if want[domain] {
			keep = append(keep, domain)
			continue
		}
		stale = append(stale, domain)
	}
	removed, err := m.stageRebindRemove(stale)
	if err != nil {
		return removed || len(added) > 0, err
	}

	if len(added) > 0 {
		logger.Info("LAN names exempted from dnsmasq's DNS rebind protection",
			zap.Strings("domains", added))
	}
	st.RebindDomains = mergeDomains(keep, added)
	return len(added) > 0 || removed, nil
}

// stageRebindRemove drops the exemptions this package added, without
// committing.
func (m *Manager) stageRebindRemove(domains []string) (bool, error) {
	if len(domains) == 0 {
		return false, nil
	}

	present := m.currentRebindDomains()
	removed := false
	for _, domain := range domains {
		if !present[domain] {
			continue // already gone; deleting again is an error on some builds
		}
		if _, err := m.run("uci", "del_list", "dhcp.@dnsmasq[0].rebind_domain="+domain); err != nil {
			return removed, fmt.Errorf("failed to remove the rebind exemption for %s: %w", domain, err)
		}
		removed = true
	}
	if removed {
		logger.Info("dnsmasq rebind exemptions removed", zap.Strings("domains", domains))
	}
	return removed, nil
}

// mergeDomains appends without duplicating, preserving the existing order.
func mergeDomains(existing, added []string) []string {
	seen := make(map[string]bool, len(existing))
	for _, d := range existing {
		seen[d] = true
	}
	out := append([]string(nil), existing...)
	for _, d := range added {
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}
