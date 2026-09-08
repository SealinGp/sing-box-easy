package subscription

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/model"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/repo"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func deletionManager(t *testing.T) (*Service, string, *fakeServiceRestarter) {
	t.Helper()
	store := newTestManager(t)
	if err := store.Add(Subscription{ID: "sub_1", Name: "one", URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "sing-box")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	raw := `{"dns":{"rules":[{"action":"evaluate","future":true}]},"outbounds":[
 {"type":"socks","tag":"node | sub_1","server":"shared.example","server_port":1080},
 {"type":"socks","tag":"node | sub_10","server":"shared.example","server_port":1080},
 {"type":"direct","tag":"manual"},
 {"type":"selector","tag":"proxy","outbounds":["node | sub_1","node | sub_10","manual"],"default":"node | sub_1"}
 ]}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	restarter := &fakeServiceRestarter{}
	return newService(config.NewManager(path, binary, ""), store, nil, nil, nil, restarter), path, restarter
}

func TestDeleteSubscriptionRemovesOwnedNodes(t *testing.T) {
	manager, path, restarter := deletionManager(t)
	manager.recordSuccess("sub_1")
	if err := manager.Delete("sub_1"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Get("sub_1"); err == nil {
		t.Fatal("subscription still exists")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"node | sub_10", "manual", "evaluate", "future"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing %q in %s", want, raw)
		}
	}
	if strings.Contains(string(raw), `"node | sub_1"`) {
		t.Fatalf("owned node or reference survived: %s", raw)
	}
	if restarter.calls != 1 {
		t.Fatalf("restart calls = %d", restarter.calls)
	}
	if _, ok := manager.GetStats()["sub_1"]; ok {
		t.Fatal("deleted subscription stats survived")
	}
	// A cron snapshot obtained before deletion must not fetch or restore nodes.
	if _, err := manager.UpdateSubscription(&Subscription{ID: "sub_1"}); err == nil {
		t.Fatal("stale refresh succeeded")
	}
	if _, ok := manager.GetStats()["sub_1"]; ok {
		t.Fatal("stale refresh recreated stats")
	}
}

func TestDeleteSubscriptionRestartFailureCanRetry(t *testing.T) {
	manager, _, restarter := deletionManager(t)
	restarter.err = errors.New("restart failed")
	if err := manager.Delete("sub_1"); !errors.Is(err, restarter.err) {
		t.Fatalf("error = %v", err)
	}
	if _, err := manager.Get("sub_1"); err != nil {
		t.Fatalf("record lost on failure: %v", err)
	}
	restarter.err = nil
	if err := manager.Delete("sub_1"); err != nil {
		t.Fatal(err)
	}
	if restarter.calls != 2 {
		t.Fatalf("retry did not restart: %d", restarter.calls)
	}
}

func TestDeleteSubscriptionConfigFailureRetainsRecord(t *testing.T) {
	manager, path, restarter := deletionManager(t)
	// The validator rejects the mutation; neither the file nor DB should change.
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "sing-box"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := manager.Delete("sub_1"); err == nil {
		t.Fatal("expected validation error")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("failed deletion changed config")
	}
	if _, err := manager.Get("sub_1"); err != nil {
		t.Fatal(err)
	}
	if restarter.calls != 0 {
		t.Fatal("restarted after validation failure")
	}
}

func TestDeleteMissingSubscriptionDoesNotChangeConfig(t *testing.T) {
	manager, path, restarter := deletionManager(t)
	before, _ := os.ReadFile(path)
	if err := manager.Delete("missing"); err == nil {
		t.Fatal("expected missing subscription error")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) || restarter.calls != 0 {
		t.Fatal("missing subscription caused side effects")
	}
}

func TestDeletionRecoveryRemovesHistoryAndPendingState(t *testing.T) {
	manager, _, restarter := deletionManager(t)
	e, err := database.GetEngine()
	if err != nil {
		t.Fatal(err)
	}
	history := repo.NewProbeStore(e)
	if err := history.Insert("sub_1", time.Now(), model.Sample{Total: 1, Reachable: 1}); err != nil {
		t.Fatal(err)
	}
	restarter.err = errors.New("engine unavailable")
	if err := manager.Delete("sub_1"); err == nil {
		t.Fatal("expected restart failure")
	}
	pending, err := manager.repository.(deletionRepository).IsDeleting("sub_1")
	if err != nil || !pending {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
	// A new service instance resumes the persisted operation before its timers start.
	restarter.err = nil
	recovered := newService(manager.configManager, manager.repository, nil, nil, nil, restarter)
	if err := recovered.StartBackground("*/5 * * * *"); err != nil {
		t.Fatal(err)
	}
	recovered.StopBackground()
	if _, err := recovered.Get("sub_1"); err == nil {
		t.Fatal("record survived recovery")
	}
	if n, err := history.CountAll(); err != nil || n != 0 {
		t.Fatalf("history=%d err=%v", n, err)
	}
	ids, err := manager.repository.(deletionRepository).PendingDeletions()
	if err != nil || len(ids) != 0 {
		t.Fatalf("pending=%v err=%v", ids, err)
	}
}

func TestDeletionDatabaseFailureRollsBackHistory(t *testing.T) {
	manager, _, _ := deletionManager(t)
	e, err := database.GetEngine()
	if err != nil {
		t.Fatal(err)
	}
	history := repo.NewProbeStore(e)
	if err := history.Insert("sub_1", time.Now(), model.Sample{Total: 1, Reachable: 1}); err != nil {
		t.Fatal(err)
	}
	_, err = e.Exec("CREATE TRIGGER reject_subscription_delete BEFORE DELETE ON subscriptions BEGIN SELECT RAISE(FAIL, 'simulated database failure'); END")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = e.Exec("DROP TRIGGER IF EXISTS reject_subscription_delete") })
	if err := manager.Delete("sub_1"); err == nil {
		t.Fatal("expected database failure")
	}
	if _, err := manager.Get("sub_1"); err != nil {
		t.Fatal("record removed despite failed transaction")
	}
	if n, err := history.CountAll(); err != nil || n != 1 {
		t.Fatalf("history lost on failed transaction: %d %v", n, err)
	}
	if _, err := e.Exec("DROP TRIGGER reject_subscription_delete"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Delete("sub_1"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndPartialEditUseBusinessDefaults(t *testing.T) {
	manager, _, _ := deletionManager(t)
	name, url := "provider", "https://example.com/feed"
	created, err := manager.Create(EditCommand{Name: &name, URL: &url})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || !created.ProbeEnabled {
		t.Fatalf("missing identity/default: %+v", created)
	}
	disabled := false
	edited, err := manager.Edit(created.ID, EditCommand{ProbeEnabled: &disabled})
	if err != nil {
		t.Fatal(err)
	}
	if edited.URL != url || edited.Name != name || edited.ProbeEnabled {
		t.Fatalf("partial edit reset fields: %+v", edited)
	}
	if _, err := manager.Edit(created.ID, EditCommand{Name: ptr("")}); err == nil {
		t.Fatal("accepted empty name")
	}
}
func ptr(s string) *string { return &s }

type cancelledFeed struct{ started chan struct{} }

func (f cancelledFeed) Resolve(ctx context.Context, _ []string, _ sublink.FetchOptions) ([]*node.SubNode, *sublink.FetchMeta, error) {
	close(f.started)
	<-ctx.Done()
	return nil, nil, ctx.Err()
}
func TestDeleteDrainsAnInFlightRefresh(t *testing.T) {
	manager, _, _ := deletionManager(t)
	feed := cancelledFeed{started: make(chan struct{})}
	manager.sublinkManager = feed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	refreshed := make(chan error, 1)
	go func() { _, err := manager.Refresh(ctx, "sub_1"); refreshed <- err }()
	select {
	case <-feed.started:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not start")
	}
	deleted := make(chan error, 1)
	go func() { deleted <- manager.Delete("sub_1") }()
	// Deletion cannot commit before the refresh leaves its subscription gate.
	select {
	case err := <-deleted:
		t.Fatalf("delete overtook refresh: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	if err := <-refreshed; !errors.Is(err, context.Canceled) {
		t.Fatalf("refresh ignored cancellation: %v", err)
	}
	if err := <-deleted; err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Get("sub_1"); err == nil {
		t.Fatal("deleted subscription returned")
	}
	if _, ok := manager.GetStats()["sub_1"]; ok {
		t.Fatal("refresh recreated statistics")
	}
}
