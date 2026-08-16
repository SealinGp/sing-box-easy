package config

import (
	"context"
	"reflect"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// TestOutboundTypesAreRegistered 守住 OutboundTypes 与 CreateOutboundOptions 的一致性。
func TestOutboundTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(OutboundTypes) == 0 {
		t.Fatal("OutboundTypes is empty")
	}

	seen := make(map[string]bool, len(OutboundTypes))
	for _, outboundType := range OutboundTypes {
		if seen[outboundType] {
			t.Errorf("duplicate entry %q in OutboundTypes", outboundType)
		}
		seen[outboundType] = true

		options, ok := r.CreateOutboundOptions(outboundType)
		if !ok {
			t.Errorf("CreateOutboundOptions(%q) = not registered", outboundType)
			continue
		}
		if !IsKnownOutboundType(outboundType) {
			t.Errorf("IsKnownOutboundType(%q) = false for a listed type", outboundType)
		}

		rt := reflect.TypeOf(options)
		if rt == nil || rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("CreateOutboundOptions(%q) returned %v; want pointer to struct", outboundType, rt)
		}
	}
}

// TestOutboundTypesCoversRegistryDrift pins the three types that were
// constructible on the backend while the frontend's hardcoded list never
// offered them.
func TestOutboundTypesCoversRegistryDrift(t *testing.T) {
	for _, outboundType := range []string{"anytls", "dns", "shadowsocksr"} {
		if !IsKnownOutboundType(outboundType) {
			t.Errorf("outbound type %q is registered but missing from OutboundTypes", outboundType)
		}
	}
}

// TestOutboundGroupTypes keeps the group classification in one place. The
// frontend previously carried three copies of this list and the backend a
// fourth.
func TestOutboundGroupTypes(t *testing.T) {
	if !IsOutboundGroupType(C.TypeSelector) || !IsOutboundGroupType(C.TypeURLTest) {
		t.Error("selector and urltest must classify as group types")
	}
	for _, notGroup := range []string{C.TypeDirect, C.TypeBlock, C.TypeShadowsocks} {
		if IsOutboundGroupType(notGroup) {
			t.Errorf("%q must not classify as a group type", notGroup)
		}
	}
}

// TestOutboundRoundTrip parses one outbound of every type through the same
// context the config manager uses, and asserts Options survives — the check
// that caught the DNS server handlers writing a nil Options.
//
// `block` and `dns` map to option.StubOptions, which is `struct{}`. They are
// still expected to produce a non-nil Options; a field-less type is not an
// absent one.
func TestOutboundRoundTrip(t *testing.T) {
	cases := map[string]string{
		C.TypeDirect:       `{"type":"direct","tag":"t"}`,
		C.TypeBlock:        `{"type":"block","tag":"t"}`,
		C.TypeDNS:          `{"type":"dns","tag":"t"}`,
		C.TypeSOCKS:        `{"type":"socks","tag":"t","server":"1.1.1.1","server_port":1080}`,
		C.TypeHTTP:         `{"type":"http","tag":"t","server":"1.1.1.1","server_port":8080}`,
		C.TypeShadowsocks:  `{"type":"shadowsocks","tag":"t","server":"1.1.1.1","server_port":8388,"method":"aes-128-gcm","password":"p"}`,
		C.TypeVMess:        `{"type":"vmess","tag":"t","server":"1.1.1.1","server_port":443,"uuid":"b831381d-6324-4d53-ad4f-8cda48b30811"}`,
		C.TypeVLESS:        `{"type":"vless","tag":"t","server":"1.1.1.1","server_port":443,"uuid":"b831381d-6324-4d53-ad4f-8cda48b30811"}`,
		C.TypeTrojan:       `{"type":"trojan","tag":"t","server":"1.1.1.1","server_port":443,"password":"p"}`,
		C.TypeHysteria:     `{"type":"hysteria","tag":"t","server":"1.1.1.1","server_port":443,"up_mbps":10,"down_mbps":50}`,
		C.TypeHysteria2:    `{"type":"hysteria2","tag":"t","server":"1.1.1.1","server_port":443,"password":"p"}`,
		C.TypeTUIC:         `{"type":"tuic","tag":"t","server":"1.1.1.1","server_port":443,"uuid":"b831381d-6324-4d53-ad4f-8cda48b30811"}`,
		C.TypeShadowTLS:    `{"type":"shadowtls","tag":"t","server":"1.1.1.1","server_port":443}`,
		C.TypeAnyTLS:       `{"type":"anytls","tag":"t","server":"1.1.1.1","server_port":443,"password":"p"}`,
		C.TypeShadowsocksR: `{"type":"shadowsocksr","tag":"t","server":"1.1.1.1","server_port":8388,"method":"aes-128-cfb","password":"p"}`,
		C.TypeWireGuard:    `{"type":"wireguard","tag":"t","server":"1.1.1.1","server_port":51820,"private_key":"k","peer_public_key":"k","local_address":["10.0.0.2/32"]}`,
		C.TypeSSH:          `{"type":"ssh","tag":"t","server":"1.1.1.1"}`,
		C.TypeTor:          `{"type":"tor","tag":"t"}`,
		C.TypeSelector:     `{"type":"selector","tag":"t","outbounds":["a","b"]}`,
		C.TypeURLTest:      `{"type":"urltest","tag":"t","outbounds":["a","b"]}`,
	}

	ctx := CreateContext(context.Background())

	for outboundType, raw := range cases {
		t.Run(outboundType, func(t *testing.T) {
			var outbound option.Outbound
			if err := json.UnmarshalContext(ctx, []byte(raw), &outbound); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if outbound.Type != outboundType {
				t.Errorf("Type = %q, want %q", outbound.Type, outboundType)
			}
			if outbound.Options == nil {
				t.Fatal("Options is nil; the outbound cannot be marshalled back")
			}
			if _, err := json.MarshalContext(ctx, &outbound); err != nil {
				t.Fatalf("marshal back: %v", err)
			}
		})
	}

	if len(cases) != len(OutboundTypes) {
		t.Errorf("round-trip covers %d types but OutboundTypes lists %d", len(cases), len(OutboundTypes))
	}
}
