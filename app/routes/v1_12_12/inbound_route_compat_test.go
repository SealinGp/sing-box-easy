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

func TestInboundAndRouteReadsDoNotDecodeNewerDNSRules(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{
  "dns":{"rules":[{"action":"evaluate","server":"router","future":true}]},
  "inbounds":[{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":7893}],
  "route":{"rules":[{"domain_suffix":"example.com","action":"route","outbound":"direct"}],"rule_set":[{"type":"local","tag":"local-set","format":"source","path":"rules.json"}],"final":"direct"}
}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	handler := &Handler{configManager: configpkg.NewManager(configPath, "sing-box", "")}

	tests := []struct {
		name string
		call func(context.Context, *app.RequestContext)
	}{
		{"inbounds", handler.GetInbounds},
		{"route rules", handler.GetRouteRules},
		{"rule sets", handler.GetRuleSets},
		{"route final", handler.GetRouteFinal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestContext := app.NewContext(0)
			tt.call(context.Background(), requestContext)
			var response BasicResponse[json.RawMessage]
			if err := json.Unmarshal(requestContext.Response.Body(), &response); err != nil {
				t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
			}
			if response.Code != CodeSuccess {
				t.Fatalf("unexpected response: %s", requestContext.Response.Body())
			}
		})
	}
}

func TestRouteMutationPreservesNewerDNSRules(t *testing.T) {
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
	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","future":true}]},"route":{"final":"old","future_route":7}}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	handler := &Handler{configManager: configpkg.NewManager(configPath, binaryPath, "")}
	requestContext := app.NewContext(0)
	requestContext.Request.SetBody([]byte(`{"final":"new"}`))
	handler.UpdateRouteFinal(context.Background(), requestContext)

	var response BasicResponse[json.RawMessage]
	if err := json.Unmarshal(requestContext.Response.Body(), &response); err != nil || response.Code != CodeSuccess {
		t.Fatalf("unexpected response: %s (%v)", requestContext.Response.Body(), err)
	}
	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(saved, []byte(`"action": "evaluate"`)) || !bytes.Contains(saved, []byte(`"future_route": 7`)) {
		t.Fatalf("unrelated fields were lost: %s", saved)
	}
}

func TestRawRuleSetCascadeSupportsNewerDNSActions(t *testing.T) {
	rules := []json.RawMessage{
		json.RawMessage(`{"action":"evaluate","server":"router","tag":"result","rule_set":["remove","keep"]}`),
		json.RawMessage(`{"action":"respond","match_response":"result","rule_set":"remove"}`),
		json.RawMessage(`{"action":"route","server":"router","rule_set":"remove"}`),
	}
	updated, err := scrubRawRules(rules, "remove", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 2 || !bytes.Contains(updated[0], []byte(`"action":"evaluate"`)) || !bytes.Contains(updated[0], []byte(`"keep"`)) || !bytes.Contains(updated[1], []byte(`"match_response":"result"`)) {
		t.Fatalf("unexpected cascade result: %s", updated)
	}
}
