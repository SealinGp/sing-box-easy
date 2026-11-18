package probe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/proxies"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"xorm.io/xorm"
)

type historyRuntimeConfig struct{ controller string }

func (c historyRuntimeConfig) GetClashAPISettings() (config.ClashAPISettings, error) {
	return config.ClashAPISettings{ExternalController: c.controller}, nil
}
func (c historyRuntimeConfig) GetConfigSection(string) (json.RawMessage, bool, error) {
	return nil, false, nil
}

func TestAutomaticAndManualProbesShareDurableNodeHistory(t *testing.T) {
	e, err := xorm.NewEngine("sqlite", filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	e.SetMaxOpenConns(1)
	defer e.Close()
	history := nodehistory.NewStore(e, func() (time.Duration, int) { return 7 * 24 * time.Hour, 2016 })
	if err := history.Init(); err != nil {
		t.Fatal(err)
	}
	const node = "🇰🇷Korea 01 3b7d7 | sub_a"
	const down = "down | sub_a"
	const missing = "missing | sub_a"
	aggregates := newStore(t)
	env := &fakeEnv{
		targets: []Target{{SubID: "sub_a", URL: clashapi.DefaultDelayURL}},
		tags:    []string{node, down, missing}, settings: defaultSettings(),
		prober: &fakeProber{delays: map[string]int{node: 254}, errs: map[string]error{down: clashapi.ErrDelayFailed, missing: clashapi.ErrProxyNotFound}},
	}
	runner := NewRunner(aggregates, env, history)
	for range 2 {
		if err := runner.RunOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if points, err := history.Series(node); err != nil || len(points) != 2 || points[0].Delay != 254 || points[0].Source != "subscription" {
		t.Fatalf("automatic node history=%+v err=%v", points, err)
	}
	if points, _ := history.Series(down); len(points) != 2 || points[0].Success {
		t.Fatalf("timeouts missing: %+v", points)
	}
	if points, _ := history.Series(missing); len(points) != 0 {
		t.Fatalf("untestable node counted: %+v", points)
	}
	if n, _ := aggregates.CountAll(); n != 2 {
		t.Fatalf("subscription aggregate count=%d", n)
	}

	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/delay") {
			if fail.Load() {
				w.WriteHeader(504)
				return
			}
			_, _ = w.Write([]byte(`{"delay":187}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"proxies": map[string]any{
			"group": map[string]any{"type": "Selector", "all": []string{node}, "now": node},
			node:    map[string]any{"type": "VLESS", "history": []map[string]any{{"time": time.Now(), "delay": 187}}},
		}})
	}))
	defer server.Close()
	service := proxies.NewService(historyRuntimeConfig{server.URL}, history)
	for range 2 {
		if _, err := service.TestNode(context.Background(), proxies.TestCommand{Name: node}); err != nil {
			t.Fatal(err)
		}
	}
	fail.Store(true)
	if _, err := service.TestNode(context.Background(), proxies.TestCommand{Name: node}); err != nil {
		t.Fatal(err)
	}
	// Reconstruct the service, just like a later request / process restart.
	service = proxies.NewService(historyRuntimeConfig{server.URL}, history)
	view, err := service.View(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := view.Groups[0].Members[0].History
	if len(got) != 5 || got[4].Success || got[4].Source != "manual" {
		t.Fatalf("stored history replaced by Clash's one point: %+v", got)
	}
	if n, _ := aggregates.CountAll(); n != 2 {
		t.Fatalf("single-node clicks polluted subscription aggregates: %d", n)
	}
	fail.Store(false)
	if _, err := service.TestGroup(context.Background(), proxies.TestCommand{Name: "group"}); err != nil {
		t.Fatal(err)
	}
	full, err := service.NodeHistory(node)
	if err != nil || len(full) != 6 || full[5].Source != "manual-group" {
		t.Fatalf("group history=%+v err=%v", full, err)
	}
	if n, _ := aggregates.CountAll(); n != 2 {
		t.Fatalf("arbitrary group polluted subscription aggregates: %d", n)
	}
	before, _ := history.Count()
	broken := errors.New("controller connection refused")
	env.prober = &fakeProber{errs: map[string]error{node: broken, down: broken, missing: broken}}
	if err := runner.RunOnce(context.Background()); !errors.Is(err, broken) {
		t.Fatalf("controller failure=%v", err)
	}
	if after, _ := history.Count(); after != before {
		t.Fatal("controller outage recorded as failed nodes")
	}
	if n, _ := aggregates.CountAll(); n != 2 {
		t.Fatal("controller outage recorded as provider outage")
	}
}
