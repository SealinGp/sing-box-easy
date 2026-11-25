package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func localRuleManager(t *testing.T) (*Manager, *fakeCoreAdapter) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(`{"route":{"rule_set":[]},"custom":{"keep":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	m := NewManager(p, "sing-box", "")
	f := &fakeCoreAdapter{}
	m.core = f
	return m, f
}
func createLocalRule(t *testing.T, m *Manager, tag string) LocalRuleFile {
	t.Helper()
	initial, err := m.ReadLocalRuleFile("")
	if err != nil {
		t.Fatal(err)
	}
	result, err := m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: tag, Revision: initial.Revision, Content: json.RawMessage(`{"version":3,"rules":[{"domain_suffix":["example.com"]}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestLocalRuleCreateEditBackupConflict(t *testing.T) {
	m, f := localRuleManager(t)
	first := createLocalRule(t, m, "我的代理")
	if filepath.Base(first.Path) != "我的代理.json" {
		t.Fatal(first.Path)
	}
	config, _ := os.ReadFile(m.configPath)
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(config, &doc); err != nil || string(doc["custom"]) == "" {
		t.Fatal(string(config), err)
	}
	changed := json.RawMessage(`{"version":3,"rules":[{"domain":["other.example"],"invert":true}]}`)
	result, err := m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: first.Tag, Existing: true, Revision: first.Revision, Content: changed})
	if err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(first.Path + ".bak")
	if err != nil || string(backup) != string(first.Content) {
		t.Fatal(string(backup), err)
	}
	after, _ := os.ReadFile(m.configPath)
	if string(after) != string(config) {
		t.Fatal("editing content changed config")
	}
	if result.Revision == first.Revision || f.validateCalls < 3 {
		t.Fatal("missing validation/revision")
	}
	_, err = m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: first.Tag, Existing: true, Revision: first.Revision, Content: changed})
	if err == nil {
		t.Fatal("stale revision accepted")
	}
	if err := os.WriteFile(first.Path, first.Content, 0644); err != nil {
		t.Fatal(err)
	}
	_, err = m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: first.Tag, Existing: true, Revision: result.Revision, Content: changed})
	if err == nil {
		t.Fatal("external modification accepted")
	}
}
func TestLocalRuleValidationFailurePreservesFile(t *testing.T) {
	m, f := localRuleManager(t)
	first := createLocalRule(t, m, "proxy")
	f.validateErr = errors.New("core rejected")
	_, err := m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: first.Tag, Existing: true, Revision: first.Revision, Content: json.RawMessage(`{"version":3,"rules":[{"domain":["invalid"]}]}`)})
	if err == nil {
		t.Fatal("accepted core failure")
	}
	actual, _ := os.ReadFile(first.Path)
	if string(actual) != string(first.Content) {
		t.Fatal("active file changed")
	}
}
func TestLocalRuleNeverOverwritesNameCollision(t *testing.T) {
	m, _ := localRuleManager(t)
	first := createLocalRule(t, m, "proxy/a")
	initial, _ := m.ReadLocalRuleFile("")
	_, err := m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: "proxy:a", Revision: initial.Revision, Content: first.Content})
	if err == nil {
		t.Fatal("colliding filename overwritten")
	}
	actual, _ := os.ReadFile(first.Path)
	if string(actual) != string(first.Content) {
		t.Fatal("file changed")
	}
}
func TestLocalRuleEmptyRejected(t *testing.T) {
	for _, s := range []string{`{}`, `{"version":3,"rules":[]}`, `{"version":3,"rules":[{}]}`, `{"version":3,"rules":[{"domain":[],"invert":true}]}`} {
		if validateRuleSource([]byte(s)) == nil {
			t.Fatal("accepted", s)
		}
	}
}

func TestLocalRuleRollbackWhenConfigSaveFails(t *testing.T) {
	m, _ := localRuleManager(t)
	initial, _ := m.ReadLocalRuleFile("")
	calls := 0
	m.writeBarrier = func([]byte) error {
		calls++
		if calls > 1 {
			return errors.New("blocked save")
		}
		return nil
	}
	_, err := m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: "rollback", Revision: initial.Revision, Content: json.RawMessage(`{"version":3,"rules":[{"domain":["example.com"]}]}`)})
	if err == nil {
		t.Fatal("expected save failure")
	}
	if _, err := os.Stat(filepath.Join(initial.Directory, "rollback.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("orphaned file", err)
	}
}
func TestLocalRuleInstalledCore(t *testing.T) {
	binary, err := exec.LookPath("sing-box")
	if err != nil {
		t.Skip("sing-box not installed")
	}
	m, _ := localRuleManager(t)
	m = NewManager(m.configPath, binary, "")
	// Actual core rejects unknown top-level fields; use a minimal real configuration.
	if err := os.WriteFile(m.configPath, []byte(`{"route":{"rule_set":[]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	first := createLocalRule(t, m, "real-core")
	_, err = m.WriteLocalRuleFile(context.Background(), LocalRuleWrite{Tag: first.Tag, Existing: true, Revision: first.Revision, Content: json.RawMessage(`{"version":3,"rules":[{"not_a_real_matcher":["example.com"]}]}`)})
	if err == nil {
		t.Fatal("core accepted unknown matcher")
	}
	current, _ := os.ReadFile(first.Path)
	if string(current) != string(first.Content) {
		t.Fatal("invalid content persisted")
	}
}
