package sublink

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// TestPasteBase64Subscription verifies that a base64-encoded subscription body
// pasted directly (not via URL) is decoded and parsed, including the newly
// added vless:// and hysteria2:// protocols. Mirrors the /nodes/parse flow.
func TestPasteBase64Subscription(t *testing.T) {
	// Dummy credentials/hosts — shaped like a real subscription but with no
	// secrets. Exercises: vless ws+tls, vless reality (flow/pbk/sid), hysteria2
	// port-hopping (mport).
	const uuid = "00000000-0000-0000-0000-000000000000"
	uris := []string{
		"vless://" + uuid + "@ws.example.com:443?type=ws&encryption=none&host=cdn.example.com&path=%2Fpath&security=tls&fp=safari&insecure=0&sni=cdn.example.com#JP-ws-tls",
		"vless://" + uuid + "@reality.example.com:35248?type=tcp&encryption=none&security=reality&flow=xtls-rprx-vision&fp=safari&sni=www.example.org&pbk=TESTPUBLICKEY_dummy_value_000000000000000000&sid=abcd1234#HK-reality",
		"hysteria2://" + uuid + "@hy.example.com:60000/?insecure=1&sni=apps.example.com&mport=60000-65530#HY-jp",
	}
	blob := base64.StdEncoding.EncodeToString([]byte(strings.Join(uris, "\n")))

	l := new(SubLink)
	nodes, err := l.ListNodes([]string{blob}) // single pasted line, like the UI
	if err != nil {
		t.Fatalf("ListNodes returned error: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	byType := map[string]int{}
	for _, n := range nodes {
		byType[n.Type]++
		b, _ := json.Marshal(n)
		t.Logf("[%s] %s -> %s", n.Type, n.Tag, string(b))
	}
	if byType["vless"] != 2 || byType["hysteria2"] != 1 {
		t.Fatalf("type counts wrong: %v", byType)
	}

	// Spot-check the reality node carries pbk/sid/flow.
	for _, n := range nodes {
		if n.Tag == "HK-reality" {
			b, _ := json.Marshal(n)
			s := string(b)
			for _, want := range []string{"reality", "TESTPUBLICKEY_dummy_value_000000000000000000", "abcd1234", "xtls-rprx-vision"} {
				if !strings.Contains(s, want) {
					t.Errorf("reality node missing %q in %s", want, s)
				}
			}
		}
	}
}
