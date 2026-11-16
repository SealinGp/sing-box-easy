package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

// Opt-in real-core exercise, confined to a temporary config/cache and a local
// HTTP server. It never uses the host service or any TUN/listening inbound.
func TestRefreshWithInstalledCore(t *testing.T) {
	binary := os.Getenv("SBE_TEST_SINGBOX")
	if binary == "" {
		t.Skip("set SBE_TEST_SINGBOX to test the installed core")
	}
	var version atomic.Int32
	version.Store(1)
	var otherDownloads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/other.json" {
			otherDownloads.Add(1)
			fmt.Fprint(w, `{"version":3,"rules":[{"domain_suffix":"other.test"}]}`)
			return
		}
		if version.Load() == 3 {
			http.Error(w, "download unavailable", 503)
			return
		}
		fmt.Fprintf(w, `{"version":3,"rules":[{"domain_suffix":"version%d.test"}]}`, version.Load())
	}))
	defer server.Close()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cachePath := filepath.Join(dir, "cache.db")
	raw, _ := json.Marshal(map[string]any{
		"log":       map[string]any{"level": "error"},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
		"route": map[string]any{"final": "direct", "rule_set": []any{
			map[string]any{"type": "remote", "tag": "ai", "url": server.URL + "/ai.json", "format": "source", "download_detour": "direct", "update_interval": "1d"},
			map[string]any{"type": "remote", "tag": "other", "url": server.URL + "/other.json", "format": "source", "download_detour": "direct", "update_interval": "1d"},
		}},
		"experimental": map[string]any{"cache_file": map[string]any{"enabled": true, "path": cachePath, "cache_id": "refresh-test"}},
	})
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	backend := &installedRefreshBackend{binary: binary, path: configPath}
	if err := backend.Start(); err != nil {
		t.Fatal(err)
	}
	defer backend.Stop()
	controller := &Controller{backend: backend, configManager: config.NewManager(configPath, binary, "")}
	version.Store(2)
	result, err := controller.RefreshRuleSet(context.Background(), "ai")
	if err != nil || !result.Changed {
		t.Fatalf("refresh=%+v error=%v", result, err)
	}
	check := func() {
		cache, err := ruleset.OpenCacheFile(cachePath, "\x00refresh-test")
		if err != nil {
			t.Fatal(err)
		}
		defer cache.Close()
		saved := cache.LoadRuleSet("ai")
		if saved == nil {
			t.Fatal("missing refreshed content")
		}
		var content struct {
			Rules []struct {
				Suffix string `json:"domain_suffix"`
			} `json:"rules"`
		}
		if err = json.Unmarshal(saved.Content, &content); err != nil {
			t.Fatal(err)
		}
		if len(content.Rules) != 1 || content.Rules[0].Suffix != "version2.test" {
			t.Fatalf("wrong content %s", saved.Content)
		}
	}
	check()
	version.Store(3)
	if _, err = controller.RefreshRuleSet(context.Background(), "ai"); err == nil {
		t.Fatal("download failure reported success")
	}
	check()
	if running, _, _ := backend.ActiveAndPID(); !running {
		t.Fatal("rollback did not restore running core")
	}
	if otherDownloads.Load() != 1 {
		t.Fatalf("unrelated rule set downloaded %d times", otherDownloads.Load())
	}
	current, _ := os.ReadFile(configPath)
	if string(current) != string(raw) {
		t.Fatal("configuration changed")
	}
}

type installedRefreshBackend struct {
	Backend
	binary, path string
	cmd          *exec.Cmd
	done         chan error
}

func (b *installedRefreshBackend) Kind() string { return BackendProcess }
func (b *installedRefreshBackend) Start() error {
	b.cmd = exec.Command(b.binary, "run", "-c", b.path)
	b.cmd.Env = append(os.Environ(), "ENABLE_DEPRECATED_SPECIAL_OUTBOUNDS=true")
	if err := b.cmd.Start(); err != nil {
		return err
	}
	b.done = make(chan error, 1)
	cmd := b.cmd
	done := b.done
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-b.done:
		b.cmd = nil
		return fmt.Errorf("core exited: %v", err)
	case <-time.After(800 * time.Millisecond):
		return nil
	}
}
func (b *installedRefreshBackend) Stop() error {
	if b.cmd == nil {
		return nil
	}
	_ = b.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-b.done:
	case <-time.After(3 * time.Second):
		_ = b.cmd.Process.Kill()
		<-b.done
	}
	b.cmd = nil
	return nil
}
func (b *installedRefreshBackend) ActiveAndPID() (bool, int, error) {
	if b.cmd == nil {
		return false, 0, nil
	}
	select {
	case <-b.done:
		b.cmd = nil
		return false, 0, nil
	default:
		return true, b.cmd.Process.Pid, nil
	}
}
