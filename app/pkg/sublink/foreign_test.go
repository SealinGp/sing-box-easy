package sublink

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestShouldRetryWithClientUA(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{200, false},
		{301, false},
		{401, true}, // some panels answer 401 to an unknown client
		{403, true},
		{404, true},  // the shape seen in the wild: unknown UA -> 404
		{429, false}, // the panel asking to be left alone
		{500, false}, // the panel accepted the request and then failed
		{503, false},
	}
	for _, tt := range tests {
		if got := shouldRetryWithClientUA(tt.status); got != tt.want {
			t.Errorf("shouldRetryWithClientUA(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

// The provider shape that motivated the importer: a Clash-only endpoint whose
// body is a YAML profile of anytls nodes.
const clashBody = `
#---------------------------------------------------#
## provider header
port: 7890
proxy-groups:
  - {name: PROXY, type: select, proxies: [TW 01]}
proxies:
  - {"name":"TW 01","type":"anytls","server":"s4.example.com","port":37231,"password":"pw","sni":"storage.example.net","skip-cert-verify":true}
  - {"name":"TW 02","type":"anytls","server":"s4.example.com","port":37232,"password":"pw","sni":"storage.example.net","skip-cert-verify":true}
rules:
  - MATCH,PROXY
`

func TestParseNonBase64BodyClash(t *testing.T) {
	l := new(SubLink)
	nodes, err := l.parseNonBase64Body([]byte(clashBody), "test")
	if err != nil {
		t.Fatalf("parseNonBase64Body: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	b, _ := json.Marshal(nodes[0])
	if !strings.Contains(string(b), `"type":"anytls"`) {
		t.Errorf("first node is not anytls: %s", b)
	}
	// proxy-groups must not leak in as nodes.
	for _, n := range nodes {
		if n.Tag == "PROXY" {
			t.Error("proxy-group imported as a node")
		}
	}
}

func TestParseNonBase64BodyPlainURIList(t *testing.T) {
	body := strings.Join([]string{
		"trojan://pw@a.example.com:443#A",
		"anytls://pw@b.example.com:37231?sni=x.example.net#B",
	}, "\n")

	l := new(SubLink)
	nodes, err := l.parseNonBase64Body([]byte(body), "test")
	if err != nil {
		t.Fatalf("parseNonBase64Body: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
}

func TestParseNonBase64BodyRejectsGarbage(t *testing.T) {
	l := new(SubLink)
	for _, body := range []string{"", "<html><body>404</body></html>", "just some text"} {
		if _, err := l.parseNonBase64Body([]byte(body), "test"); err == nil {
			t.Errorf("parseNonBase64Body(%q) = nil error, want failure", body)
		}
	}
}

// A pasted Clash profile must import through the public entry point too, not
// just the fetch path.
func TestListNodesAcceptsPastedClashProfile(t *testing.T) {
	l := new(SubLink)
	nodes, err := l.ListNodes([]string{clashBody})
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
}
