package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	configpkg "github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
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
	handler := testHandler(&Handler{configManager: manager})
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

	handler := testHandler(&Handler{configManager: configpkg.NewManager(configPath, "sing-box", "")})
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

// emptyVersionStore is a history with no versions, behaving like
// history.StoreXORM does for an id it does not have.
type emptyVersionStore struct{}

func (emptyVersionStore) Save([]byte) (int64, error)             { return 1, nil }
func (emptyVersionStore) List() ([]configpkg.VersionInfo, error) { return nil, nil }
func (emptyVersionStore) Prune(int) error                        { return nil }
func (emptyVersionStore) DeleteBatch(ids []int64) (int64, error) { return 0, nil }
func (emptyVersionStore) Get(id int64) ([]byte, error) {
	return nil, fmt.Errorf("config version %d: %w", id, configpkg.ErrVersionNotFound)
}
func (emptyVersionStore) Delete(id int64) error {
	return fmt.Errorf("config version %d: %w", id, configpkg.ErrVersionNotFound)
}

// The version endpoints moved their id and not-found handling into
// config.Service. These are the response codes they returned before the move.
func TestConfigVersionEndpointsKeepTheirResponseCodes(t *testing.T) {
	manager := configpkg.NewManager(filepath.Join(t.TempDir(), "config.json"), "sing-box", "")
	manager.SetVersionStore(emptyVersionStore{})
	handler := testHandler(&Handler{configManager: manager})

	cases := []struct {
		name string
		call func(context.Context, *app.RequestContext)
		id   string
		want Code
	}{
		{"get missing version", handler.GetConfigVersion, "42", CodeNotFound},
		{"get unparsable id", handler.GetConfigVersion, "abc", CodeBadRequest},
		{"get non-positive id", handler.GetConfigVersion, "0", CodeBadRequest},
		{"delete missing version", handler.DeleteConfigVersion, "42", CodeNotFound},
		{"delete non-positive id", handler.DeleteConfigVersion, "-3", CodeBadRequest},
		{"restore missing version", handler.RollbackToConfigVersion, "42", CodeInternalError},
		{"restore unparsable id", handler.RollbackToConfigVersion, "x", CodeBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requestContext := app.NewContext(0)
			requestContext.Params = append(requestContext.Params, param.Param{Key: "id", Value: tc.id})
			tc.call(context.Background(), requestContext)

			var response BasicResponse[any]
			if err := json.Unmarshal(requestContext.Response.Body(), &response); err != nil {
				t.Fatalf("response is not valid JSON: %v (%s)", err, requestContext.Response.Body())
			}
			if response.Code != tc.want {
				t.Errorf("code = %d (%s), want %d", response.Code, response.Msg, tc.want)
			}
		})
	}
}
