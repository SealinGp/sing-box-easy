package autoselect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

type healthRuntime struct {
	fakeRuntime
	delays     map[string]int
	callsByTag map[string]int
	err        error
}

func (r *healthRuntime) Delay(_ context.Context, tag, _ string, _ time.Duration) (int, error) {
	r.callsByTag[tag]++
	if r.err != nil {
		return 0, r.err
	}
	if r.delays[tag] == 0 {
		return 0, clashapi.ErrDelayFailed
	}
	return r.delays[tag], nil
}
func TestLiveRecoveryWithoutHistory(t *testing.T) {
	m, _, _, now := managerFixture(t)
	r := &healthRuntime{delays: map[string]int{"b": 120}, callsByTag: map[string]int{}}
	group := clashapi.Proxy{Name: "g", Now: "a", All: []string{"a", "b"}}
	list := []clashapi.Proxy{{Name: "a", Type: "VLESS"}, {Name: "b", Type: "VLESS"}}
	selected, reason, err := m.chooseLive(context.Background(), r, group, list, nil, settings.DefaultNodeQuality(), now, time.Minute)
	if err != nil || selected != "b" || reason != "failover" || r.callsByTag["a"] != 2 {
		t.Fatalf("%s %s %v %+v", selected, reason, err, r.callsByTag)
	}
	r.err = errors.New("controller unreachable")
	if _, _, err = m.chooseLive(context.Background(), r, group, list, nil, settings.DefaultNodeQuality(), now, time.Minute); err == nil {
		t.Fatal("controller failure treated as node failure")
	}
}
func TestOptimizationConfirmationAndMargin(t *testing.T) {
	m, _, _, now := managerFixture(t)
	r := &healthRuntime{delays: map[string]int{"a": 800, "b": 300}, callsByTag: map[string]int{}}
	group := clashapi.Proxy{Name: "g", Now: "a", All: []string{"a", "b"}}
	list := []clashapi.Proxy{{Name: "a", Type: "VLESS"}, {Name: "b", Type: "VLESS"}}
	history := map[string][]nodehistory.Point{"a": samples(now, 100, 100, 800), "b": samples(now, 100, 100, 300)}
	for i, want := range []string{"confirming", "switched"} {
		_, reason, err := m.chooseLive(context.Background(), r, group, list, history, settings.DefaultNodeQuality(), now.Add(time.Duration(i)*time.Minute), 15*time.Minute)
		if err != nil || reason != want {
			t.Fatalf("%s %v", reason, err)
		}
	}
	r.delays["b"] = 780
	m.pending["g"] = "b"
	_, reason, err := m.chooseLive(context.Background(), r, group, list, history, settings.DefaultNodeQuality(), now, 15*time.Minute)
	if err != nil || reason != "no_improvement" {
		t.Fatalf("%s %v", reason, err)
	}
	m.setStatus("g", Status{LastSwitched: now})
	_, reason, err = m.chooseLive(context.Background(), r, group, list, history, settings.DefaultNodeQuality(), now, 15*time.Minute)
	if err != nil || reason != "cooldown" {
		t.Fatalf("%s %v", reason, err)
	}
}
