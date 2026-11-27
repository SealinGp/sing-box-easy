package subscription

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sectionsFor(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var sections map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &sections); err != nil {
		t.Fatal(err)
	}
	return sections
}
func TestOrphanPlanPreservesKnownAndManualNodes(t *testing.T) {
	raw := `{"route":{"final":"manual"},"outbounds":[
 {"tag":"manual","type":"direct","future":{"enabled":true}},
 {"tag":"node | sub_1","type":"future-protocol","secret":"keep"},
 {"tag":"node | sub_10","type":"future-protocol","secret":"remove"},
 {"tag":"node sub_10","type":"direct"},
 {"tag":"group","type":"selector","future_option":true,"outbounds":["node | sub_10","node | sub_1","manual"],"default":"node | sub_10"}]}`
	sections := sectionsFor(t, raw)
	plan, updated, err := orphanPlan(sections, map[string]bool{"sub_1": true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.NodeCount != 1 || len(plan.Subscriptions) != 1 || plan.Subscriptions[0].ID != "sub_10" {
		t.Fatalf("%+v", plan)
	}
	if len(plan.Blockers) != 0 || len(plan.UpdatedGroups) != 1 {
		t.Fatalf("%+v", plan)
	}
	if strings.Contains(string(updated), `"node | sub_10"`) || strings.Contains(string(updated), `"default"`) {
		t.Fatal(string(updated))
	}
	for _, value := range []string{"node | sub_1", "node sub_10", "future_option", "future-protocol", "keep"} {
		if !strings.Contains(string(updated), value) {
			t.Fatal("lost", value)
		}
	}
	if !strings.Contains(string(sections["outbounds"]), `"node | sub_10"`) {
		t.Fatal("mutated input")
	}
}
func TestOrphanGroupsCascadeAndBlockExternalReferences(t *testing.T) {
	sections := sectionsFor(t, `{"route":{"final":"parent"},"dns":{"servers":[{"tag":"dns","detour":"node | sub_gone"}]},"outbounds":[
 {"type":"direct","tag":"manual"},
 {"type":"future","tag":"node | sub_gone"},
 {"type":"selector","tag":"child","outbounds":["node | sub_gone"]},
 {"type":"urltest","tag":"parent","outbounds":["child"]},
 {"type":"socks","tag":"chained","detour":"child"}]}`)
	plan, updated, err := orphanPlan(sections, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.RemovedGroups) != 2 || len(plan.Blockers) < 3 {
		t.Fatalf("%+v", plan)
	}
	if strings.Contains(string(updated), `"tag":"parent"`) {
		t.Fatal(string(updated))
	}
}
func TestOrphanImplicitDefaultCannotBeRemoved(t *testing.T) {
	p, _, err := orphanPlan(sectionsFor(t, `{"outbounds":[{"tag":"first | sub_old","type":"direct"},{"tag":"manual","type":"direct"}]}`), nil)
	if err != nil || len(p.Blockers) != 1 || !strings.Contains(p.Blockers[0], "implicit") {
		t.Fatalf("%+v %v", p, err)
	}
}
func TestOrphanCleanupBacksUpAndDoesNotRestart(t *testing.T) {
	service, path, restarter := deletionManager(t)
	before, _ := os.ReadFile(path)
	plan, err := service.DetectOrphanSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.CleanupOrphanSubscriptions(context.Background(), plan.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedNodes != 1 || !result.RestartRequired || restarter.calls != 0 {
		t.Fatalf("%+v", result)
	}
	backup, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(before, &a)
	_ = json.Unmarshal(backup, &b)
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if string(aa) != string(bb) {
		t.Fatal("backup differs")
	}
	after, _ := os.ReadFile(path)
	if strings.Contains(string(after), "node | sub_10") || !strings.Contains(string(after), "node | sub_1") {
		t.Fatal(string(after))
	}
	clean, err := service.DetectOrphanSubscriptions()
	if err != nil || clean.NodeCount != 0 {
		t.Fatalf("%+v %v", clean, err)
	}
	if _, err = service.CleanupOrphanSubscriptions(context.Background(), plan.Revision); err == nil {
		t.Fatal("stale preview allowed")
	}
}
func TestOrphanCleanupRejectsRestoredSubscription(t *testing.T) {
	service, path, _ := deletionManager(t)
	plan, err := service.DetectOrphanSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := service.add(Subscription{ID: "sub_10", Name: "restored", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.CleanupOrphanSubscriptions(context.Background(), plan.Revision); err == nil {
		t.Fatal("restored subscription removed")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("config changed")
	}
}
func TestOrphanCleanupRejectsChangedConfiguration(t *testing.T) {
	service, path, _ := deletionManager(t)
	plan, _ := service.DetectOrphanSubscriptions()
	before, _ := os.ReadFile(path)
	changed := strings.Replace(string(before), "manual", "manual-new", -1)
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CleanupOrphanSubscriptions(context.Background(), plan.Revision); err == nil {
		t.Fatal("stale preview allowed")
	}
	after, _ := os.ReadFile(path)
	if string(after) != changed {
		t.Fatal("external edit lost")
	}
}
func TestOrphanCleanupCoreValidationFailureDoesNotWrite(t *testing.T) {
	service, path, _ := deletionManager(t)
	plan, _ := service.DetectOrphanSubscriptions()
	// deletionManager's fake core is located beside config.json.
	binary := strings.TrimSuffix(path, "config.json") + "sing-box"
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if _, err := service.CleanupOrphanSubscriptions(context.Background(), plan.Revision); err == nil {
		t.Fatal("validation failure ignored")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid config persisted")
	}
}
func TestOrphanOwnerRequiresExactSuffix(t *testing.T) {
	for tag, want := range map[string]string{"node | sub_1": "sub_1", "node | sub_10": "sub_10", "node sub_1": "", "sub_1": "", "node | sub_1 trailing": "", "node | abc": ""} {
		if got := orphanOwner(tag); got != want {
			t.Fatalf("%s => %s", tag, got)
		}
	}
}
