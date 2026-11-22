package autoselect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"go.uber.org/zap"
)

const key = "proxy_auto_selection"

type Policy struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"interval_seconds"`
}
type Status struct {
	Policy
	Reason       string    `json:"reason"`
	LastChecked  time.Time `json:"last_checked,omitempty"`
	NextCheck    time.Time `json:"next_check,omitempty"`
	LastSwitched time.Time `json:"last_switched,omitempty"`
	Selected     string    `json:"selected,omitempty"`
	Error        string    `json:"error,omitempty"`
}
type Settings interface {
	Get(string) (string, error)
	Set(string, string) error
	GetNodeQuality() settings.NodeQuality
	GetProbeInterval() time.Duration
}
type Runtime interface {
	Proxies(context.Context) ([]clashapi.Proxy, error)
	SelectProxy(context.Context, string, string) error
}
type History interface {
	Recent([]string, int) (map[string][]nodehistory.Point, error)
}
type Manager struct {
	mu          sync.Mutex
	statusMu    sync.Mutex
	lifecycleMu sync.Mutex
	settings    Settings
	history     History
	connect     func() (Runtime, error)
	policies    map[string]Policy
	states      map[string]Status
	cancel      context.CancelFunc
	done        chan struct{}
}

func New(settingsStore Settings, history History, connect func() (Runtime, error)) *Manager {
	return &Manager{settings: settingsStore, history: history, connect: connect, policies: map[string]Policy{}, states: map[string]Status{}}
}
func (m *Manager) Init() error {
	raw, err := m.settings.Get(key)
	if errors.Is(err, settings.ErrSettingNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(raw), &m.policies); err != nil {
		return err
	}
	if m.policies == nil {
		m.policies = map[string]Policy{}
	}
	for group, p := range m.policies {
		if err = validate(p); err != nil {
			return err
		}
		m.setStatus(group, initialStatus(p))
	}
	return nil
}
func validate(p Policy) error {
	if p.IntervalSeconds < 60 || p.IntervalSeconds > 86400 {
		return fault.New(fault.Input, "interval_seconds must be between 60 and 86400")
	}
	return nil
}
func (m *Manager) persist(policies map[string]Policy) error {
	raw, err := json.Marshal(policies)
	if err == nil {
		err = m.settings.Set(key, string(raw))
	}
	return err
}

// Configure serializes policy changes against evaluations and persists before
// activation. Disabled policies can be removed even when sing-box is offline.
func (m *Manager) Configure(ctx context.Context, group string, p Policy) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	group = strings.TrimSpace(group)
	if group == "" {
		return Status{}, fault.New(fault.Input, "group is required")
	}
	if err := validate(p); err != nil {
		return Status{}, err
	}
	if p.Enabled {
		runtime, err := m.connect()
		if err != nil {
			return Status{}, err
		}
		list, err := runtime.Proxies(ctx)
		if err != nil {
			return Status{}, err
		}
		found := false
		for _, g := range list {
			if g.Name == group && g.Type == "Selector" {
				found = true
				break
			}
		}
		if !found {
			return Status{}, fault.New(fault.Input, "quality auto-selection requires a running Selector group")
		}
	}
	next := map[string]Policy{}
	for k, v := range m.policies {
		next[k] = v
	}
	next[group] = p
	if err := m.persist(next); err != nil {
		return Status{}, err
	}
	m.policies = next
	state := initialStatus(p)
	previous := m.state(group)
	state.LastSwitched = previous.LastSwitched
	state.Selected = previous.Selected
	m.setStatus(group, state)
	return state, nil
}
func initialStatus(p Policy) Status {
	s := Status{Policy: p, Reason: "disabled"}
	if p.Enabled {
		s.Reason = "waiting"
	}
	return s
}
func (m *Manager) setStatus(group string, s Status) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.states[group] = s
}
func (m *Manager) state(group string) Status {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	return m.states[group]
}
func (m *Manager) Statuses() map[string]Status {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	result := map[string]Status{}
	for group, s := range m.states {
		result[group] = s
	}
	return result
}

// Pause completes before a manual selection, preventing a concurrent scheduler
// tick from undoing the user's explicit choice.
func (m *Manager) Pause(group string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.policies[group]
	if !ok || !p.Enabled {
		return nil
	}
	p.Enabled = false
	next := map[string]Policy{}
	for k, v := range m.policies {
		next[k] = v
	}
	next[group] = p
	if err := m.persist(next); err != nil {
		return err
	}
	m.policies = next
	m.setStatus(group, Status{Policy: p, Reason: "disabled"})
	return nil
}
func (m *Manager) Start() {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.done = make(chan struct{})
	go func() {
		defer close(m.done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.Tick(ctx, time.Now())
			}
		}
	}()
}
func (m *Manager) Close() {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()
	if m.cancel != nil {
		m.cancel()
		<-m.done
		m.cancel = nil
	}
}
func (m *Manager) Tick(ctx context.Context, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for group, p := range m.policies {
		if ctx.Err() != nil {
			return
		}
		state := m.state(group)
		if !p.Enabled || now.Before(state.NextCheck) {
			continue
		}
		state.Policy = p
		state.LastChecked = now
		state.NextCheck = now.Add(time.Duration(p.IntervalSeconds) * time.Second)
		state.Error = ""
		runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		selected, reason, err := m.evaluate(runCtx, group, now)
		cancel()
		state.Reason = reason
		if err != nil {
			state.Reason = "error"
			state.Error = err.Error()
		}
		if err == nil && selected != "" {
			state.Selected = selected
		}
		if err == nil && reason == "switched" {
			state.LastSwitched = now
			logger.Info("Quality auto-selection switched node", zap.String("group", group), zap.String("node", selected))
		}
		m.setStatus(group, state)
	}
}
func (m *Manager) evaluate(ctx context.Context, name string, now time.Time) (string, string, error) {
	runtime, err := m.connect()
	if err != nil {
		return "", "", err
	}
	list, err := runtime.Proxies(ctx)
	if err != nil {
		return "", "", err
	}
	var group *clashapi.Proxy
	for i := range list {
		if list[i].Name == name && list[i].Type == "Selector" {
			group = &list[i]
			break
		}
	}
	if group == nil {
		return "", "group_missing", nil
	}
	rules := m.settings.GetNodeQuality()
	points, err := m.history.Recent(group.All, rules.Samples)
	if err != nil {
		return "", "", err
	}
	maxAge := 2 * m.settings.GetProbeInterval()
	if maxAge < 15*time.Minute {
		maxAge = 15 * time.Minute
	}
	selected, reason := Decide(*group, list, points, rules, now, maxAge)
	if reason != "switched" {
		return selected, reason, nil
	}
	// Re-read membership/current selection immediately before applying the decision.
	fresh, err := runtime.Proxies(ctx)
	if err != nil {
		return "", "", err
	}
	for _, g := range fresh {
		if g.Name == name && g.Type == "Selector" && g.Now == group.Now {
			for _, tag := range g.All {
				if tag == selected {
					return selected, reason, runtime.SelectProxy(ctx, name, selected)
				}
			}
		}
	}
	return "", "", fmt.Errorf("group changed during quality evaluation; retrying at next interval")
}
