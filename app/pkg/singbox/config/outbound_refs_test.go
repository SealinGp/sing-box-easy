package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
)

func sectionsOf(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var sections map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &sections); err != nil {
		t.Fatal(err)
	}
	return sections
}

func tags(names ...string) []Outbound {
	outbounds := make([]Outbound, 0, len(names))
	for _, name := range names {
		outbounds = append(outbounds, Outbound{Type: "direct", Tag: name})
	}
	return outbounds
}

func TestOutboundReferencesFindsEveryPlaceTheCoreResolvesAtStart(t *testing.T) {
	references := outboundReferences(sectionsOf(t, `{
	  "route": {
	    "final": "Other Nodes",
	    "rules": [
	      {"action": "sniff"},
	      {"rule_set": ["geosite-cn"], "outbound": "direct"},
	      {"type": "logical", "mode": "or", "outbound": "JP",
	       "rules": [{"domain": ["a.com"], "outbound": "nested"}]}
	    ],
	    "rule_set": [{"tag": "geosite-cn", "download_detour": "US"}, {"tag": "local"}]
	  },
	  "dns": {"servers": [{"tag": "remote", "detour": "KR"}, {"tag": "local"}]}
	}`))

	got := make(map[string]string, len(references))
	for _, reference := range references {
		got[reference.Where] = reference.Tag
	}
	want := map[string]string{
		"route.final":                       "Other Nodes",
		"route.rules[1]":                    "direct",
		"route.rules[2]":                    "JP",
		"route.rules[2].rules[0]":           "nested",
		"route.rule_set[0].download_detour": "US",
		"dns.servers[0].detour":             "KR",
	}
	if len(got) != len(want) {
		t.Fatalf("references = %v, want %v", got, want)
	}
	for where, tag := range want {
		if got[where] != tag {
			t.Errorf("%s = %q, want %q", where, got[where], tag)
		}
	}
}

// A section this build cannot read must not stop an unrelated outbound edit.
func TestOutboundReferencesSkipsShapesItDoesNotUnderstand(t *testing.T) {
	references := outboundReferences(sectionsOf(t, `{
	  "route": {"final": 7, "rules": "not-a-list", "rule_set": [3, {"download_detour": ""}]},
	  "dns": ["unexpected"]
	}`))
	if len(references) != 0 {
		t.Fatalf("references = %v, want none", references)
	}
	if got := outboundReferences(map[string]json.RawMessage{}); len(got) != 0 {
		t.Fatalf("no sections: %v", got)
	}
}

func TestDanglingAfterReportsOnlyWhatThisEditBroke(t *testing.T) {
	sections := sectionsOf(t, `{"route": {"final": "Other Nodes",
	  "rules": [{"outbound": "JP"}, {"outbound": "never-existed"}]}}`)

	// The edit drops "Other Nodes"; "JP" survives; "never-existed" was already
	// dangling before this edit and is not this edit's doing.
	dangling := danglingAfter(tags("Other Nodes", "JP", "direct"), tags("JP", "direct"), sections)
	if len(dangling) != 1 || dangling[0].Where != "route.final" || dangling[0].Tag != "Other Nodes" {
		t.Fatalf("dangling = %+v", dangling)
	}
	if got := danglingAfter(tags("Other Nodes", "JP"), tags("Other Nodes", "JP", "new"), sections); len(got) != 0 {
		t.Fatalf("an edit that removes nothing must report nothing: %+v", got)
	}
}

func newRefsManager(t *testing.T, document string) (*Manager, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(document), 0600); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(configPath, "sing-box", "")
	manager.core = &fakeCoreAdapter{version: core.CoreVersion{Major: 1, Minor: 12, Patch: 25, Raw: "1.12.25"}}
	return manager, configPath
}

const fallbackFinalConfig = `{
  "outbounds": [
    {"type": "direct", "tag": "direct"},
    {"type": "urltest", "tag": "Other Nodes", "outbounds": ["direct"]},
    {"type": "urltest", "tag": "JP", "outbounds": ["direct"]}
  ],
  "route": {"final": "Other Nodes", "rules": [{"domain": ["a.com"], "outbound": "JP"}]}
}`

// Issue #14: the fallback group is dropped because it matches no nodes while
// route.final still names it. `sing-box check` passes that config and the core
// then dies at start with "default outbound not found: Other Nodes".
func TestUpdateOutboundsConfigRefusesToStrandRouteFinal(t *testing.T) {
	manager, configPath := newRefsManager(t, fallbackFinalConfig)

	err := manager.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
		kept := cfg.Outbounds[:0:0]
		for _, outbound := range cfg.Outbounds {
			if outbound.Tag != "Other Nodes" {
				kept = append(kept, outbound)
			}
		}
		cfg.Outbounds = kept
		return nil
	})

	if !errors.Is(err, ErrDanglingOutboundReference) {
		t.Fatalf("err = %v, want ErrDanglingOutboundReference", err)
	}
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Stage != ValidationStagePanelGuard {
		t.Fatalf("err = %#v, want a panel-guard ValidationError", err)
	}
	for _, want := range []string{`"Other Nodes"`, "route.final"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q does not name %s", err.Error(), want)
		}
	}

	// The working config must be exactly as it was.
	after, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != fallbackFinalConfig {
		t.Fatalf("config was modified by a refused edit:\n%s", after)
	}
}

func TestUpdateOutboundsConfigAllowsRemovingAnUnreferencedOutbound(t *testing.T) {
	manager, configPath := newRefsManager(t, strings.Replace(fallbackFinalConfig, `"final": "Other Nodes"`, `"final": "JP"`, 1))

	if err := manager.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
		kept := cfg.Outbounds[:0:0]
		for _, outbound := range cfg.Outbounds {
			if outbound.Tag != "Other Nodes" {
				kept = append(kept, outbound)
			}
		}
		cfg.Outbounds = kept
		return nil
	}); err != nil {
		t.Fatalf("UpdateOutboundsConfig() error = %v", err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "Other Nodes") {
		t.Fatalf("the outbound was not removed:\n%s", after)
	}
}

// A reference that was ALREADY broken must not block later, unrelated edits —
// otherwise the operator could not fix anything from the panel.
func TestUpdateOutboundsConfigIgnoresReferencesThatWereAlreadyBroken(t *testing.T) {
	manager, _ := newRefsManager(t, `{
  "outbounds": [{"type": "direct", "tag": "direct"}],
  "route": {"final": "gone-long-ago"}
}`)
	if err := manager.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
		cfg.Outbounds = append(cfg.Outbounds, Outbound{Type: "block", Tag: "blocked"})
		return nil
	}); err != nil {
		t.Fatalf("UpdateOutboundsConfig() error = %v", err)
	}
}
