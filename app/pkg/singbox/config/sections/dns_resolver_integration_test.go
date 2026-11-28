package sections

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

func TestAddSecondDNSWithInstalledCore(t *testing.T) {
	binary := os.Getenv("SBE_TEST_CORE")
	if binary == "" {
		t.Skip("set SBE_TEST_CORE for released-core integration")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"dns":{"servers":[{"type":"tls","tag":"google","server":"8.8.8.8"}]},"outbounds":[{"type":"direct"}],"route":{"rules":[{"port":53,"action":"hijack-dns"}]}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	service := New(config.NewManager(path, binary, ""))
	if _, err := service.AddDNSServer(context.Background(), []byte(`{"tag":"us-1","type":"udp","server":"7.7.7.7"}`)); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]json.RawMessage
	if err = json.Unmarshal(saved, &result); err != nil {
		t.Fatal(err)
	}
	if rawStringField(result["route"], "default_domain_resolver") != "google" {
		t.Fatal("implicit resolver not preserved")
	}
}

func TestSecondDNSValidationFailureKeepsResolverAndServers(t *testing.T) {
	binary := os.Getenv("SBE_TEST_CORE")
	if binary == "" {
		t.Skip("set SBE_TEST_CORE")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"dns":{"servers":[{"type":"tls","tag":"google","server":"8.8.8.8"}]},"outbounds":[{"type":"direct"}]}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	service := New(config.NewManager(path, binary, ""))
	if _, err := service.AddDNSServer(context.Background(), []byte(`{"tag":"bad","type":"udp","server":"7.7.7.7","invalid_option":true}`)); err == nil {
		t.Fatal("invalid config accepted")
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(original) {
		t.Fatal("failed validation changed original config")
	}
}
