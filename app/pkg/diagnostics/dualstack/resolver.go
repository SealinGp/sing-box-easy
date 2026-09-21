// Package dualstack answers "does this domain work over IPv4 and over IPv6,
// and does each family go where the config says it should?"
//
// It is the server-side form of scripts/validate-ipv6-split.sh, and it exists
// because that script cannot run where the answer is: its network sections are
// SKIPPED on a client without global IPv6, which is most laptops, so the one
// machine that can actually test the router's IPv6 egress is the router.
//
// The three questions are deliberately kept apart, because a config can fail
// any one of them while passing the others:
//
//	resolve   what does sing-box return for A and for AAAA?
//	predict   where would each of those addresses be routed?
//	observe   where did it actually go, and did it get there?
//
// A suppressed AAAA is a PASS, not a failure — it is what an IPv6 split does —
// so "no address for this family" is its own outcome and never an error.
package dualstack

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"
)

// Source says how the DNS endpoint was determined, which is as much of the
// answer as the address is.
type Source string

const (
	// SourceNone means no route rule hijacks DNS at all, so a LAN client's
	// queries never reach sing-box.
	SourceNone Source = "none"
	// SourceInbound means a hijack rule names an inbound, and that inbound's
	// listen address is where clients should point their resolver.
	SourceInbound Source = "inbound"
	// SourceProtocol means a `protocol: dns` rule hijacks DNS on every
	// inbound. Emphatically not SourceNone: DNS is hijacked everywhere, there
	// is simply no single address to name.
	SourceProtocol Source = "protocol_dns"
)

// DNSEndpoint is where a client's DNS lands, per the running config.
type DNSEndpoint struct {
	Source Source `json:"source"`
	// Inbound is the tag the hijack rule named, when it named one.
	Inbound string `json:"inbound,omitempty"`
	// Listen and Port are the inbound's configured bind, verbatim.
	Listen string `json:"listen,omitempty"`
	Port   uint16 `json:"port,omitempty"`
	// Address is host:port with a wildcard bind rewritten to loopback, ready
	// to query. Empty when the config does not pin one down.
	Address string `json:"address,omitempty"`
}

// hijackAction is the route action that sends a connection into the DNS
// router instead of an outbound.
const hijackAction = "hijack-dns"

// DiscoverDNSEndpoint walks the raw route and inbound sections for the DNS
// hijack, and resolves it to an address.
//
// Raw JSON rather than the decoded option structs on purpose: this runs
// against whatever sing-box the host installed, and the panel's pinned schema
// rejects configs a newer core accepts. Both keys read here — a rule's
// `action`/`inbound` and an inbound's `listen`/`listen_port` — have been
// stable across every version this panel supports.
//
// Every malformed input degrades to SourceNone. This is third-party config
// text reached from an HTTP handler; it must not be able to panic one.
func DiscoverDNSEndpoint(route, inbounds json.RawMessage) DNSEndpoint {
	rules := rawRules(route)
	listens := inboundListens(inbounds)

	// Two passes rather than one, because the useful rule is not the first
	// one. A production config hijacks a dedicated inbound AND, below it,
	// everything with `protocol: dns`; only the former yields an address, and
	// taking the first match in config order returns the useless one.
	protocolHijack := false
	var namedFallback DNSEndpoint

	for _, rule := range rules {
		if rawString(rule, "action") != hijackAction {
			continue
		}

		tags := rawStringList(rule, "inbound")
		if len(tags) == 0 {
			protocolHijack = true
			continue
		}

		for _, tag := range tags {
			listen, ok := listens[tag]
			if !ok {
				// A tag no inbound declares has no address behind it.
				continue
			}
			endpoint := DNSEndpoint{
				Source:  SourceInbound,
				Inbound: tag,
				Listen:  listen.host,
				Port:    listen.port,
			}
			if listen.port == 0 {
				// Named but unusable: keep it only as a last resort, so a
				// protocol-wide hijack elsewhere is preferred over an
				// address that cannot be queried.
				if namedFallback.Source == "" {
					namedFallback = endpoint
				}
				continue
			}
			endpoint.Address = net.JoinHostPort(reachableHost(listen.host), strconv.Itoa(int(listen.port)))
			return endpoint
		}
	}

	if protocolHijack {
		return DNSEndpoint{Source: SourceProtocol}
	}
	if namedFallback.Source != "" {
		return namedFallback
	}
	return DNSEndpoint{Source: SourceNone}
}

// reachableHost turns a bind address into one that can be dialled.
//
// `listen` is where sing-box binds, not where it can be reached: an empty
// value or a wildcard means "every interface", which is not an address. The
// panel shares a host with sing-box, so loopback is the correct rewrite — the
// same one clashapi.ControllerURL makes for external_controller.
func reachableHost(listen string) string {
	switch strings.TrimSpace(listen) {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	default:
		return strings.TrimSpace(listen)
	}
}

// listenAddress is one inbound's bind, as configured.
type listenAddress struct {
	host string
	port uint16
}

// inboundListens indexes the inbound array by tag.
func inboundListens(inbounds json.RawMessage) map[string]listenAddress {
	result := map[string]listenAddress{}

	var list []json.RawMessage
	if json.Unmarshal(inbounds, &list) != nil {
		return result
	}
	for _, entry := range list {
		tag := rawString(entry, "tag")
		if tag == "" {
			continue
		}
		result[tag] = listenAddress{
			host: rawString(entry, "listen"),
			port: rawPort(entry, "listen_port"),
		}
	}
	return result
}

// rawRules pulls `route.rules` out, tolerating every shape it is not.
func rawRules(route json.RawMessage) []json.RawMessage {
	var object map[string]json.RawMessage
	if json.Unmarshal(route, &object) != nil {
		return nil
	}
	var rules []json.RawMessage
	if json.Unmarshal(object["rules"], &rules) != nil {
		return nil
	}
	return rules
}

// rawString reads a string field, or "" for anything else.
func rawString(object json.RawMessage, key string) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(object, &fields) != nil {
		return ""
	}
	var value string
	if json.Unmarshal(fields[key], &value) != nil {
		return ""
	}
	return value
}

// rawStringList reads a field sing-box declares as Listable: either a bare
// string or an array of them. Both spellings appear in real configs.
func rawStringList(object json.RawMessage, key string) []string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(object, &fields) != nil {
		return nil
	}
	value, ok := fields[key]
	if !ok {
		return nil
	}
	var list []string
	if json.Unmarshal(value, &list) == nil {
		return list
	}
	var single string
	if json.Unmarshal(value, &single) == nil && single != "" {
		return []string{single}
	}
	return nil
}

// rawPort reads a port field, rejecting anything outside 1-65535.
func rawPort(object json.RawMessage, key string) uint16 {
	var fields map[string]json.RawMessage
	if json.Unmarshal(object, &fields) != nil {
		return 0
	}
	var value int
	if json.Unmarshal(fields[key], &value) != nil {
		return 0
	}
	if value < 1 || value > 65535 {
		return 0
	}
	return uint16(value)
}
