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

func TestGetCacheFileDoesNotDecodeNewerDNSRules(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","server":"router"}]},"experimental":{"cache_file":{"enabled":true,"path":"/tmp/cache.db"}}}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	handler := testHandler(&Handler{configManager: configpkg.NewManager(configPath, "sing-box", "")})
	requestContext := app.NewContext(0)
	handler.GetCacheFile(context.Background(), requestContext)

	var response BasicResponse[map[string]any]
	if err := json.NewDecoder(bytes.NewReader(requestContext.Response.Body())).Decode(&response); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
	}
	if response.Code != CodeSuccess || response.Data["enabled"] != true {
		t.Fatalf("unexpected response: %s", requestContext.Response.Body())
	}
}

func TestUpdateCacheFilePreservesNewerDNSAndExperimentalFields(t *testing.T) {
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
	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","future":true}]},"experimental":{"cache_file":{"enabled":false},"future_feature":{"value":7}}}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	handler := testHandler(&Handler{configManager: configpkg.NewManager(configPath, binaryPath, "")})
	requestContext := app.NewContext(0)
	requestContext.Request.SetBody([]byte(`{"enabled":true,"path":"/tmp/new-cache.db","rdrc_timeout":""}`))
	handler.UpdateCacheFile(context.Background(), requestContext)

	var response BasicResponse[map[string]any]
	if err := json.Unmarshal(requestContext.Response.Body(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != CodeSuccess {
		t.Fatalf("unexpected response: %s", requestContext.Response.Body())
	}

	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(saved, &document); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(document["dns"], []byte(`"action": "evaluate"`)) ||
		!bytes.Contains(document["experimental"], []byte(`"future_feature"`)) {
		t.Fatalf("unrelated newer fields were lost: %s", saved)
	}
	if bytes.Contains(document["experimental"], []byte(`rdrc_timeout`)) {
		t.Fatalf("blank duration was not omitted: %s", document["experimental"])
	}
}
