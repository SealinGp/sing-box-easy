package settings

import (
	"context"
	"testing"
)

type retentionRecorder struct{ keep int }

func (r *retentionRecorder) SetKeepVersions(n int) { r.keep = n }

func TestUpdateSettingsValidatesWholeFormBeforeWriting(t *testing.T) {
	store := newTestManager(t)
	if err := store.SetConfigVersionsKeep(10); err != nil {
		t.Fatal(err)
	}
	live := &retentionRecorder{keep: 10}
	svc := NewService(store, live, func() bool { return false })
	keep, invalid := 20, "bad"
	if _, err := svc.UpdateSettings(context.Background(), UpdateSettingsCommand{ConfigVersionsKeep: &keep, GitHubOAuthClientID: &invalid}); err == nil {
		t.Fatal("invalid client ID accepted")
	}
	if store.GetConfigVersionsKeep() != 10 || live.keep != 10 {
		t.Fatal("rejected form changed persisted or live retention")
	}
	valid := "Ov23liAbCdEf01234567"
	if _, err := svc.UpdateSettings(context.Background(), UpdateSettingsCommand{ConfigVersionsKeep: &keep, GitHubOAuthClientID: &valid}); err != nil {
		t.Fatal(err)
	}
	if store.GetConfigVersionsKeep() != 20 || live.keep != 20 || store.GetGitHubOAuthClientID() != valid {
		t.Fatal("valid form was not applied")
	}
}
