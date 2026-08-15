package service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The UCI-driven /etc/init.d/sing-box on OpenWrt returns 0 without starting
// anything when `sing-box.main.enabled` is 0. These tests pin the contract
// that a zero exit code alone is never treated as a successful start.

func TestWaitForServiceStart(t *testing.T) {
	probeErr := errors.New("probe blew up")

	tests := []struct {
		name      string
		responses []bool
		err       error
		wantErr   bool
		wantCalls int
	}{
		{
			name:      "running on first poll",
			responses: []bool{true},
			wantCalls: 1,
		},
		{
			name:      "appears after a few polls",
			responses: []bool{false, false, true},
			wantCalls: 3,
		},
		{
			name:      "never appears",
			responses: []bool{false},
			wantErr:   true,
		},
		{
			name:    "probe error propagates immediately",
			err:     probeErr,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			check := func() (bool, int, error) {
				calls++
				if tt.err != nil {
					return false, 0, tt.err
				}
				idx := calls - 1
				if idx >= len(tt.responses) {
					idx = len(tt.responses) - 1
				}
				if tt.responses[idx] {
					return true, 4242, nil
				}
				return false, 0, nil
			}

			err := waitForServiceStart(check, 250*time.Millisecond, time.Millisecond)

			if tt.wantErr && err == nil {
				t.Fatalf("waitForServiceStart() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("waitForServiceStart() = %v, want nil", err)
			}
			if tt.err != nil && !errors.Is(err, probeErr) {
				t.Errorf("waitForServiceStart() = %v, want it to wrap the probe error", err)
			}
			if tt.wantCalls > 0 && calls != tt.wantCalls {
				t.Errorf("check called %d times, want %d", calls, tt.wantCalls)
			}
		})
	}
}

// newTestProcd builds a procdBackend with both exec seams stubbed so no init
// script or process table is touched.
func newTestProcd(initErr error, running ...bool) (*procdBackend, *[]string) {
	actions := []string{}
	polls := 0
	return &procdBackend{
		initScript:   "/etc/init.d/sing-box",
		logPath:      func() string { return "" },
		startTimeout: 50 * time.Millisecond,
		runInit: func(action string) error {
			actions = append(actions, action)
			return initErr
		},
		probe: func() (bool, int, error) {
			idx := polls
			polls++
			if idx >= len(running) {
				idx = len(running) - 1
			}
			if len(running) == 0 || !running[idx] {
				return false, 0, nil
			}
			return true, 777, nil
		},
	}, &actions
}

func TestProcdStartVerifiesServiceCameUp(t *testing.T) {
	b, actions := newTestProcd(nil, true)

	if err := b.Start(); err != nil {
		t.Fatalf("Start() = %v, want nil", err)
	}
	if len(*actions) != 1 || (*actions)[0] != "start" {
		t.Errorf("actions = %v, want [start]", *actions)
	}
}

func TestProcdStartFailsWhenInitScriptSilentlyNoOps(t *testing.T) {
	// The exact ImmortalWrt failure mode: `/etc/init.d/sing-box start` exits 0
	// because UCI has enabled=0, but no process ever appears.
	b, _ := newTestProcd(nil, false)

	err := b.Start()
	if err == nil {
		t.Fatal("Start() = nil, want an error when the service never comes up")
	}
	// The message has to point the operator at the actual cause.
	if !strings.Contains(err.Error(), "enabled") {
		t.Errorf("Start() error = %q, want it to mention the UCI enabled flag", err)
	}
}

func TestProcdStartDoesNotProbeWhenInitScriptFails(t *testing.T) {
	initErr := errors.New("start failed: exit status 1")
	b, actions := newTestProcd(initErr, false)

	err := b.Start()
	if !errors.Is(err, initErr) {
		t.Fatalf("Start() = %v, want it to wrap the init script error", err)
	}
	if len(*actions) != 1 {
		t.Errorf("actions = %v, want exactly one init invocation", *actions)
	}
}

func TestProcdRestartVerifiesServiceCameUp(t *testing.T) {
	b, actions := newTestProcd(nil, false, true)

	if err := b.Restart(); err != nil {
		t.Fatalf("Restart() = %v, want nil", err)
	}
	if len(*actions) != 1 || (*actions)[0] != "restart" {
		t.Errorf("actions = %v, want [restart]", *actions)
	}
}

func TestProcdRestartFailsWhenServiceNeverAppears(t *testing.T) {
	b, _ := newTestProcd(nil, false)

	if err := b.Restart(); err == nil {
		t.Fatal("Restart() = nil, want an error when the service never comes up")
	}
}

func TestProcdReloadFallsBackToRestartWhenNothingCameUp(t *testing.T) {
	// rc.common's reload is a no-op for a disabled/stopped instance and still
	// exits 0, so a reload that leaves nothing running must escalate.
	b, actions := newTestProcd(nil, false, false, true)

	if err := b.Reload(); err != nil {
		t.Fatalf("Reload() = %v, want nil", err)
	}
	if len(*actions) != 2 || (*actions)[0] != "reload" || (*actions)[1] != "restart" {
		t.Errorf("actions = %v, want [reload restart]", *actions)
	}
}
