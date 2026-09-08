// Package openwrtnet applies the host-side network integration that a sing-box
// TUN gateway needs on OpenWrt, and takes it back down again when sing-box
// stops.
//
// sing-box's auto_route sets up policy routing, but it knows nothing about
// OpenWrt's zone-based firewall or about dnsmasq. Two host-side pieces are
// therefore required, and neither is sing-box's to create:
//
//  1. The tun interface must belong to a firewall zone. Without one, fw4's
//     input chain falls through to handle_reject and answers every packet
//     returning from the tunnel with a TCP reset — connections fail instantly
//     even though routing is perfect.
//
//  2. dnsmasq must forward to sing-box's DNS inbound. Otherwise LAN clients
//     keep resolving through whatever dnsmasq pointed at before, and none of
//     sing-box's DNS rules apply.
//
//  3. dnsmasq's DNS rebind protection must be told about the names sing-box
//     answers with a private address. Once (2) is in place those answers
//     arrive from an "upstream" server, and stop-dns-rebind strips them —
//     the client gets NOERROR with zero records for a name sing-box resolved
//     perfectly well.
//
// All three are reverted on stop, so stopping sing-box returns the router to
// plain routing rather than leaving it with a zone for a dead interface and a
// dnsmasq pointing at a closed port — which is exactly how a stopped proxy
// takes the whole LAN's DNS down with it.
package openwrtnet

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/sagernet/sing/common/json"
)

// defaultTunInterface mirrors sing-box's own default when a tun inbound omits
// interface_name.
const defaultTunInterface = "tun0"

// hijackDNSAction is the route-rule action that sends a DNS query into
// sing-box's internal resolver.
const hijackDNSAction = "hijack-dns"

// hostsDNSType is the sing-box DNS server that answers from a predefined
// name -> address map. It is the one server type whose answers are known
// ahead of time, which is what makes the rebind exemption derivable at all:
// every other server is resolved at query time and could return anything.
const hostsDNSType = "hosts"

// Plan is what the running sing-box config needs from the host.
type Plan struct {
	// TunInterface is the tun device that needs a firewall zone, or "".
	TunInterface string
	// DNSAddress and DNSPort locate the inbound that a hijack-dns rule feeds.
	DNSAddress string
	DNSPort    uint16

	// LANDomains are the names a hosts DNS server maps to a private address.
	// Sorted and deduplicated so the exemption written to dnsmasq is stable
	// across restarts and does not churn the uci list.
	LANDomains []string

	// hijack records that a hijack-dns rule actually points at that inbound.
	// A DNS-shaped inbound without one is a plain forwarder, and redirecting
	// dnsmasq into it would send queries somewhere they are not resolved.
	hijack bool
}

// NeedsFirewallZone reports whether a tun device requires a zone.
func (p Plan) NeedsFirewallZone() bool { return p.TunInterface != "" }

// NeedsDNSRedirect reports whether dnsmasq should be pointed at sing-box.
func (p Plan) NeedsDNSRedirect() bool { return p.hijack && p.DNSPort != 0 }

// NeedsRebindAllow reports whether any name has to be exempted from dnsmasq's
// rebind protection.
func (p Plan) NeedsRebindAllow() bool { return len(p.LANDomains) > 0 }

// DNSUpstream renders the plan as a dnsmasq `server=` value.
//
// A wildcard listen address is rewritten to loopback: dnsmasq needs somewhere
// to dial, and "0.0.0.0#5333" is not a destination.
func (p Plan) DNSUpstream() string {
	addr := strings.TrimSpace(p.DNSAddress)
	switch addr {
	case "", "0.0.0.0", "::", "[::]":
		addr = "127.0.0.1"
	}
	return fmt.Sprintf("%s#%d", addr, p.DNSPort)
}

// DerivePlan inspects a sing-box config and reports what the host needs.
//
// The config is walked as generic JSON rather than through sing-box's typed
// option structs. Inbound.Options is an `any` populated by a type registry, so
// typed access would mean enumerating every inbound type and would silently
// yield nothing for configs decoded by a plain json.Unmarshal. This mirrors
// config.outboundOptionsAsMap, which exists for the same reason.
func DerivePlan(cfg *config.SingBoxConfig) Plan {
	var plan Plan
	if cfg == nil {
		return plan
	}

	ctx := config.CreateContext(context.Background())
	raw, err := json.MarshalContext(ctx, cfg)
	if err != nil {
		return plan
	}
	var doc struct {
		Inbounds []map[string]any `json:"inbounds"`
		DNS      struct {
			Servers []map[string]any `json:"servers"`
		} `json:"dns"`
		Route struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.UnmarshalContext(ctx, raw, &doc); err != nil {
		return plan
	}

	plan.LANDomains = privateHostsDomains(doc.DNS.Servers)
	hijacked := hijackedInboundTags(doc.Route.Rules)

	for _, in := range doc.Inbounds {
		switch stringField(in, "type") {
		case "tun":
			name := stringField(in, "interface_name")
			if name == "" {
				name = defaultTunInterface
			}
			plan.TunInterface = name
		default:
			tag := stringField(in, "tag")
			if tag == "" || !hijacked[tag] {
				continue
			}
			port := uintField(in, "listen_port")
			if port == 0 {
				continue
			}
			plan.DNSAddress = stringField(in, "listen")
			plan.DNSPort = port
			plan.hijack = true
		}
	}

	return plan
}

// hijackedInboundTags collects the inbound tags named by hijack-dns rules.
// `inbound` may be a single string or a list, matching sing-box's Listable.
func hijackedInboundTags(rules []map[string]any) map[string]bool {
	tags := make(map[string]bool)
	for _, rule := range rules {
		if stringField(rule, "action") != hijackDNSAction {
			continue
		}
		switch v := rule["inbound"].(type) {
		case string:
			tags[v] = true
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					tags[s] = true
				}
			}
		}
	}
	return tags
}

// privateHostsDomains collects the names that a hosts DNS server answers with
// an address dnsmasq's rebind protection would strip.
//
// Only names with a private answer are returned. A hosts entry pinning a
// public IP passes rebind protection untouched, and exempting it would widen
// the hole in the protection for no benefit.
func privateHostsDomains(servers []map[string]any) []string {
	seen := make(map[string]bool)
	var domains []string

	for _, srv := range servers {
		if stringField(srv, "type") != hostsDNSType {
			continue
		}
		predefined, ok := srv["predefined"].(map[string]any)
		if !ok {
			continue
		}
		for name, addrs := range predefined {
			domain := normalizeDomain(name)
			if domain == "" || seen[domain] || !hasPrivateAddress(addrs) {
				continue
			}
			seen[domain] = true
			domains = append(domains, domain)
		}
	}

	// Map iteration is random; sort so the derived plan — and therefore the
	// uci list and the recorded state — is the same on every start.
	sort.Strings(domains)
	return domains
}

// normalizeDomain reduces a hosts key to the form dnsmasq matches on.
func normalizeDomain(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// hasPrivateAddress reports whether any of a predefined entry's values is
// private. The value is a sing-box Listable, so a single address marshals as a
// bare string and several as an array; both shapes reach here.
func hasPrivateAddress(v any) bool {
	switch addrs := v.(type) {
	case string:
		return isPrivateAddress(addrs)
	case []any:
		for _, item := range addrs {
			if s, ok := item.(string); ok && isPrivateAddress(s) {
				return true
			}
		}
	}
	return false
}

// isPrivateAddress reports whether dnsmasq would treat the address as a
// rebind attempt.
//
// That is RFC1918 and ULA space, plus loopback and link-local. OpenWrt sets
// rebind_localhost_ok by default so 127.x usually survives, but "usually" is
// not a property to build on, and a name resolving to loopback through
// sing-box is a LAN name either way.
func isPrivateAddress(s string) bool {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()
}

func stringField(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return strings.TrimSpace(s)
}

// uintField reads a numeric field. JSON numbers decode as float64 here, but
// tolerate the other shapes a decoder might hand back.
func uintField(m map[string]any, key string) uint16 {
	switch v := m[key].(type) {
	case float64:
		return uint16(v)
	case int:
		return uint16(v)
	case uint16:
		return v
	case int64:
		return uint16(v)
	}
	return 0
}
