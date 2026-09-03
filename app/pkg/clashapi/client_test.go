package clashapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestControllerURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "127.0.0.1:9090", want: "http://127.0.0.1:9090"},
		{in: "192.168.9.253:9095", want: "http://192.168.9.253:9095"},
		// A bind address is not a reachable one; the panel shares the host.
		{in: ":9090", want: "http://127.0.0.1:9090"},
		{in: "0.0.0.0:9090", want: "http://127.0.0.1:9090"},
		{in: "[::]:9090", want: "http://127.0.0.1:9090"},
		{in: "http://example:9090/", want: "http://example:9090"},
		{in: "nonsense", wantErr: true},
	}
	for _, c := range cases {
		got, err := ControllerURL(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ControllerURL(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ControllerURL(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ControllerURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNewRequiresController(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrDisabled) {
		t.Fatalf("New(nil) err = %v, want ErrDisabled", err)
	}
	if _, err := New(&option.ExperimentalOptions{ClashAPI: &option.ClashAPIOptions{}}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("New(empty controller) err = %v, want ErrDisabled", err)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(&option.ExperimentalOptions{ClashAPI: &option.ClashAPIOptions{
		ExternalController: server.URL,
		Secret:             "s3cret",
	}})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestGetSendsBearerAndDecodes(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer s3cret" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/rules" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rules":[{"type":"default","payload":"rule_set=geosite-google","proxy":"route(Google)"}]}`))
	})

	rules, err := client.Rules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Payload != "rule_set=geosite-google" || rules[0].Proxy != "route(Google)" {
		t.Fatalf("rules = %+v", rules)
	}
}

func TestGetClassifiesUnauthorized(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	if err := client.Get(context.Background(), "/version", nil); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestGetReportsOtherStatuses(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	err := client.Get(context.Background(), "/version", nil)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err = %v, want status 502 named", err)
	}
}

// The payload shape this package documents, as sing-box 1.12/1.13 actually
// emits it — chains reversed, rule as "<string> => <action>", start as RFC3339.
func TestConnectionsDecodesSingBoxShape(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
		  "downloadTotal": 100, "uploadTotal": 50, "memory": 7,
		  "connections": [{
		    "id": "5a1c", "upload": 10, "download": 90,
		    "start": "2026-09-03T10:00:00Z",
		    "chains": ["新加坡03 29db6db5 | sub_1788274931", "🤖 AI"],
		    "rule": "rule_set=sea-rulesets-ai => route(🤖 AI)",
		    "rulePayload": "",
		    "metadata": {"network":"tcp","type":"tun/tun-in","sourceIP":"192.168.9.20",
		      "destinationIP":"1.2.3.4","sourcePort":"5123","destinationPort":"443",
		      "host":"api.openai.com","dnsMode":"normal","processPath":""}
		  }]
		}`))
	})

	snapshot, err := client.Connections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.DownloadTotal != 100 || len(snapshot.Connections) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	conn := snapshot.Connections[0]
	if conn.Chains[len(conn.Chains)-1] != "🤖 AI" {
		t.Errorf("exit (last chain) = %q", conn.Chains[len(conn.Chains)-1])
	}
	if conn.Metadata.Type != "tun/tun-in" || conn.Metadata.Host != "api.openai.com" {
		t.Errorf("metadata = %+v", conn.Metadata)
	}
	if conn.Start.IsZero() {
		t.Error("start did not decode")
	}
}
