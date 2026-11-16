package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type ruleSetRefreshPlan struct{ path, id, tag string }

func planRuleSetRefresh(raw []byte, tag string, pid int, direct bool) (ruleSetRefreshPlan, error) {
	var cfg struct {
		Route struct {
			Sets []struct {
				Type        string          `json:"type"`
				Tag         json.RawMessage `json:"tag"`
				InitialPath string          `json:"initial_path"`
			} `json:"rule_set"`
		} `json:"route"`
		Experimental struct {
			Cache struct {
				Enabled bool   `json:"enabled"`
				Path    string `json:"path"`
				ID      string `json:"cache_id"`
			} `json:"cache_file"`
		} `json:"experimental"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ruleSetRefreshPlan{}, err
	}
	found := false
	for _, set := range cfg.Route.Sets {
		var tags []string
		var single string
		if json.Unmarshal(set.Tag, &single) == nil {
			tags = []string{single}
		} else {
			_ = json.Unmarshal(set.Tag, &tags)
		}
		for _, name := range tags {
			if name == tag {
				if set.Type != "remote" {
					return ruleSetRefreshPlan{}, fmt.Errorf("only remote rule sets can be refreshed")
				}
				// initial_path can postpone the initial download; do not report that as applied.
				if set.InitialPath != "" {
					return ruleSetRefreshPlan{}, fmt.Errorf("immediate refresh is not supported with initial_path")
				}
				found = true
			}
		}
	}
	if !found {
		return ruleSetRefreshPlan{}, fmt.Errorf("rule set %q not found", tag)
	}
	cache := cfg.Experimental.Cache
	if !cache.Enabled {
		return ruleSetRefreshPlan{}, fmt.Errorf("immediate refresh requires rule-set caching to be enabled")
	}
	if cache.Path == "" {
		cache.Path = "cache.db"
	}
	if !filepath.IsAbs(cache.Path) {
		cwd, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "cwd"))
		if err != nil && direct {
			cwd, err = os.Getwd()
		}
		if err != nil {
			return ruleSetRefreshPlan{}, fmt.Errorf("cannot determine the running core's cache directory: %w", err)
		}
		cache.Path = filepath.Join(cwd, cache.Path)
	}
	return ruleSetRefreshPlan{path: cache.Path, id: cache.ID, tag: tag}, nil
}
