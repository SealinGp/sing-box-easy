package dualstack

import (
	"encoding/json"
	"testing"
)

func raw(t *testing.T, value string) json.RawMessage {
	t.Helper()
	if !json.Valid([]byte(value)) {
		t.Fatalf("test fixture is not valid JSON: %s", value)
	}
	return json.RawMessage(value)
}

// productionShape mirrors bin/1.json: a sniff rule, an inbound-scoped hijack,
// and a protocol-scoped one after it.
const productionRoute = `{"rules":[
  {"action":"sniff","timeout":"5s"},
  {"inbound":"dns-in","action":"hijack-dns"},
  {"protocol":"dns","action":"hijack-dns"}
]}`

const productionInbounds = `[
  {"type":"direct","tag":"dns-in","listen":"0.0.0.0","listen_port":5333},
  {"type":"tun","tag":"tun-in","interface_name":"tun0"}
]`

func TestDiscoverPrefersTheInboundScopedHijack(t *testing.T) {
	// Both rules hijack, but only the inbound-scoped one yields an address a
	// client could actually be pointed at. Taking the first rule in order
	// would have been the obvious reading and gives the useless answer here.
	endpoint := DiscoverDNSEndpoint(raw(t, productionRoute), raw(t, productionInbounds))

	if endpoint.Source != SourceInbound {
		t.Fatalf("source = %q, want %q", endpoint.Source, SourceInbound)
	}
	if endpoint.Inbound != "dns-in" {
		t.Fatalf("inbound = %q", endpoint.Inbound)
	}
	if endpoint.Port != 5333 {
		t.Fatalf("port = %d, want 5333", endpoint.Port)
	}
	// 0.0.0.0 is a BIND address, not a reachable one. The panel and sing-box
	// share a host, so it resolves to loopback — the same rewrite the Clash
	// API client makes for external_controller.
	if endpoint.Address != "127.0.0.1:5333" {
		t.Fatalf("address = %q, want 127.0.0.1:5333", endpoint.Address)
	}
}

func TestDiscoverHandlesAListOfInbounds(t *testing.T) {
	route := `{"rules":[{"inbound":["other","dns-in"],"action":"hijack-dns"}]}`
	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, productionInbounds))

	if endpoint.Inbound != "dns-in" {
		t.Fatalf("inbound = %q, want the tag that exists", endpoint.Inbound)
	}
	if endpoint.Address != "127.0.0.1:5333" {
		t.Fatalf("address = %q", endpoint.Address)
	}
}

func TestDiscoverRewritesIPv6Wildcard(t *testing.T) {
	inbounds := `[{"type":"direct","tag":"dns-in","listen":"::","listen_port":53}]`
	route := `{"rules":[{"inbound":"dns-in","action":"hijack-dns"}]}`

	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, inbounds))
	if endpoint.Address != "127.0.0.1:53" {
		t.Fatalf("address = %q", endpoint.Address)
	}
}

func TestDiscoverKeepsAConcreteListenAddress(t *testing.T) {
	inbounds := `[{"type":"direct","tag":"dns-in","listen":"192.168.9.253","listen_port":5333}]`
	route := `{"rules":[{"inbound":"dns-in","action":"hijack-dns"}]}`

	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, inbounds))
	if endpoint.Address != "192.168.9.253:5333" {
		t.Fatalf("address = %q, want the configured address left alone", endpoint.Address)
	}
}

func TestDiscoverFallsBackToProtocolDNS(t *testing.T) {
	// A protocol-scoped hijack applies to every inbound and names no port, so
	// there is no address to report — but it is emphatically not "no hijack",
	// and conflating the two would tell the operator their DNS is unhijacked
	// when in fact it is hijacked everywhere.
	route := `{"rules":[{"protocol":"dns","action":"hijack-dns"}]}`
	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, productionInbounds))

	if endpoint.Source != SourceProtocol {
		t.Fatalf("source = %q, want %q", endpoint.Source, SourceProtocol)
	}
	if endpoint.Address != "" {
		t.Fatalf("address = %q, want none", endpoint.Address)
	}
}

func TestDiscoverReportsNoHijackAtAll(t *testing.T) {
	route := `{"rules":[{"action":"sniff"}]}`
	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, productionInbounds))

	if endpoint.Source != SourceNone {
		t.Fatalf("source = %q, want %q", endpoint.Source, SourceNone)
	}
}

func TestDiscoverIgnoresAHijackNamingAnAbsentInbound(t *testing.T) {
	// The tag does not exist, so there is no address behind it. Reporting the
	// tag with an empty address would look like a discovered endpoint.
	route := `{"rules":[
	  {"inbound":"ghost","action":"hijack-dns"},
	  {"protocol":"dns","action":"hijack-dns"}
	]}`
	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, productionInbounds))

	if endpoint.Source != SourceProtocol {
		t.Fatalf("source = %q, want the usable rule to win", endpoint.Source)
	}
}

func TestDiscoverMissingInboundPortIsNotAnAddress(t *testing.T) {
	inbounds := `[{"type":"direct","tag":"dns-in","listen":"0.0.0.0"}]`
	route := `{"rules":[{"inbound":"dns-in","action":"hijack-dns"}]}`

	endpoint := DiscoverDNSEndpoint(raw(t, route), raw(t, inbounds))
	if endpoint.Address != "" {
		t.Fatalf("address = %q, want none without a port", endpoint.Address)
	}
	if endpoint.Inbound != "dns-in" {
		t.Fatalf("the inbound should still be named, got %q", endpoint.Inbound)
	}
}

func TestDiscoverSurvivesGarbage(t *testing.T) {
	// Every field here is third-party config text. A malformed section must
	// degrade to "unknown", never panic a diagnostic endpoint.
	for _, bad := range []string{`{}`, `{"rules":"nope"}`, `{"rules":[null,3]}`, `null`} {
		endpoint := DiscoverDNSEndpoint(json.RawMessage(bad), json.RawMessage(`"nope"`))
		if endpoint.Source != SourceNone {
			t.Fatalf("%s: source = %q, want none", bad, endpoint.Source)
		}
	}
}
