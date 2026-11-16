package singbox

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
	"github.com/sagernet/bbolt"
)

type refreshCore struct{}

func (refreshCore) Validate(context.Context, string) error            { return nil }
func (refreshCore) Version(context.Context) (core.CoreVersion, error) { return core.CoreVersion{}, nil }

type refreshBackend struct {
	Backend
	running       bool
	starts, stops int
	start         func(int) error
}

func (b *refreshBackend) Kind() string                     { return BackendProcess }
func (b *refreshBackend) ActiveAndPID() (bool, int, error) { return b.running, 123, nil }
func (b *refreshBackend) Stop() error                      { b.stops++; b.running = false; return nil }
func (b *refreshBackend) Start() error {
	b.starts++
	if b.start != nil {
		if err := b.start(b.starts); err != nil {
			return err
		}
	}
	b.running = true
	return nil
}

func saveRefreshCache(t *testing.T, path, body string) []byte {
	t.Helper()
	var raw bytes.Buffer
	raw.WriteByte(1)
	var n [10]byte
	size := binary.PutUvarint(n[:], uint64(len(body)))
	raw.Write(n[:size])
	raw.WriteString(body)
	_ = binary.Write(&raw, binary.BigEndian, time.Now().Unix())
	raw.WriteByte(0)
	data := append([]byte(nil), raw.Bytes()...)
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		b, e := tx.CreateBucketIfNotExists([]byte("rule_set"))
		if e != nil {
			return e
		}
		return b.Put([]byte("ai"), data)
	})
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRefreshRuleSetAppliesOrRestoresWithoutChangingConfig(t *testing.T) {
	for _, mode := range []string{"changed", "unchanged", "download failure", "missing cache write"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "cache.db")
			old := saveRefreshCache(t, path, "old content")
			raw, _ := json.Marshal(map[string]any{"route": map[string]any{"rule_set": []any{map[string]any{"tag": "ai", "type": "remote", "url": "https://example.com/ai.json", "update_interval": "1d"}}}, "experimental": map[string]any{"cache_file": map[string]any{"enabled": true, "path": path}}})
			configPath := filepath.Join(dir, "config.json")
			if err := os.WriteFile(configPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			backend := &refreshBackend{running: true}
			backend.start = func(n int) error {
				if n > 1 {
					return nil
				}
				if mode == "download failure" {
					return errors.New("download failed")
				}
				if mode == "missing cache write" {
					return nil
				}
				body := "new content"
				if mode == "unchanged" {
					body = "old content"
				}
				saveRefreshCache(t, path, body)
				return nil
			}
			c := &Controller{backend: backend, configManager: config.NewManagerWithCore(configPath, refreshCore{})}
			// Exercise the coordinator with a short deadline for the missing-write path.
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			result, err := c.refreshRuleSet(ctx, ruleSetRefreshPlan{path: path, tag: "ai"})
			failure := mode == "download failure" || mode == "missing cache write"
			if (err != nil) != failure {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if !backend.running {
				t.Fatal("core left stopped")
			}
			actual, _ := os.ReadFile(configPath)
			if !bytes.Equal(actual, raw) {
				t.Fatal("configuration changed")
			}
			if failure {
				db, _ := bbolt.Open(path, 0600, &bbolt.Options{ReadOnly: true})
				defer db.Close()
				_ = db.View(func(tx *bbolt.Tx) error {
					if !bytes.Equal(tx.Bucket([]byte("rule_set")).Get([]byte("ai")), old) {
						t.Fatal("old cache not restored exactly")
					}
					return nil
				})
				if backend.starts != 2 {
					t.Fatal("recovery did not restart core")
				}
			} else if result.Changed != (mode == "changed") || result.UpdatedAt == 0 {
				t.Fatalf("incorrect change result: %+v", result)
			}
		})
	}
}

func TestRefreshRuleSetRejectsConcurrentLifecycle(t *testing.T) {
	c := &Controller{}
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	if _, err := c.RefreshRuleSet(context.Background(), "ai"); err == nil {
		t.Fatal("concurrent operation accepted")
	}
}

func TestRefreshPlanRejectsLocalMissingAndDisabledBeforeStopping(t *testing.T) {
	for _, raw := range []string{
		`{"route":{"rule_set":[{"tag":"ai","type":"local"}]}}`,
		`{"route":{"rule_set":[]}}`,
		`{"route":{"rule_set":[{"tag":"ai","type":"remote"}]}}`,
		`{"route":{"rule_set":[{"tag":"ai","type":"remote","initial_path":"init.json"}]}}`,
	} {
		if _, err := planRuleSetRefresh([]byte(raw), "ai", 0, true); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
