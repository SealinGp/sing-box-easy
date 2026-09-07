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

func TestValidateConfigDelegatesNewDNSActionsToInstalledCore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "sing-box")
	script := []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo 'sing-box version 1.14.0'; exit 0; fi\nif [ \"$1\" = check ]; then exit 0; fi\nexit 1\n")
	if err := os.WriteFile(binaryPath, script, 0700); err != nil {
		t.Fatal(err)
	}

	manager := configpkg.NewManager(filepath.Join(dir, "config.json"), binaryPath, "")
	handler := &Handler{configManager: manager}
	requestContext := app.NewContext(0)
	requestContext.Request.SetBody([]byte(`{"dns":{"rules":[{"action":"evaluate","server":"dns_router"}]}}`))

	handler.ValidateConfig(context.Background(), requestContext)

	var response BasicResponse[map[string]any]
	if err := json.NewDecoder(bytes.NewReader(requestContext.Response.Body())).Decode(&response); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
	}
	if response.Code != CodeSuccess {
		t.Fatalf("response code = %d, want %d: %s", response.Code, CodeSuccess, response.Msg)
	}
	if response.Data["valid"] != true {
		t.Fatalf("response data = %#v, want valid=true", response.Data)
	}
}

func TestGetConfigReturnsUnknownFieldsInsideResponseEnvelope(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","future":true}]},"future_section":{"value":7}}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	handler := &Handler{configManager: configpkg.NewManager(configPath, "sing-box", "")}
	requestContext := app.NewContext(0)
	handler.GetConfig(context.Background(), requestContext)

	var response struct {
		Code Code                       `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
		Msg  string                     `json:"msg"`
	}
	if err := json.Unmarshal(requestContext.Response.Body(), &response); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
	}
	if response.Code != CodeSuccess {
		t.Fatalf("response code = %d, want %d: %s", response.Code, CodeSuccess, response.Msg)
	}
	if _, ok := response.Data["future_section"]; !ok {
		t.Fatalf("unknown top-level field was lost: %s", requestContext.Response.Body())
	}
}
