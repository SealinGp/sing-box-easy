package v1_13_0

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	configpkg "github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

func TestAddDNSRulePreservesNewerActionsAndUnknownConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "sing-box")
	script := []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo 'sing-box version 1.14.0'; exit 0; fi\nif [ \"$1\" = check ]; then exit 0; fi\nexit 1\n")
	if err := os.WriteFile(binaryPath, script, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	initial := []byte(`{
  "dns": {"servers": [{"type":"udp","tag":"router","server":"192.0.2.1"}], "rules": [{"action":"evaluate","server":"router","tag":"first"}], "future_dns": true},
  "future_section": {"enabled": true}
}`)
	if err := os.WriteFile(configPath, initial, 0600); err != nil {
		t.Fatal(err)
	}

	handler := testHandler(&Handler{configManager: configpkg.NewManager(configPath, binaryPath, "")})
	requestContext := app.NewContext(0)
	requestContext.Request.SetBody([]byte(`{"match_response":"first","response_rcode":"NOERROR","action":"respond","race":true}`))
	handler.AddDNSRule(context.Background(), requestContext)

	var response BasicResponse[map[string]any]
	if err := json.NewDecoder(bytes.NewReader(requestContext.Response.Body())).Decode(&response); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
	}
	if response.Code != CodeSuccess {
		t.Fatalf("response code = %d, want %d: %s", response.Code, CodeSuccess, response.Msg)
	}

	var saved map[string]json.RawMessage
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	var dns map[string]json.RawMessage
	if err := json.Unmarshal(saved["dns"], &dns); err != nil {
		t.Fatal(err)
	}
	var rules []map[string]any
	if err := json.Unmarshal(dns["rules"], &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0]["action"] != "evaluate" || rules[1]["action"] != "respond" {
		t.Fatalf("new DNS actions were not preserved: %#v", rules)
	}
	if dns["future_dns"] == nil || saved["future_section"] == nil {
		t.Fatalf("unknown fields were lost: %s", raw)
	}
}
