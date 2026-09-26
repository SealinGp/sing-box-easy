package config

import (
	"reflect"
	"testing"
)

// TestInboundTypesAreRegistered 守住 InboundTypes 与 CreateInboundOptions 的一致性。
//
// The schema generator trusts InboundTypes completely: whatever is listed there
// gets reflected and shipped to the frontend as the authoritative field
// inventory. An entry with no matching case in CreateInboundOptions would
// silently emit an empty field set, and the inbound form for that type would
// render nothing — with no error anywhere. Fail here instead.
func TestInboundTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(InboundTypes) == 0 {
		t.Fatal("InboundTypes is empty")
	}

	seen := make(map[string]bool, len(InboundTypes))
	for _, inboundType := range InboundTypes {
		if seen[inboundType] {
			t.Errorf("duplicate entry %q in InboundTypes", inboundType)
		}
		seen[inboundType] = true

		options, ok := r.CreateInboundOptions(inboundType)
		if !ok {
			t.Errorf("CreateInboundOptions(%q) = not registered; listed in InboundTypes but has no case", inboundType)
			continue
		}

		// A registered type must yield a struct pointer for reflection to walk.
		// Anything else (nil, a bare map) would produce a field-less type in the
		// generated inventory.
		rt := reflect.TypeOf(options)
		if rt == nil || rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("CreateInboundOptions(%q) returned %v; want pointer to struct", inboundType, rt)
		}
	}
}

// TestInboundTypesCoversKnownRegistrations catches the direction the type switch
// cannot report on its own: a type that CreateInboundOptions handles but nobody
// listed. It is a hand-maintained lower bound rather than a true inverse — Go
// offers no way to enumerate switch cases — so it only guards the types we know
// exist today. "anytls" is called out because it is the one that actually
// slipped: registered on the backend since 1.12 and absent from the frontend.
func TestInboundTypesCoversKnownRegistrations(t *testing.T) {
	mustList := []string{
		"tun", "redirect", "tproxy", "direct",
		"mixed", "http", "socks",
		"shadowsocks", "vmess", "vless", "trojan", "naive",
		"hysteria", "hysteria2", "tuic", "shadowtls", "anytls",
	}

	listed := make(map[string]bool, len(InboundTypes))
	for _, inboundType := range InboundTypes {
		listed[inboundType] = true
	}

	for _, inboundType := range mustList {
		if !listed[inboundType] {
			t.Errorf("inbound type %q is registered but missing from InboundTypes", inboundType)
		}
	}
}
