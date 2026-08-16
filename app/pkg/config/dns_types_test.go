package config

import (
	"context"
	"reflect"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// TestDNSTypesAreRegistered 守住 DNSTypes 与 CreateDNSOptions 的一致性。
func TestDNSTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(DNSTypes) == 0 {
		t.Fatal("DNSTypes is empty")
	}

	seen := make(map[string]bool, len(DNSTypes))
	for _, dnsType := range DNSTypes {
		if seen[dnsType] {
			t.Errorf("duplicate entry %q in DNSTypes", dnsType)
		}
		seen[dnsType] = true

		options, ok := r.CreateDNSOptions(dnsType)
		if !ok {
			t.Errorf("CreateDNSOptions(%q) = not registered", dnsType)
			continue
		}

		rt := reflect.TypeOf(options)
		if rt == nil || rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("CreateDNSOptions(%q) returned %v; want pointer to struct", dnsType, rt)
		}

		if !IsKnownDNSType(dnsType) {
			t.Errorf("IsKnownDNSType(%q) = false for a listed type", dnsType)
		}
	}
}

// TestDNSTypeHTTP3IsH3 pins the exact regression that made this file necessary.
//
// The registry used to case on "http3". sing-box's constant is "h3" and its
// transport registers under that name, so the panel could neither open a valid
// config nor write a startable one. A literal is used on purpose here: asserting
// against C.DNSTypeHTTP3 on both sides would pass even if the constant changed.
func TestDNSTypeHTTP3IsH3(t *testing.T) {
	if C.DNSTypeHTTP3 != "h3" {
		t.Fatalf("sing-box changed DNSTypeHTTP3 to %q; the frontend spelling must follow", C.DNSTypeHTTP3)
	}

	r := &Registry{}

	if _, ok := r.CreateDNSOptions("h3"); !ok {
		t.Error(`CreateDNSOptions("h3") = not registered; a valid config would fail to parse`)
	}

	if _, ok := r.CreateDNSOptions("http3"); ok {
		t.Error(`CreateDNSOptions("http3") = registered; sing-box has no such transport, so this writes unstartable configs`)
	}
}

// TestDNSServerRoundTrip parses one server of every type through the same
// context the config manager uses.
//
// This is the check that would have caught both shipped bugs: "h3" failing to
// parse, and the handlers producing a server whose Options is nil and therefore
// cannot be marshalled back.
func TestDNSServerRoundTrip(t *testing.T) {
	cases := map[string]string{
		C.DNSTypeUDP:       `{"type":"udp","tag":"t","server":"1.1.1.1","server_port":53}`,
		C.DNSTypeTCP:       `{"type":"tcp","tag":"t","server":"1.1.1.1"}`,
		C.DNSTypeTLS:       `{"type":"tls","tag":"t","server":"8.8.8.8"}`,
		C.DNSTypeQUIC:      `{"type":"quic","tag":"t","server":"8.8.8.8"}`,
		C.DNSTypeHTTPS:     `{"type":"https","tag":"t","server":"1.1.1.1","path":"/dns-query"}`,
		C.DNSTypeHTTP3:     `{"type":"h3","tag":"t","server":"1.1.1.1","path":"/dns-query"}`,
		C.DNSTypeLocal:     `{"type":"local","tag":"t"}`,
		C.DNSTypeHosts:     `{"type":"hosts","tag":"t","predefined":{"a.example":"192.0.2.1"}}`,
		C.DNSTypeFakeIP:    `{"type":"fakeip","tag":"t","inet4_range":"198.18.0.0/15"}`,
		C.DNSTypeDHCP:      `{"type":"dhcp","tag":"t","interface":"eth0"}`,
		C.DNSTypeTailscale: `{"type":"tailscale","tag":"t"}`,
	}

	ctx := CreateContext(context.Background())

	for dnsType, raw := range cases {
		t.Run(dnsType, func(t *testing.T) {
			var server option.DNSServerOptions
			if err := json.UnmarshalContext(ctx, []byte(raw), &server); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if server.Type != dnsType {
				t.Errorf("Type = %q, want %q", server.Type, dnsType)
			}

			// The nil-Options case is exactly what c.Bind produced, and what
			// made every add/edit fail at marshal time.
			if server.Options == nil {
				t.Fatal("Options is nil; the server cannot be marshalled back")
			}

			out, err := json.MarshalContext(ctx, &server)
			if err != nil {
				t.Fatalf("marshal back: %v", err)
			}
			if len(out) == 0 {
				t.Fatal("marshalled to nothing")
			}
		})
	}
}

// TestDNSServerRoundTripPreservesFields guards the specific data loss: binding
// with reflection kept type and tag and dropped everything else.
func TestDNSServerRoundTripPreservesFields(t *testing.T) {
	ctx := CreateContext(context.Background())

	var server option.DNSServerOptions
	raw := `{"type":"tls","tag":"probe","server":"9.9.9.9","server_port":853,"detour":"direct"}`
	if err := json.UnmarshalContext(ctx, []byte(raw), &server); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	options, ok := server.Options.(*option.RemoteTLSDNSServerOptions)
	if !ok {
		t.Fatalf("Options is %T, want *option.RemoteTLSDNSServerOptions", server.Options)
	}

	if options.Server != "9.9.9.9" {
		t.Errorf("Server = %q, want 9.9.9.9", options.Server)
	}
	if options.ServerPort != 853 {
		t.Errorf("ServerPort = %d, want 853", options.ServerPort)
	}
	if options.Detour != "direct" {
		t.Errorf("Detour = %q, want direct", options.Detour)
	}
}

// TestLegacyDNSServerUpgrades documents why the form never edits the legacy
// shape: sing-box rewrites it to a typed server on read, before the UI sees it.
func TestLegacyDNSServerUpgrades(t *testing.T) {
	ctx := CreateContext(context.Background())

	var server option.DNSServerOptions
	if err := json.UnmarshalContext(ctx, []byte(`{"tag":"legacy","address":"tls://1.1.1.1"}`), &server); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if server.Type != C.DNSTypeTLS {
		t.Errorf("Type = %q, want %q — legacy should upgrade on read", server.Type, C.DNSTypeTLS)
	}
	if _, ok := server.Options.(*option.RemoteTLSDNSServerOptions); !ok {
		t.Errorf("Options is %T, want *option.RemoteTLSDNSServerOptions", server.Options)
	}
}
