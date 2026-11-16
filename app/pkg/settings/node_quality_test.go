package settings

import (
	"context"
	"testing"
)

func TestNodeQualityDefaultsPersistenceAndAtomicValidation(t *testing.T) {
	store := newTestManager(t)
	if got := store.GetNodeQuality(); got != DefaultNodeQuality() {
		t.Fatalf("defaults: %+v", got)
	}
	svc := NewService(store, &retentionRecorder{}, func() bool { return false })
	q := NodeQuality{Samples: 20, MaxDelayMs: 300, GoodPercent: 90, FairPercent: 60}
	if _, err := svc.UpdateSettings(context.Background(), UpdateSettingsCommand{NodeQuality: &q}); err != nil {
		t.Fatal(err)
	}
	if store.GetNodeQuality() != q {
		t.Fatal("quality criteria were not persisted")
	}
	badID := "invalid"
	changed := DefaultNodeQuality()
	if _, err := svc.UpdateSettings(context.Background(), UpdateSettingsCommand{NodeQuality: &changed, GitHubOAuthClientID: &badID}); err == nil {
		t.Fatal("invalid update accepted")
	}
	if store.GetNodeQuality() != q {
		t.Fatal("rejected update changed quality criteria")
	}
	for _, bad := range []NodeQuality{{0, 500, 80, 50}, {101, 500, 80, 50}, {10, 0, 80, 50}, {10, 500, 50, 50}, {10, 500, 101, 50}, {10, 500, 80, 0}} {
		if bad.Validate() == nil {
			t.Fatalf("invalid criteria accepted: %+v", bad)
		}
	}
}
