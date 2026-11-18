package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
)

type emptyGroupCore struct{ calls int }

func (c *emptyGroupCore) Version(context.Context) (core.CoreVersion, error) {
	return core.CoreVersion{Raw: "1.14.0", Major: 1, Minor: 14}, nil
}

func TestSubscriptionEmptyGroupWithInstalledCore(t *testing.T) {
	binary, err := exec.LookPath("sing-box")
	if err != nil {
		t.Skip("sing-box executable is not installed")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{"outbounds":[
		{"type":"direct","tag":"direct"},
		{"type":"selector","tag":"其他节点","outbounds":["old | sub_test"]},
		{"type":"selector","tag":"流媒体","outbounds":["其他节点","direct"]},
		{"type":"direct","tag":"old | sub_test"}
	]}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	manager := config.NewManager(path, binary, "")
	service := &Service{configManager: manager, nodeRules: &fakeRules{}}
	if err := service.applyChanges(map[string]struct{}{"old | sub_test": {}}, []config.Outbound{{Type: "direct", Tag: "new | sub_test"}}, nil, "sub_test"); err != nil {
		t.Fatal(err)
	}
	if err := manager.ValidateCurrentConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (c *emptyGroupCore) Validate(_ context.Context, path string) error {
	c.calls++
	return nil
}

// Match the production failure: a manual selector contains only a replaced
// subscription node, while another manual selector references that selector.
func TestSubscriptionUpdateRemovesEmptyGroupsSafely(t *testing.T) {
	for _, tc := range []struct {
		name, route, first, wantErr string
	}{
		{name: "prunes manual child", first: "direct"},
		{name: "protects explicit route", route: `,"route":{"final":"其他节点"}`, first: "direct", wantErr: "其他节点"},
		{name: "protects route rule", route: `,"route":{"rules":[{"outbound":"其他节点"}]}`, first: "direct", wantErr: "其他节点"},
		{name: "protects implicit route", first: "其他节点", wantErr: "implicit default"},
		{name: "allows explicit surviving route", route: `,"route":{"final":"direct"}`, first: "其他节点"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groups := []json.RawMessage{
				json.RawMessage(`{"type":"direct","tag":"direct"}`),
				json.RawMessage(`{"type":"selector","tag":"其他节点","outbounds":["old | sub_test"]}`),
				json.RawMessage(`{"type":"selector","tag":"流媒体","outbounds":["其他节点","direct"],"default":"其他节点"}`),
				json.RawMessage(`{"type":"direct","tag":"old | sub_test"}`),
			}
			if tc.first != "direct" {
				groups[0], groups[1] = groups[1], groups[0]
			}
			encoded, _ := json.Marshal(groups)
			raw := append([]byte(`{"outbounds":`), encoded...)
			raw = append(raw, []byte(tc.route+"}")...)
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			core := &emptyGroupCore{}
			manager := config.NewManagerWithCore(path, core)
			service := &Service{configManager: manager, nodeRules: &fakeRules{}}
			err := service.applyChanges(map[string]struct{}{"old | sub_test": {}}, []config.Outbound{{Type: "direct", Tag: "new | sub_test"}}, nil, "sub_test")
			updated, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %v, want error containing %q", err, tc.wantErr)
				}
				if !bytes.Equal(raw, updated) || core.calls != 0 {
					t.Fatal("unsafe update reached core validation or changed config")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := manager.GetOutboundsConfig()
			if err != nil {
				t.Fatal(err)
			}
			if hasTag(cfg.Outbounds, "其他节点") || hasTag(cfg.Outbounds, "old | sub_test") || !hasTag(cfg.Outbounds, "new | sub_test") {
				t.Fatal("empty group or subscription replacement was not reconciled")
			}
			if got := groupMembers(t, cfg.Outbounds, "流媒体"); !slices.Equal(got, []string{"direct"}) {
				t.Fatalf("parent members = %v", got)
			}
			if core.calls != 1 {
				t.Fatalf("core validation calls = %d", core.calls)
			}
		})
	}
}
