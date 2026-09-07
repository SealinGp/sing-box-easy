package subscription

import (
	"errors"
	"testing"
	"time"
)

type fakeServiceRestarter struct {
	calls int
	err   error
}

func (f *fakeServiceRestarter) Restart() error {
	f.calls++
	return f.err
}

func TestRestartService(t *testing.T) {
	t.Run("calls configured restarter", func(t *testing.T) {
		restarter := &fakeServiceRestarter{}
		au := &AutoUpdater{serviceRestarter: restarter}

		if err := au.restartService(); err != nil {
			t.Fatalf("restartService() = %v", err)
		}
		if restarter.calls != 1 {
			t.Fatalf("Restart calls = %d, want 1", restarter.calls)
		}
	})

	t.Run("propagates restart failure", func(t *testing.T) {
		want := errors.New("restart failed")
		au := &AutoUpdater{serviceRestarter: &fakeServiceRestarter{err: want}}

		if err := au.restartService(); !errors.Is(err, want) {
			t.Fatalf("restartService() = %v, want %v", err, want)
		}
	})

	t.Run("rejects missing restarter", func(t *testing.T) {
		if err := (&AutoUpdater{}).restartService(); err == nil {
			t.Fatal("restartService() = nil, want configuration error")
		}
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		// Standard time.ParseDuration cases
		{"hours", "24h", 24 * time.Hour},
		{"minutes", "30m", 30 * time.Minute},
		{"seconds", "10s", 10 * time.Second},
		{"mixed", "1h30m", 90 * time.Minute},

		// Custom day/week/month suffixes
		{"days short", "7d", 7 * 24 * time.Hour},
		{"days long", "3days", 3 * 24 * time.Hour},
		{"weeks short", "2w", 2 * 7 * 24 * time.Hour},
		{"weeks long", "1week", 7 * 24 * time.Hour},
		{"months short", "1mo", 30 * 24 * time.Hour},
		{"months long", "2months", 60 * 24 * time.Hour},

		// Fallbacks
		{"empty defaults to 24h", "", 24 * time.Hour},
		{"unknown unit defaults to 24h", "5xyz", 24 * time.Hour},
		{"non-numeric defaults to 24h", "abc", 24 * time.Hour},
		{"zero defaults to 24h", "0d", 24 * time.Hour},

		// Case insensitive + whitespace
		{"uppercase", "7D", 7 * 24 * time.Hour},
		{"padded", "  7d  ", 7 * 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDuration(tt.in)
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
