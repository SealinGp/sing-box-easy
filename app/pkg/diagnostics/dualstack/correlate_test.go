package dualstack

import (
	"net/netip"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
)

func conn(id, ip, port string, start time.Time, rule string, chains []string) clashapi.Connection {
	return clashapi.Connection{
		ID:     id,
		Start:  start,
		Rule:   rule,
		Chains: chains,
		Metadata: clashapi.ConnectionMetadata{
			Network: "tcp", Type: "tun/tun-in",
			DestinationIP: ip, DestinationPort: port, Host: "www.google.com",
		},
	}
}

func TestCorrelateFindsTheDial(t *testing.T) {
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "2001:db8::1", "443", dialedAt.Add(50*time.Millisecond),
			"ip_version=6 => ➡️ 直连", []string{"➡️ 直连"}),
	}}

	observed := Correlate(snapshot, netip.MustParseAddr("2001:db8::1"), 443, dialedAt)
	if observed == nil {
		t.Fatal("expected the connection to be found")
	}
	if observed.Rule != "ip_version=6 => ➡️ 直连" {
		t.Fatalf("rule = %q", observed.Rule)
	}
	// chains is reversed by sing-box: the LAST element is the outbound the
	// rule named, index 0 the leaf actually dialled.
	if observed.Outbound != "➡️ 直连" {
		t.Fatalf("outbound = %q", observed.Outbound)
	}
}

func TestCorrelateReadsChainsFromTheEnd(t *testing.T) {
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "1.2.3.4", "443", dialedAt.Add(time.Millisecond),
			"rule_set=geosite-google => 🤖 AI", []string{"🇭🇰HK-01", "自动选择", "🤖 AI"}),
	}}

	observed := Correlate(snapshot, netip.MustParseAddr("1.2.3.4"), 443, dialedAt)
	if observed.Outbound != "🤖 AI" {
		t.Fatalf("outbound = %q, want the rule's named group", observed.Outbound)
	}
	if observed.Via != "🇭🇰HK-01" {
		t.Fatalf("via = %q, want the leaf actually dialled", observed.Via)
	}
}

func TestCorrelateIgnoresADifferentPortOnTheSameAddress(t *testing.T) {
	// A browser tab already talking to the same CDN address on 80 is not this
	// probe's connection. Matching on the address alone — which the shell
	// script does — picks it up and reports its rule as the probe's.
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "1.2.3.4", "80", dialedAt.Add(time.Millisecond), "other", []string{"direct"}),
	}}

	if observed := Correlate(snapshot, netip.MustParseAddr("1.2.3.4"), 443, dialedAt); observed != nil {
		t.Fatalf("matched an unrelated port: %+v", observed)
	}
}

func TestCorrelateIgnoresAConnectionOlderThanTheDial(t *testing.T) {
	// Same address, same port, but it was already open before the probe ran —
	// so it is somebody else's, and its rule is not evidence about this dial.
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "1.2.3.4", "443", dialedAt.Add(-time.Minute), "stale", []string{"direct"}),
	}}

	if observed := Correlate(snapshot, netip.MustParseAddr("1.2.3.4"), 443, dialedAt); observed != nil {
		t.Fatalf("matched a pre-existing connection: %+v", observed)
	}
}

func TestCorrelateToleratesClockSkewAtTheBoundary(t *testing.T) {
	// sing-box stamps Start from its own clock. A connection reported a few
	// milliseconds BEFORE the dial began is this dial, not a stale one, and
	// an exact comparison would discard the very connection being sought.
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "1.2.3.4", "443", dialedAt.Add(-200*time.Millisecond), "fresh", []string{"direct"}),
	}}

	if observed := Correlate(snapshot, netip.MustParseAddr("1.2.3.4"), 443, dialedAt); observed == nil {
		t.Fatal("a connection within the skew window should still match")
	}
}

func TestCorrelatePrefersTheNewestMatch(t *testing.T) {
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("old", "1.2.3.4", "443", dialedAt.Add(10*time.Millisecond), "first", []string{"direct"}),
		conn("new", "1.2.3.4", "443", dialedAt.Add(90*time.Millisecond), "second", []string{"proxy"}),
	}}

	observed := Correlate(snapshot, netip.MustParseAddr("1.2.3.4"), 443, dialedAt)
	if observed.Rule != "second" {
		t.Fatalf("rule = %q, want the most recent match", observed.Rule)
	}
}

func TestCorrelateNormalizesAddressSpelling(t *testing.T) {
	// sing-box prints the address it parsed; a literal typed by the operator
	// or returned by DNS may be spelled differently. Comparing the TEXT would
	// miss 2001:0db8::1 against 2001:db8::1.
	dialedAt := time.Now()
	snapshot := &clashapi.Snapshot{Connections: []clashapi.Connection{
		conn("a", "2001:0db8:0000::1", "443", dialedAt.Add(time.Millisecond), "ok", []string{"direct"}),
	}}

	if observed := Correlate(snapshot, netip.MustParseAddr("2001:db8::1"), 443, dialedAt); observed == nil {
		t.Fatal("equivalent IPv6 spellings should correlate")
	}
}

func TestCorrelateHandlesAnEmptySnapshot(t *testing.T) {
	if observed := Correlate(nil, netip.MustParseAddr("1.2.3.4"), 443, time.Now()); observed != nil {
		t.Fatal("a nil snapshot must not match anything")
	}
}
