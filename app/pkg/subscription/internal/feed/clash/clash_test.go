package clash

import (
	"encoding/json"
	"strings"
	"testing"
)

// sampleProfile mirrors the shape real providers ship: a header comment, the
// Clash runtime keys the importer must ignore, then a mixed `proxies:` list —
// including one type sing-box has no outbound for.
const sampleProfile = `
# Provider header
port: 7890
mode: rule
proxies:
  - {"name":"TW 01","type":"anytls","server":"s4.example.com","port":37231,"password":"pw-anytls","client-fingerprint":"chrome","udp":true,"idle-session-check-interval":30,"idle-session-timeout":30,"min-idle-session":5,"sni":"storage.example.net","skip-cert-verify":true}
  - name: "SS 01"
    type: ss
    server: ss.example.com
    port: 8388
    cipher: aes-128-gcm
    password: pw-ss
  - name: "VMESS WS"
    type: vmess
    server: vm.example.com
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    alterId: 0
    cipher: auto
    tls: true
    servername: cdn.example.com
    network: ws
    ws-opts:
      path: /path
      headers:
        Host: cdn.example.com
  - name: "TROJAN 01"
    type: trojan
    server: tj.example.com
    port: 443
    password: pw-trojan
    sni: tj.example.com
    skip-cert-verify: true
  - name: "VLESS REALITY"
    type: vless
    server: re.example.com
    port: 35248
    uuid: 00000000-0000-0000-0000-000000000000
    flow: xtls-rprx-vision
    servername: www.example.org
    client-fingerprint: safari
    reality-opts:
      public-key: TESTPUBLICKEY
      short-id: abcd1234
  - name: "HY2 01"
    type: hysteria2
    server: hy.example.com
    port: 60000
    password: pw-hy2
    ports: "60000-65530"
    obfs: salamander
    obfs-password: pw-obfs
    sni: apps.example.com
    up: "50 Mbps"
    down: 200
  - name: "SSR 01"
    type: ssr
    server: ssr.example.com
    port: 8388
`

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"clash profile", sampleProfile, true},
		{"uri list", "vless://x@example.com:443#a\nss://y@example.com:8388#b", false},
		{"base64 blob", "dmxlc3M6Ly94QGV4YW1wbGUuY29tOjQ0Mw==", false},
		{"empty", "", false},
		// "proxies:" appears as prose but there is no list — must not match, or
		// a genuine format error is reported as "no proxies found".
		{"mentions proxies in a comment", "# proxies: see the docs\nfoo: bar", false},
		{"proxy-providers only", "proxy-providers:\n  a:\n    url: http://x\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Detect([]byte(tt.body)); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMixedProfile(t *testing.T) {
	nodes, skipped, err := Parse([]byte(sampleProfile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(nodes) != 6 {
		t.Fatalf("imported %d nodes, want 6", len(nodes))
	}
	if len(skipped) != 1 || skipped[0].Type != "ssr" {
		t.Fatalf("skipped = %v, want one ssr entry", skipped)
	}

	byTag := map[string]string{}
	for _, n := range nodes {
		b, err := json.Marshal(n)
		if err != nil {
			t.Fatalf("marshal %s: %v", n.Tag, err)
		}
		byTag[n.Tag] = string(b)
		t.Logf("[%s] %s", n.Type, b)
	}

	checks := []struct {
		tag   string
		wants []string
	}{
		{"TW 01", []string{
			`"type":"anytls"`, `"server":"s4.example.com"`, `"server_port":37231`,
			`"password":"pw-anytls"`, `"server_name":"storage.example.net"`,
			`"insecure":true`, `"fingerprint":"chrome"`,
			`"idle_session_check_interval":"30s"`, `"min_idle_session":5`,
		}},
		{"SS 01", []string{`"type":"shadowsocks"`, `"method":"aes-128-gcm"`, `"password":"pw-ss"`}},
		{"VMESS WS", []string{
			`"type":"vmess"`, `"security":"auto"`, `"tls":{"enabled":true`,
			`"server_name":"cdn.example.com"`, `"type":"ws"`, `"path":"/path"`, `"Host":"cdn.example.com"`,
		}},
		{"TROJAN 01", []string{`"type":"trojan"`, `"password":"pw-trojan"`, `"insecure":true`}},
		{"VLESS REALITY", []string{
			`"type":"vless"`, `"flow":"xtls-rprx-vision"`,
			`"reality":{"enabled":true,"public_key":"TESTPUBLICKEY","short_id":"abcd1234"}`,
			`"fingerprint":"safari"`,
		}},
		{"HY2 01", []string{
			`"type":"hysteria2"`, `"server_ports":["60000:65530"]`,
			`"obfs":{"password":"pw-obfs","type":"salamander"}`,
			`"up_mbps":50`, `"down_mbps":200`,
		}},
	}
	for _, c := range checks {
		got, ok := byTag[c.tag]
		if !ok {
			t.Errorf("node %q missing from import", c.tag)
			continue
		}
		for _, want := range c.wants {
			if !strings.Contains(got, want) {
				t.Errorf("node %q missing %s\ngot: %s", c.tag, want, got)
			}
		}
	}
}

// Trojan/hysteria2/anytls are TLS-framed, so Clash omits `tls: true`. Dropping
// the TLS block for them would produce an outbound that dials plaintext.
func TestTLSForcedOnTLSFramedTypes(t *testing.T) {
	const body = `
proxies:
  - {name: "T", type: trojan, server: a.example.com, port: 443, password: pw}
  - {name: "H", type: hysteria2, server: b.example.com, port: 443, password: pw}
  - {name: "A", type: anytls, server: c.example.com, port: 443, password: pw}
  - {name: "V", type: vmess, server: d.example.com, port: 443, uuid: 0-0-0-0-0}
`
	nodes, _, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, n := range nodes {
		b, _ := json.Marshal(n)
		hasTLS := strings.Contains(string(b), `"tls":{"enabled":true`)
		// vmess has no `tls: true` in this profile, so it must NOT get a block.
		want := n.Tag != "V"
		if hasTLS != want {
			t.Errorf("node %q tls=%v, want %v: %s", n.Tag, hasTLS, want, b)
		}
	}
}

func TestParseErrors(t *testing.T) {
	if _, _, err := Parse([]byte("port: 7890\n")); err == nil {
		t.Error("profile with no proxies should error")
	}
	if _, _, err := Parse([]byte("\t- not: yaml\n  bad")); err == nil {
		t.Error("invalid yaml should error")
	}
	// Every proxy unsupported: the caller must get an error, not an empty list.
	_, skipped, err := Parse([]byte("proxies:\n  - {name: A, type: ssr, server: a.com, port: 1}\n"))
	if err == nil {
		t.Error("all-unsupported profile should error")
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %d, want 1", len(skipped))
	}
}

// A proxy with no name still has to land in config.json with a usable tag —
// an empty outbound tag is rejected by sing-box.
func TestUnnamedProxyGetsFallbackTag(t *testing.T) {
	nodes, _, err := Parse([]byte("proxies:\n  - {type: trojan, server: a.example.com, port: 443, password: pw}\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if nodes[0].Tag != "trojan-a.example.com-443" {
		t.Errorf("Tag = %q, want trojan-a.example.com-443", nodes[0].Tag)
	}
}

// Transports other than ws, and the shadowsocks plugin forms. Clash nests
// these settings; sing-box wants a flat transport block and a plugin's own
// "k=v;k=v" string, so the mapping is where a silent mistranslation would hide.
func TestTransportAndPluginMapping(t *testing.T) {
	const body = `
proxies:
  - name: GRPC
    type: vless
    server: g.example.com
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    tls: true
    network: grpc
    grpc-opts:
      grpc-service-name: TunService
  - name: H2
    type: vmess
    server: h.example.com
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    tls: true
    network: h2
    h2-opts:
      host:
        - a.example.com
        - b.example.com
      path: /h2path
  - name: TCP
    type: vmess
    server: t.example.com
    port: 443
    uuid: 00000000-0000-0000-0000-000000000000
    network: tcp
  - name: OBFS
    type: ss
    server: o.example.com
    port: 8388
    cipher: aes-128-gcm
    password: pw
    plugin: obfs
    plugin-opts:
      mode: http
      host: bing.com
  - name: V2RAYPLUGIN
    type: ss
    server: p.example.com
    port: 8388
    cipher: aes-128-gcm
    password: pw
    plugin: v2ray-plugin
    plugin-opts:
      mode: websocket
      host: cdn.example.com
      path: /ws
      tls: true
`
	nodes, skipped, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped = %v, want none", skipped)
	}

	byTag := map[string]string{}
	for _, n := range nodes {
		b, _ := json.Marshal(n)
		byTag[n.Tag] = string(b)
	}

	checks := []struct {
		tag        string
		wants      []string
		wantAbsent []string
	}{
		{"GRPC", []string{`"type":"grpc"`, `"service_name":"TunService"`}, nil},
		{"H2", []string{`"type":"http"`, `"host":["a.example.com","b.example.com"]`, `"path":"/h2path"`}, nil},
		// tcp is the absence of a transport, not a transport named "tcp".
		{"TCP", nil, []string{`"transport"`}},
		{"OBFS", []string{`"plugin":"obfs-local"`, `"plugin_opts":"obfs=http;obfs-host=bing.com"`}, nil},
		{"V2RAYPLUGIN", []string{`"plugin":"v2ray-plugin"`, `"plugin_opts":"mode=websocket;host=cdn.example.com;path=/ws;tls=true"`}, nil},
	}
	for _, c := range checks {
		got, ok := byTag[c.tag]
		if !ok {
			t.Errorf("node %q missing", c.tag)
			continue
		}
		for _, want := range c.wants {
			if !strings.Contains(got, want) {
				t.Errorf("node %q missing %s\ngot: %s", c.tag, want, got)
			}
		}
		for _, absent := range c.wantAbsent {
			if strings.Contains(got, absent) {
				t.Errorf("node %q should not contain %s\ngot: %s", c.tag, absent, got)
			}
		}
	}
}

// Scalars arrive with whatever YAML type the provider wrote. A numeric
// password must not be dropped, and a quoted port must still parse.
func TestScalarCoercion(t *testing.T) {
	const body = `
proxies:
  - {name: N, type: trojan, server: a.example.com, port: "443", password: 123456, alpn: "h2,http/1.1", skip-cert-verify: "true"}
`
	nodes, _, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	b, _ := json.Marshal(nodes[0])
	for _, want := range []string{
		`"server_port":443`,
		`"password":"123456"`,
		`"alpn":["h2","http/1.1"]`,
		`"insecure":true`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s\ngot: %s", want, b)
		}
	}
}

// One entry that is not a mapping at all (a bare string, a null, a
// placeholder some panels emit) makes yaml.v3 report a TypeError — but only
// after decoding every sibling it could. Treating that as fatal threw away a
// whole subscription over one bad row.
func TestStrayEntryDoesNotDropTheProfile(t *testing.T) {
	const body = `
proxies:
  - {name: A, type: trojan, server: a.example.com, port: 443, password: pw}
  - just-a-string
  - null
  - {name: B, type: trojan, server: b.example.com, port: 443, password: pw}
`
	if !Detect([]byte(body)) {
		t.Fatal("Detect() = false, want true: the profile is still a clash profile")
	}
	nodes, skipped, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("imported %d nodes, want the 2 valid ones", len(nodes))
	}
	// The unreadable rows are reported, not silently dropped.
	if len(skipped) == 0 {
		t.Error("skipped is empty, want the unreadable entries reported")
	}
	for _, s := range skipped {
		t.Logf("skipped: %s", s)
	}
}

// A real syntax error is still fatal — nothing decoded, nothing to salvage.
func TestSyntaxErrorIsFatal(t *testing.T) {
	if Detect([]byte("proxies:\n\t- {a: b}\n")) {
		t.Error("Detect() = true on a body with a yaml syntax error")
	}
	if _, _, err := Parse([]byte("proxies:\n\t- {a: b}\n")); err == nil {
		t.Error("Parse() = nil error on a body with a yaml syntax error")
	}
}

// sing-box treats an outbound tag as a primary key: a duplicate fails
// `sing-box check` and rolls back the whole config update, naming the tag but
// not the two feed entries that collided. Providers do repeat names.
func TestDuplicateTagsAreMadeUnique(t *testing.T) {
	const body = `
proxies:
  - {name: "TW 01", type: trojan, server: a.example.com, port: 443, password: pw}
  - {name: "TW 01", type: trojan, server: b.example.com, port: 443, password: pw}
  - {name: "TW 01", type: trojan, server: c.example.com, port: 443, password: pw}
  - {type: trojan, server: d.example.com, port: 443, password: pw}
  - {type: trojan, server: d.example.com, port: 8443, password: pw}
`
	nodes, _, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(nodes) != 5 {
		t.Fatalf("imported %d nodes, want 5", len(nodes))
	}
	seen := map[string]bool{}
	for _, n := range nodes {
		if n.Tag == "" {
			t.Error("node has an empty tag")
		}
		if seen[n.Tag] {
			t.Errorf("duplicate tag %q", n.Tag)
		}
		seen[n.Tag] = true
	}
	// The first keeps the provider's name; only the collisions are suffixed.
	if !seen["TW 01"] || !seen["TW 01-2"] || !seen["TW 01-3"] {
		t.Errorf("unexpected tag set: %v", seen)
	}
	// Anonymous proxies on one host are told apart by port, not by suffix.
	if !seen["trojan-d.example.com-443"] || !seen["trojan-d.example.com-8443"] {
		t.Errorf("port-hopping proxies collapsed: %v", seen)
	}
}
