package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/database"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
)

// TestMain initializes the logger and one process-wide SQLite database
// (database.Init is guarded by sync.Once).
func TestMain(m *testing.M) {
	logger.InitDefault()
	dir, err := os.MkdirTemp("", "settings_test")
	if err != nil {
		panic(err)
	}
	if err := database.Init(filepath.Join(dir, "test.db")); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = database.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// newTestManager returns a manager over an empty settings table, so one test's
// writes cannot become another's "default".
func newTestManager(t *testing.T) *ManagerXORM {
	t.Helper()
	m := NewManagerXORM()
	if err := m.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	e, err := database.GetEngine()
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if _, err := e.Exec("DELETE FROM settings"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return m
}

// TestProbeSettingsDefaults — an install that has never touched these must get
// working values, not zeros. A zero interval would spin the prober.
func TestProbeSettingsDefaults(t *testing.T) {
	m := newTestManager(t)

	if got := m.GetProbeInterval(); got != DefaultProbeInterval {
		t.Errorf("interval = %s, want %s", got, DefaultProbeInterval)
	}
	if got := m.GetProbeTimeout(); got != DefaultProbeTimeoutMs*time.Millisecond {
		t.Errorf("timeout = %s, want %dms", got, DefaultProbeTimeoutMs)
	}
	if got := m.GetProbeRetentionDays(); got != DefaultProbeRetentionDays {
		t.Errorf("retention = %d, want %d", got, DefaultProbeRetentionDays)
	}
	if got := m.GetProbeMaxPoints(); got != DefaultProbeMaxPoints {
		t.Errorf("max points = %d, want %d", got, DefaultProbeMaxPoints)
	}
}

// TestProbeSettingsRoundTrip covers the write path the Settings page uses.
func TestProbeSettingsRoundTrip(t *testing.T) {
	m := newTestManager(t)

	if err := m.SetProbeInterval(30 * time.Minute); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	if got := m.GetProbeInterval(); got != 30*time.Minute {
		t.Errorf("interval = %s, want 30m", got)
	}

	if err := m.SetProbeRetentionDays(30); err != nil {
		t.Fatalf("set retention: %v", err)
	}
	if got := m.GetProbeRetentionDays(); got != 30 {
		t.Errorf("retention = %d, want 30", got)
	}
}

// TestProbeSettingsReject covers the boundary the API reports back to the
// operator. The floor on the interval is the one that matters: it is what stops
// a mistyped value turning the panel into a load generator against a provider.
func TestProbeSettingsReject(t *testing.T) {
	m := newTestManager(t)

	if err := m.SetProbeInterval(10 * time.Second); err == nil {
		t.Error("accepted a 10s interval, want a rejection")
	}
	if err := m.SetProbeInterval(48 * time.Hour); err == nil {
		t.Error("accepted a 48h interval, want a rejection")
	}
	if err := m.SetProbeTimeoutMs(30000); err == nil {
		t.Error("accepted a 30s timeout, want a rejection (the clash client cuts off at 10s)")
	}
	if err := m.SetProbeRetentionDays(0); err == nil {
		t.Error("accepted 0 retention days, want a rejection")
	}
	if err := m.SetProbeMaxPoints(1); err == nil {
		t.Error("accepted a 1-point cap, want a rejection")
	}
}

// TestValidateProbeSettingsDoesNotWrite is the regression test for a real
// defect: the handler validated each knob by calling its setter, and each
// setter commits its own transaction — so a request whose third field was out
// of range had already persisted the first two. The client saw an error and
// reasonably assumed nothing had changed.
//
// Validation must therefore be available WITHOUT writing.
func TestValidateProbeSettingsDoesNotWrite(t *testing.T) {
	m := newTestManager(t)

	if err := ValidateProbeInterval(30 * time.Minute); err != nil {
		t.Errorf("valid interval rejected: %v", err)
	}
	if err := ValidateProbeTimeoutMs(30000); err == nil {
		t.Error("30s timeout accepted, want a rejection")
	}
	if err := ValidateProbeRetentionDays(0); err == nil {
		t.Error("0 retention days accepted, want a rejection")
	}
	if err := ValidateProbeMaxPoints(1); err == nil {
		t.Error("a 1-point cap accepted, want a rejection")
	}

	// Nothing above may have touched storage: the interval must still be the
	// default, not the 30 minutes that was merely validated.
	if got := m.GetProbeInterval(); got != DefaultProbeInterval {
		t.Errorf("interval = %s after validation only, want the untouched default %s", got, DefaultProbeInterval)
	}
}

// TestProbeSettingsClampStoredGarbage — a row written by an older build, or by
// hand, must not disable the prober or the retention bound.
func TestProbeSettingsClampStoredGarbage(t *testing.T) {
	m := newTestManager(t)

	if err := m.Set(KeyProbeInterval, "0"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := m.GetProbeInterval(); got != MinProbeInterval {
		t.Errorf("interval = %s, want it clamped to %s", got, MinProbeInterval)
	}

	if err := m.Set(KeyProbeMaxPoints, "not-a-number"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := m.GetProbeMaxPoints(); got != DefaultProbeMaxPoints {
		t.Errorf("max points = %d, want the default", got)
	}

	if err := m.Set(KeyProbeRetentionDays, "99999"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := m.GetProbeRetentionDays(); got != MaxProbeRetentionDays {
		t.Errorf("retention = %d, want it clamped to %d", got, MaxProbeRetentionDays)
	}
}
