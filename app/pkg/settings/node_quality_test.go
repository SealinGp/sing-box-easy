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
	for _, bad := range []NodeQuality{{0, 500, 80, 50}, {101, 500, 80, 50}, {10, 0, 80, 50}, {10, 500, 0, 50}, {10, 500, 101, 50}} {
		if bad.Validate() == nil {
			t.Fatalf("invalid criteria accepted: %+v", bad)
		}
	}
}

func TestNodeQualityWatermarksAndLegacySettings(t *testing.T) {
	if q := DefaultNodeQuality(); q.Samples != 100 || q.GoodPercent != 97 || q.MaxDelayMs != 500 {
		t.Fatalf("unexpected defaults: %+v", q)
	}
	store := newTestManager(t)
	legacy := `{"samples":20,"max_delay_ms":300,"good_percent":90,"fair_percent":60}`
	if err := store.Set(KeyNodeQuality, legacy); err != nil {
		t.Fatal(err)
	}
	if got := store.GetNodeQuality(); got.Samples != 20 || got.GoodPercent != 90 || got.MaxDelayMs != 300 {
		t.Fatalf("legacy override lost: %+v", got)
	}
	for _, percent := range []int{85, 93, 95, 97} {
		q := DefaultNodeQuality()
		q.GoodPercent = percent
		q.FairPercent = 99 // This obsolete field cannot constrain the new watermark.
		if err := q.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
