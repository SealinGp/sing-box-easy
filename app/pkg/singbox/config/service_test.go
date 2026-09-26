package config

import (
	"context"
	"errors"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
)

// The service exists to put the whole-document and history rules in one place
// the HTTP layer cannot bypass. These pin the classification the handlers used
// to do inline, message for message, so moving it changed no response.

func faultKind(t *testing.T, err error) fault.Kind {
	t.Helper()
	var e *fault.Error
	if !errors.As(err, &e) {
		t.Fatalf("error %v is not a fault.Error", err)
	}
	return e.Kind
}

func TestServiceRejectsNonPositiveVersionIDs(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	s := NewService(m)
	calls := map[string]func() error{
		"HistoryVersion": func() error { _, err := s.HistoryVersion(0); return err },
		"DeleteHistory":  func() error { return s.DeleteHistory(-1) },
		"RestoreVersion": func() error { return s.RestoreVersion(0) },
	}
	for name, call := range calls {
		err := call()
		if faultKind(t, err) != fault.Input || err.Error() != "invalid version id" {
			t.Errorf("%s: got %v, want input fault %q", name, err, "invalid version id")
		}
	}
}

func TestServiceReportsAMissingVersionAsMissing(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	s := NewService(m)
	if _, err := s.HistoryVersion(42); faultKind(t, err) != fault.Missing {
		t.Errorf("HistoryVersion(42) = %v, want a missing fault", err)
	}
	if err := s.DeleteHistory(42); faultKind(t, err) != fault.Missing {
		t.Errorf("DeleteHistory(42) = %v, want a missing fault", err)
	}
}

func TestServiceValidatesBatchDeleteIDs(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	s := NewService(m)
	cases := map[string]struct {
		ids  []int64
		want string
	}{
		"empty":        {nil, "ids array is required and cannot be empty"},
		"non-positive": {[]int64{3, 0}, "ids must be positive integers"},
	}
	for name, tc := range cases {
		_, err := s.DeleteHistoryBatch(tc.ids)
		if faultKind(t, err) != fault.Input || err.Error() != tc.want {
			t.Errorf("%s: got %v, want input fault %q", name, err, tc.want)
		}
	}
}

type failingCore struct{}

func (failingCore) Version(context.Context) (core.CoreVersion, error) {
	return core.CoreVersion{}, errors.New("exec: sing-box not found")
}
func (failingCore) Validate(context.Context, string) error { return nil }

func TestServiceReportsAnUnreachableCoreAsUnavailable(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	m.core = failingCore{}
	if _, err := NewService(m).CoreInfo(context.Background()); faultKind(t, err) != fault.Unavailable {
		t.Errorf("CoreInfo = %v, want an unavailable fault", err)
	}
}

func TestServicePassesValidationErrorsThroughUnwrapped(t *testing.T) {
	m, _ := newTestManager(t, newFakeStore(), 10)
	m.core = &fakeCoreAdapter{validateErr: errors.New("unknown field")}
	err := NewService(m).Validate(context.Background(), []byte(`{}`))
	// The handler reads the stage and core version off this type; wrapping it
	// in a fault would strip the fields the UI shows next to the error.
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Validate = %v, want a *ValidationError", err)
	}
}
