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
// Both are reverted on stop, so stopping sing-box returns the router to plain
// routing rather than leaving it with a zone for a dead interface and a
// dnsmasq pointing at a closed port — which is exactly how a stopped proxy
// takes the whole LAN's DNS down with it.
package openwrtnet

import (
	"context"
	"fmt"
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

// Plan is what the running sing-box config needs from the host.
type Plan struct {
	// TunInterface is the tun device that needs a firewall zone, or "".
	TunInterface string
	// DNSAddress and DNSPort locate the inbound that a hijack-dns rule feeds.
	DNSAddress string
	DNSPort    uint16

	// hijack records that a hijack-dns rule actually points at that inbound.
	// A DNS-shaped inbound without one is a plain forwarder, and redirecting
	// dnsmasq into it would send queries somewhere they are not resolved.
	hijack bool
}

// NeedsFirewallZone reports whether a tun device requires a zone.
func (p Plan) NeedsFirewallZone() bool { return p.TunInterface != "" }

// NeedsDNSRedirect reports whether dnsmasq should be pointed at sing-box.
func (p Plan) NeedsDNSRedirect() bool { return p.hijack && p.DNSPort != 0 }

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
		Route    struct {
			Rules []map[string]any `json:"rules"`
		} `json:"route"`
	}
	if err := json.UnmarshalContext(ctx, raw, &doc); err != nil {
		return plan
	}

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
