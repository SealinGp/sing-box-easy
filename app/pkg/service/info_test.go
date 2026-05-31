package service

import (
	"testing"
	"time"
)

func TestHumanizeDuration(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"sub-minute", 30 * time.Second, "<1m"},
		{"minutes", 14 * time.Minute, "14m"},
		{"hours-mins", 2*time.Hour + 3*time.Minute, "2h3m"},
		{"days-hours", 26 * time.Hour, "1d2h"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := humanizeDuration(tc.d); got != tc.want {
				t.Errorf("humanizeDuration(%v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestParseActiveAndPID(t *testing.T) {
	cases := []struct {
		name       string
		out        string
		wantActive bool
		wantPID    int
	}{
		{"active", "ActiveState=active\nMainPID=1607\n", true, 1607},
		{"inactive zeroes pid", "ActiveState=inactive\nMainPID=0\n", false, 0},
		{"failed", "ActiveState=failed\nMainPID=0\n", false, 0},
		{"active but order swapped", "MainPID=42\nActiveState=active\n", true, 42},
		{"garbage tolerated", "junk\nActiveState=active\nMainPID=99\nextra=1\n", true, 99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			active, pid := parseActiveAndPID(tc.out)
			if active != tc.wantActive || pid != tc.wantPID {
				t.Errorf("parse(%q) = (%v,%d), want (%v,%d)", tc.out, active, pid, tc.wantActive, tc.wantPID)
			}
		})
	}
}
