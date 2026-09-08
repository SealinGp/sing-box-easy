package user

import (
	"errors"
	"reflect"
	"testing"

	"xorm.io/xorm"
)

func TestPreferencesIsolationAndPersistence(t *testing.T) {
	m := newTestManager(t)
	first, err := m.CreateUser("first", "password", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.CreateUser("second", "password", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	initial, err := m.GetPreferences(first.ID)
	if err != nil || !reflect.DeepEqual(initial.OverviewOrder, overviewCards) {
		t.Fatalf("defaults: %+v, %v", initial, err)
	}
	wanted := []string{"api-endpoints", "dns-probe"}
	if _, err := m.UpdatePreferences(first.ID, Preferences{OverviewOrder: wanted}); err != nil {
		t.Fatal(err)
	}
	reloaded := NewManagerXORM(testEngine(), "", "")
	got, err := reloaded.GetPreferences(first.ID)
	if err != nil || !reflect.DeepEqual(got.OverviewOrder, normalizeOverviewOrder(wanted)) {
		t.Fatalf("reload: %+v, %v", got, err)
	}
	other, err := reloaded.GetPreferences(second.ID)
	if err != nil || !reflect.DeepEqual(other.OverviewOrder, overviewCards) {
		t.Fatalf("other user changed: %+v, %v", other, err)
	}
	for _, invalid := range [][]string{{"dns-probe", "dns-probe"}, {"unknown"}} {
		if _, err := m.UpdatePreferences(first.ID, Preferences{OverviewOrder: invalid}); !errors.Is(err, ErrInvalidPreferences) {
			t.Fatalf("invalid order accepted: %v", err)
		}
	}
	unchanged, err := m.GetPreferences(first.ID)
	if err != nil || !reflect.DeepEqual(unchanged, got) {
		t.Fatalf("invalid update changed preferences: %+v, %v", unchanged, err)
	}
	u, _, err := m.Authenticate("first", "password")
	if err != nil || u.Role != "viewer" {
		t.Fatalf("profile changed: %+v, %v", u, err)
	}
	reset, err := m.UpdatePreferences(first.ID, Preferences{OverviewOrder: []string{}})
	if err != nil || !reflect.DeepEqual(reset.OverviewOrder, overviewCards) {
		t.Fatalf("reset: %+v, %v", reset, err)
	}
	if _, err := m.UpdatePreferences(99999, Preferences{}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}
}

func TestPreferencesUpgradeExistingDatabase(t *testing.T) {
	e, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.SetMaxOpenConns(1)
	_, err = e.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'viewer', created_at DATETIME, updated_at DATETIME)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Exec(`INSERT INTO users (username, password_hash, role) VALUES ('legacy', 'hash', 'viewer')`); err != nil {
		t.Fatal(err)
	}
	m := &ManagerXORM{e: e}
	// Sync2 adds the preference column to existing databases and is idempotent.
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetPreferences(1)
	if err != nil || !reflect.DeepEqual(got.OverviewOrder, overviewCards) {
		t.Fatalf("legacy defaults: %+v, %v", got, err)
	}
	if _, err := m.UpdatePreferences(1, Preferences{OverviewOrder: []string{"dns-probe"}}); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeRetiredCards(t *testing.T) {
	got := normalizeOverviewOrder([]string{"retired", "dns-probe", "dns-probe"})
	if got[0] != "dns-probe" || len(got) != len(overviewCards) {
		t.Fatalf("unexpected normalized order: %v", got)
	}
}
