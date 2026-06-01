package protocol

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

// TestNormalizeFlow verifies that Xray-specific flow variants are mapped to the
// only value sing-box accepts ("xtls-rprx-vision"), and that unknown/unsupported
// flows are dropped so they never reach `sing-box check`, which would otherwise
// fail with "unsupported flow: ...".
func TestNormalizeFlow(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"vision", "xtls-rprx-vision", "xtls-rprx-vision"},
		{"udp443 variant", "xtls-rprx-vision-udp443", "xtls-rprx-vision"},
		{"whitespace trimmed", "  xtls-rprx-vision-udp443  ", "xtls-rprx-vision"},
		{"legacy direct dropped", "xtls-rprx-direct", ""},
		{"unknown dropped", "some-future-flow", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeFlow(tc.in); got != tc.want {
				t.Errorf("normalizeFlow(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestVLESSParseNormalizesFlow ensures the udp443 variant coming from a real
// subscription link is normalized during parsing.
func TestVLESSParseNormalizesFlow(t *testing.T) {
	uri := "vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443" +
		"?type=tcp&security=reality&flow=xtls-rprx-vision-udp443" +
		"&pbk=TESTPUBLICKEY_dummy_value_000000000000000000&sid=abcd1234&sni=www.example.org&fp=chrome#node"

	n, err := new(VLESS).Parse(uri)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	opts, ok := n.Options.(option.VLESSOutboundOptions)
	if !ok {
		t.Fatalf("Options is %T, want option.VLESSOutboundOptions", n.Options)
	}
	if opts.Flow != "xtls-rprx-vision" {
		t.Errorf("parsed flow = %q, want %q", opts.Flow, "xtls-rprx-vision")
	}
}
