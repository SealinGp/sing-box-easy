package subscription

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
)

func newTestManager(t *testing.T) *ManagerXORM {
	t.Helper()
	m := NewManagerXORM()
	if err := m.Init(); err != nil {
		t.Fatalf("manager init: %v", err)
	}
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("get engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM subscriptions"); err != nil {
		t.Fatalf("truncate subscriptions: %v", err)
	}
	return m
}

// The official site has to survive every path a subscription takes through the
// store — insert, the targeted refresh write, an operator edit, and both reads.
// A column missing from any one of those column lists loses the value silently,
// which is exactly the kind of bug that only shows up after the next refresh.
func TestOfficialURLRoundTrip(t *testing.T) {
	m := newTestManager(t)

	sub := Subscription{
		ID:             "sub_official_1",
		Name:           "Provider",
		URL:            "https://example.com/sub",
		UpdateInterval: "24h",
	}
	if err := m.Add(sub); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := m.Get(sub.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.OfficialURL != "" {
		t.Errorf("new subscription OfficialURL = %q, want empty", got.OfficialURL)
	}

	// The refresh path: a targeted single-column write.
	if err := m.UpdateOfficialURL(sub.ID, "https://provider.example.com"); err != nil {
		t.Fatalf("UpdateOfficialURL: %v", err)
	}
	got, err = m.Get(sub.ID)
	if err != nil {
		t.Fatalf("Get after UpdateOfficialURL: %v", err)
	}
	if got.OfficialURL != "https://provider.example.com" {
		t.Errorf("OfficialURL = %q, want the refreshed value", got.OfficialURL)
	}
	// The single-column write must not disturb the rest of the row.
	if got.Name != "Provider" || got.URL != "https://example.com/sub" {
		t.Errorf("UpdateOfficialURL clobbered other fields: %+v", got)
	}

	// The operator edit path: a full Update.
	edited := *got
	edited.OfficialURL = "https://operator.example.com"
	if err := m.Update(sub.ID, edited); err != nil {
		t.Fatalf("Update: %v", err)
	}
	list, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List returned %d subscriptions, want 1", len(list))
	}
	if list[0].OfficialURL != "https://operator.example.com" {
		t.Errorf("List OfficialURL = %q, want the edited value", list[0].OfficialURL)
	}

	// Clearing it must stick too — otherwise an operator cannot remove a link
	// the provider got wrong.
	edited.OfficialURL = ""
	if err := m.Update(sub.ID, edited); err != nil {
		t.Fatalf("Update (clear): %v", err)
	}
	got, err = m.Get(sub.ID)
	if err != nil {
		t.Fatalf("Get after clear: %v", err)
	}
	if got.OfficialURL != "" {
		t.Errorf("OfficialURL = %q after clearing, want empty", got.OfficialURL)
	}
}

func TestUpdateOfficialURLUnknownID(t *testing.T) {
	m := newTestManager(t)
	if err := m.UpdateOfficialURL("nope", "https://example.com"); err == nil {
		t.Error("UpdateOfficialURL on an unknown id = nil error, want failure")
	}
}
