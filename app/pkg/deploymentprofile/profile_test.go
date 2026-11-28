package deploymentprofile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExitProfile(t *testing.T) {
	result, err := Generate(context.Background(), Request{Role: "exit", Port: 23456})
	if err != nil {
		t.Fatal(err)
	}
	in := result.Config["inbounds"].([]any)[0].(map[string]any)
	key, err := base64.StdEncoding.DecodeString(in["password"].(string))
	if err != nil || len(key) != 16 {
		t.Fatal("invalid SS2022 key")
	}
	if result.Config["route"].(map[string]any)["default_domain_resolver"] != "google" {
		t.Fatal("missing resolver")
	}
	if in["listen_port"] != uint16(23456) {
		t.Fatal("port not preserved")
	}
	checkCore(t, result)
}
func TestClientProfile(t *testing.T) {
	uri := "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:example-password")) + "@192.0.2.10:443#node"
	result, err := Generate(context.Background(), Request{Role: "client", Subscription: uri, LANAddress: "192.168.31.2"})
	if err != nil {
		t.Fatal(err)
	}
	if result.NodeCount != 1 {
		t.Fatal("node import failed")
	}
	in := result.Config["inbounds"].([]any)
	if len(in) != 2 || in[0].(map[string]any)["tag"] != "tun-in" || in[1].(map[string]any)["listen"] != "192.168.31.2" {
		t.Fatal("missing TUN/LAN DNS")
	}
	route := result.Config["route"].(map[string]any)
	if route["default_domain_resolver"] != "bootstrap" || route["final"] != "proxy" {
		t.Fatal("DNS recursion or missing proxy")
	}
	checkCore(t, result)
	for _, address := range []string{"0.0.0.0", "::", "8.8.8.8"} {
		if _, err := Generate(context.Background(), Request{Role: "client", Subscription: uri, LANAddress: address}); err == nil {
			t.Fatalf("public DNS allowed: %s", address)
		}
	}
}
func TestProfileRejectsMissingInputs(t *testing.T) {
	for _, request := range []Request{{Role: "unknown"}, {Role: "client"}, {Role: "exit", Listen: "domain.test"}} {
		if _, err := Generate(context.Background(), request); err == nil {
			t.Fatal("expected rejection")
		}
	}
}
func checkCore(t *testing.T, result Result) {
	t.Helper()
	binary := os.Getenv("SBE_TEST_CORE")
	if binary == "" {
		return
	}
	raw, err := json.Marshal(result.Config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(binary, "check", "-c", path).CombinedOutput(); err != nil {
		t.Fatalf("core rejected profile: %v: %s", err, output)
	}
}

func TestClientProfileRejectsNonStandaloneNodes(t *testing.T) {
	for _, input := range []string{
		`{"type":"direct","tag":"direct"}`,
		`{"type":"selector","tag":"group","outbounds":["node"]}`,
		`{"type":"socks","tag":"node","server":"192.0.2.10","server_port":1080,"detour":"other"}`,
	} {
		if _, err := Generate(context.Background(), Request{Role: "client", Subscription: input}); err == nil {
			t.Fatalf("accepted non-standalone node %s", input)
		}
	}
}
