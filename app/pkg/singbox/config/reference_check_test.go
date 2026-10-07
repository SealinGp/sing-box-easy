package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
)

func problemsByWhere(problems []ReferenceProblem) map[string]ReferenceProblem {
	indexed := make(map[string]ReferenceProblem, len(problems))
	for _, problem := range problems {
		indexed[problem.Where] = problem
	}
	return indexed
}

func TestCheckReferencesFindsEveryPlaceTheCoreResolvesAtStart(t *testing.T) {
	problems := problemsByWhere(CheckReferences([]byte(`{
	  "outbounds": [
	    {"type": "direct", "tag": "direct"},
	    {"type": "urltest", "tag": "JP", "outbounds": ["direct"]},
	    {"type": "selector", "tag": "hollow", "outbounds": []}
	  ],
	  "route": {
	    "final": "Other Nodes",
	    "rules": [
	      {"action": "sniff"},
	      {"rule_set": ["geosite-cn"], "outbound": "direct"},
	      {"type": "logical", "mode": "or", "outbound": "JP",
	       "rules": [{"domain": ["a.com"], "outbound": "nested-missing"}]},
	      {"domain": ["b.com"], "outbound": "hollow"}
	    ],
	    "rule_set": [{"tag": "geosite-cn", "download_detour": "gone"}, {"tag": "local"}]
	  },
	  "dns": {"servers": [{"tag": "remote", "detour": "also-gone"}, {"tag": "local", "detour": "direct"}]}
	}`)))

	want := map[string]ReferenceProblem{
		"route.final":                       {Kind: ReferenceMissing, Field: FieldRouteFinal, Tag: "Other Nodes"},
		"route.rules[2].rules[0]":           {Kind: ReferenceMissing, Field: FieldRouteRules, Tag: "nested-missing"},
		"route.rules[3]":                    {Kind: ReferenceEmptyGroup, Field: FieldRouteRules, Tag: "hollow"},
		"route.rule_set[0].download_detour": {Kind: ReferenceMissing, Field: FieldRouteRuleSet, Tag: "gone"},
		"dns.servers[0].detour":             {Kind: ReferenceMissing, Field: FieldDNSServers, Tag: "also-gone"},
	}
	if len(problems) != len(want) {
		t.Fatalf("problems = %+v, want %d", problems, len(want))
	}
	for where, expected := range want {
		got, found := problems[where]
		if !found || got.Kind != expected.Kind || got.Field != expected.Field || got.Tag != expected.Tag {
			t.Errorf("%s = %+v, want %+v", where, got, expected)
		}
	}
}

// A wireguard/tailscale endpoint lives in its own section and is a legal route
// target since sing-box 1.11. Reporting it as missing would refuse to start a
// config that works.
func TestCheckReferencesAcceptsEndpointsAsTargets(t *testing.T) {
	problems := CheckReferences([]byte(`{
	  "outbounds": [{"type": "direct", "tag": "direct"}],
	  "endpoints": [{"type": "wireguard", "tag": "wg-home"}],
	  "route": {"final": "wg-home", "rules": [{"ip_cidr": ["10.0.0.0/8"], "outbound": "wg-home"}]}
	}`))
	if len(problems) != 0 {
		t.Fatalf("problems = %+v, want none", problems)
	}
}

func TestCheckReferencesHasNothingToSayAboutAHealthyOrMinimalConfig(t *testing.T) {
	for name, document := range map[string]string{
		"no route":       `{"outbounds": [{"type": "direct", "tag": "direct"}]}`,
		"no final":       `{"outbounds": [{"type": "direct", "tag": "direct"}], "route": {"rules": []}}`,
		"empty final":    `{"outbounds": [{"type": "direct", "tag": "direct"}], "route": {"final": ""}}`,
		"empty document": `{}`,
		"group with members": `{"outbounds": [{"type": "direct", "tag": "d"},
		  {"type": "selector", "tag": "g", "outbounds": ["d"]}], "route": {"final": "g"}}`,
		// An unused empty group is the core's business, not a dangling reference.
		"unreferenced empty group": `{"outbounds": [{"type": "selector", "tag": "g", "outbounds": []}]}`,
	} {
		if problems := CheckReferences([]byte(document)); len(problems) != 0 {
			t.Errorf("%s: problems = %+v, want none", name, problems)
		}
	}
}

// A guard that cannot read a section must not invent a problem in it.
func TestCheckReferencesSkipsShapesItDoesNotUnderstand(t *testing.T) {
	for name, document := range map[string]string{
		"not json":             `{not json`,
		"outbounds unreadable": `{"outbounds": "nope", "route": {"final": "x"}}`,
		"odd route":            `{"outbounds": [], "route": {"final": 7, "rules": "not-a-list", "rule_set": [3]}}`,
		"odd dns":              `{"outbounds": [], "dns": ["unexpected"]}`,
	} {
		if problems := CheckReferences([]byte(document)); len(problems) != 0 {
			t.Errorf("%s: problems = %+v, want none", name, problems)
		}
	}
}

func TestReferenceErrorNamesEachSettingAndIsRecognisable(t *testing.T) {
	err := error(&ValidationError{Stage: ValidationStagePanelGuard, Err: &ReferenceError{Problems: []ReferenceProblem{
		{Kind: ReferenceMissing, Field: FieldRouteFinal, Where: "route.final", Tag: "Other Nodes"},
		{Kind: ReferenceEmptyGroup, Field: FieldRouteRules, Where: "route.rules[3]", Tag: "hollow"},
	}}})
	if !errors.Is(err, ErrOutboundReference) {
		t.Fatal("a reference error must match ErrOutboundReference through ValidationError")
	}
	var reference *ReferenceError
	if !errors.As(err, &reference) || len(reference.Problems) != 2 {
		t.Fatalf("errors.As lost the problems: %#v", err)
	}
	for _, want := range []string{"route.final", `"Other Nodes"`, "route.rules[3]", "no members", "sing-box check"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q does not mention %s", err.Error(), want)
		}
	}
}

func newReferenceManager(t *testing.T, document string) (*Manager, string) {
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

func dropOutbound(tag string) func(*SingBoxConfig) error {
	return func(cfg *SingBoxConfig) error {
		kept := cfg.Outbounds[:0:0]
		for _, outbound := range cfg.Outbounds {
			if outbound.Tag != tag {
				kept = append(kept, outbound)
			}
		}
		cfg.Outbounds = kept
		return nil
	}
}

// Issue #14: the fallback group is dropped because it matches no nodes while
// route.final still names it. The fake core here accepts everything, exactly as
// the real `sing-box check` accepts this config.
func TestSaveRefusesToStrandRouteFinal(t *testing.T) {
	manager, configPath := newReferenceManager(t, fallbackFinalConfig)

	err := manager.UpdateOutboundsConfig(context.Background(), dropOutbound("Other Nodes"))

	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Stage != ValidationStagePanelGuard {
		t.Fatalf("err = %#v, want a panel-guard ValidationError", err)
	}
	var reference *ReferenceError
	if !errors.As(err, &reference) || len(reference.Problems) != 1 {
		t.Fatalf("err = %v, want one reference problem", err)
	}
	if got := reference.Problems[0]; got.Field != FieldRouteFinal || got.Tag != "Other Nodes" || got.Kind != ReferenceMissing {
		t.Errorf("problem = %+v", got)
	}
	after, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != fallbackFinalConfig {
		t.Fatalf("config was modified by a refused save:\n%s", after)
	}
}

// The same guard on a different door: setting route.final to a tag that does
// not exist goes through a section edit, not an outbound edit.
func TestSaveRefusesARouteEditThatNamesAMissingOutbound(t *testing.T) {
	manager, _ := newReferenceManager(t, fallbackFinalConfig)
	err := manager.SaveDocument(context.Background(), ConfigDocument{Raw: []byte(strings.Replace(fallbackFinalConfig, `"final": "Other Nodes"`, `"final": "typo"`, 1))})
	if !errors.Is(err, ErrOutboundReference) {
		t.Fatalf("err = %v, want ErrOutboundReference", err)
	}
}

func TestSaveAllowsRemovingAnUnreferencedOutbound(t *testing.T) {
	manager, configPath := newReferenceManager(t, strings.Replace(fallbackFinalConfig, `"final": "Other Nodes"`, `"final": "JP"`, 1))
	if err := manager.UpdateOutboundsConfig(context.Background(), dropOutbound("Other Nodes")); err != nil {
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

const twiceBrokenConfig = `{
  "outbounds": [{"type": "direct", "tag": "direct"}],
  "route": {"final": "gone", "rules": [{"domain": ["a.com"], "outbound": "also-gone"}]}
}`

// A config with two broken references must be repairable one edit at a time:
// fixing route.final cannot be refused because a rule is still broken.
func TestSaveAllowsRepairingOneProblemWhileAnotherRemains(t *testing.T) {
	manager, _ := newReferenceManager(t, twiceBrokenConfig)
	half := strings.Replace(twiceBrokenConfig, `"final": "gone"`, `"final": "direct"`, 1)
	if err := manager.SaveDocument(context.Background(), ConfigDocument{Raw: []byte(half)}); err != nil {
		t.Fatalf("repairing route.final was refused: %v", err)
	}
	// …and an unrelated edit to an already-broken config is not blocked either.
	if err := manager.UpdateOutboundsConfig(context.Background(), func(cfg *SingBoxConfig) error {
		cfg.Outbounds = append(cfg.Outbounds, Outbound{Type: "block", Tag: "blocked"})
		return nil
	}); err != nil {
		t.Fatalf("unrelated edit refused: %v", err)
	}
}

// Moving a broken rule to another index is not a NEW problem.
func TestSaveDoesNotTreatAReorderedBrokenRuleAsNew(t *testing.T) {
	manager, _ := newReferenceManager(t, twiceBrokenConfig)
	reordered := strings.Replace(twiceBrokenConfig, `"rules": [`, `"rules": [{"action": "sniff"}, `, 1)
	if err := manager.SaveDocument(context.Background(), ConfigDocument{Raw: []byte(reordered)}); err != nil {
		t.Fatalf("reorder refused: %v", err)
	}
}

// Before start nothing is grandfathered: the core is about to fail on it.
func TestValidateCurrentConfigRefusesAnExistingProblem(t *testing.T) {
	manager, _ := newReferenceManager(t, twiceBrokenConfig)
	err := manager.ValidateCurrentConfig(context.Background())
	var reference *ReferenceError
	if !errors.As(err, &reference) || len(reference.Problems) != 2 {
		t.Fatalf("err = %v, want both problems reported", err)
	}

	healthy, _ := newReferenceManager(t, strings.Replace(fallbackFinalConfig, `"final": "Other Nodes"`, `"final": "JP"`, 1))
	if err := healthy.ValidateCurrentConfig(context.Background()); err != nil {
		t.Fatalf("a healthy config was refused: %v", err)
	}
}

// An explicit "validate this document" reports every problem in it.
func TestValidateRawConfigReportsEveryProblem(t *testing.T) {
	manager, _ := newReferenceManager(t, fallbackFinalConfig)
	err := manager.ValidateRawConfig(context.Background(), []byte(twiceBrokenConfig))
	var reference *ReferenceError
	if !errors.As(err, &reference) || len(reference.Problems) != 2 {
		t.Fatalf("err = %v, want both problems reported", err)
	}
}
