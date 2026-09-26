package diagnostics

// Building a rule-set loader out of a config this build may not fully parse.
//
// The loader needs exactly two things — `route.rule_set` and
// `experimental.cache_file` — out of a document that, on a host running a
// newer sing-box than the pinned library, does not decode as a whole. Asking
// for the whole subset and giving up on failure would disable rule-set
// evaluation precisely on the deployments where the binary fallback exists to
// help, so the fallback here narrows the projection to those two keys, which
// have been stable across every version this panel supports.
//
// Every failure degrades to an empty loader rather than an error: a probe that
// cannot read rule sets still reports them as undecidable, which is what it
// did before and is strictly better than refusing to run.

import (
	"context"
	"encoding/json"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	configpkg "github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

// ruleSetLoader builds a loader for the running config, with the binary
// fallback enabled when a sing-box executable is known.
//
// The caller owns the returned loader and must Close it: it may hold an open
// handle on sing-box's cache file, or a temporary copy of it.
func (h *Service) ruleSetLoader() *ruleset.Loader {
	options := h.ruleSetOptions()
	loader := ruleset.NewLoader(context.Background(), options)
	if h.serviceController != nil {
		loader.EnableCLIFallback(h.serviceController.BinaryPath())
	}
	return loader
}

// ruleSetOptions projects the config down to what the loader reads.
func (h *Service) ruleSetOptions() *option.Options {
	if cfg, err := h.configManager.GetConfigSubset("route", "experimental"); err == nil {
		return &cfg.Options
	}
	return h.narrowRuleSetOptions()
}

// narrowRuleSetOptions re-projects using only `route.rule_set` and
// `experimental.cache_file`, so one unparseable sibling key — a new inbound
// field, a new clash_api option — does not cost the whole feature.
func (h *Service) narrowRuleSetOptions() *option.Options {
	narrowed := map[string]any{}

	if route, ok, err := h.configManager.GetConfigSection("route"); err == nil && ok {
		if value := rawField(route, "rule_set"); value != nil {
			narrowed["route"] = map[string]json.RawMessage{"rule_set": value}
		}
	}
	if experimental, ok, err := h.configManager.GetConfigSection("experimental"); err == nil && ok {
		if value := rawField(experimental, "cache_file"); value != nil {
			narrowed["experimental"] = map[string]json.RawMessage{"cache_file": value}
		}
	}
	if len(narrowed) == 0 {
		return nil
	}

	encoded, err := json.Marshal(narrowed)
	if err != nil {
		return nil
	}
	var options option.Options
	if err := singjson.UnmarshalContext(
		configpkg.CreateContext(context.Background()), encoded, &options); err != nil {
		return nil
	}
	return &options
}

// rawField pulls one key out of a raw JSON object, or nil if absent.
func rawField(raw json.RawMessage, key string) json.RawMessage {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil
	}
	return object[key]
}
