package protocol

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnyTLSParse(t *testing.T) {
	const pw = "47aae2bb-ce10-316d-9caa-0d3bec6e6d39"
	uri := "anytls://" + pw + "@s4.example.com:37231/?sni=storage.example.net&insecure=1&fp=chrome#" +
		"%E5%8F%B0%E6%B9%BE%2001"

	parser, err := NewParser(uri)
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	n, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if n.Type != "anytls" {
		t.Errorf("Type = %q, want anytls", n.Type)
	}
	if n.Tag != "台湾 01" {
		t.Errorf("Tag = %q, want %q", n.Tag, "台湾 01")
	}

	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{
		`"server":"s4.example.com"`,
		`"server_port":37231`,
		`"password":"` + pw + `"`,
		`"server_name":"storage.example.net"`,
		`"insecure":true`,
		`"fingerprint":"chrome"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("marshaled node missing %s\ngot: %s", want, s)
		}
	}
}

// TLS is unconditional for anytls: a URI with no query at all must still emit
// an enabled TLS block, or sing-box dials plaintext and the handshake fails.
func TestAnyTLSAlwaysEnablesTLS(t *testing.T) {
	parser, err := NewParser("anytls://pw@example.com:443")
	if err != nil {
		t.Fatalf("NewParser: %v", err)
	}
	n, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	b, _ := json.Marshal(n)
	if !strings.Contains(string(b), `"tls":{"enabled":true}`) {
		t.Errorf("want enabled tls block, got: %s", b)
	}
}

func TestAnyTLSValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"missing @", "anytls://example.com:443"},
		{"missing port", "anytls://pw@example.com"},
		{"port out of range", "anytls://pw@example.com:70000"},
		{"empty password", "anytls://@example.com:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.uri)
			if err != nil {
				t.Fatalf("NewParser: %v", err)
			}
			if _, err := parser.Parse(); err == nil {
				t.Errorf("Parse(%q) = nil error, want failure", tt.uri)
			}
		})
	}
}

// The password half may legally contain "/" — these secrets are often raw
// base64, whose alphabet includes it. Stripping the path before splitting on
// "@" cut the link at the password and lost the server entirely; the node then
// vanished from an otherwise valid subscription with no error the user sees,
// because parseBody swallows per-line failures by design.
func TestUserinfoURIKeepsSlashInPassword(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string // expected password
	}{
		{"anytls slash", "anytls://ab/cd@s4.example.com:37231/?sni=x.example.net#T", "ab/cd"},
		{"anytls no path", "anytls://ab/cd@s4.example.com:37231?sni=x.example.net#T", "ab/cd"},
		{"anytls base64 password", "anytls://aGVsbG8vd29ybGQ/x@s4.example.com:37231#T", "aGVsbG8vd29ybGQ/x"},
		{"hysteria2 slash", "hysteria2://ab/cd@s4.example.com:60000/?sni=x.example.net#T", "ab/cd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser(tt.uri)
			if err != nil {
				t.Fatalf("NewParser: %v", err)
			}
			n, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.uri, err)
			}
			b, _ := json.Marshal(n)
			if !strings.Contains(string(b), `"password":"`+tt.want+`"`) {
				t.Errorf("password not preserved, want %q\ngot: %s", tt.want, b)
			}
			if !strings.Contains(string(b), `"server":"s4.example.com"`) {
				t.Errorf("server lost\ngot: %s", b)
			}
		})
	}
}

// An "@" inside the password must lose to the real separator.
func TestUserinfoURIHandlesAtInPassword(t *testing.T) {
	parser, _ := NewParser("anytls://p@ss@s4.example.com:37231#T")
	n, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	b, _ := json.Marshal(n)
	if !strings.Contains(string(b), `"password":"p@ss"`) ||
		!strings.Contains(string(b), `"server":"s4.example.com"`) {
		t.Errorf("unexpected split: %s", b)
	}
}
