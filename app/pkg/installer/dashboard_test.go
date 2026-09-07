package installer

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func TestPersistExternalUIPreservesNewerConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX shell script")
	}
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "sing-box")
	script := []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo 'sing-box version 1.14.0'; exit 0; fi\nif [ \"$1\" = check ]; then exit 0; fi\nexit 1\n")
	if err := os.WriteFile(binaryPath, script, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	raw := []byte(`{"dns":{"rules":[{"action":"evaluate","future":true}]},"experimental":{"clash_api":{"external_ui":"/old"},"future_feature":{"value":7}}}`)
	if err := os.WriteFile(configPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	manager := config.NewManager(configPath, binaryPath, "")
	dashboard := NewDashboardManager(nil, manager)
	dashboard.persistExternalUI("/new")

	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{[]byte(`"action": "evaluate"`), []byte(`"future_feature"`), []byte(`"external_ui": "/new"`)} {
		if !bytes.Contains(saved, want) {
			t.Fatalf("updated config lost %s: %s", want, saved)
		}
	}
}

// writeZip builds a zip containing the given "name -> content" entries.
func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "dash.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return path
}

// The shape GitHub serves for archive/<branch>.zip. sing-box expects
// index.html directly inside external_ui, so the wrapper must not survive.
func TestExtractDashboardZipStripsSingleRootDir(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"zashboard-gh-pages/index.html":    "<html>",
		"zashboard-gh-pages/assets/app.js": "console.log(1)",
	})

	target := t.TempDir()
	if err := extractDashboardZip(zipPath, target); err != nil {
		t.Fatalf("extract: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "index.html")); err != nil {
		t.Fatalf("index.html not at the root of external_ui: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "assets", "app.js")); err != nil {
		t.Fatalf("nested asset missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "zashboard-gh-pages")); !os.IsNotExist(err) {
		t.Fatalf("wrapper directory survived extraction")
	}
}

// An archive that is already flat must be left alone — stripping its first path
// element would drop every file in it.
func TestExtractDashboardZipKeepsFlatArchive(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"index.html":    "<html>",
		"assets/app.js": "console.log(1)",
	})

	target := t.TempDir()
	if err := extractDashboardZip(zipPath, target); err != nil {
		t.Fatalf("extract: %v", err)
	}

	for _, want := range []string{"index.html", filepath.Join("assets", "app.js")} {
		if _, err := os.Stat(filepath.Join(target, want)); err != nil {
			t.Fatalf("%s missing: %v", want, err)
		}
	}
}

// Two top-level directories share no wrapper; stripping either would collide.
func TestZipRootDirRequiresASingleDirectory(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"a/index.html": "<html>",
		"b/index.html": "<html>",
	})

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()

	if root := zipRootDir(reader.File); root != "" {
		t.Fatalf("expected no common root, got %q", root)
	}
}

// A stale install from a build that did not strip the wrapper is cleaned up, so
// the operator is not left with two copies of the dashboard.
func TestExtractDashboardZipRemovesStaleWrapperDir(t *testing.T) {
	target := t.TempDir()
	stale := filepath.Join(target, "zashboard-gh-pages")
	if err := os.MkdirAll(stale, 0755); err != nil {
		t.Fatalf("seed stale dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stale, "index.html"), []byte("old"), 0644); err != nil {
		t.Fatalf("seed stale file: %v", err)
	}

	zipPath := writeZip(t, map[string]string{"zashboard-gh-pages/index.html": "new"})
	if err := extractDashboardZip(zipPath, target); err != nil {
		t.Fatalf("extract: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale wrapper directory was not removed")
	}
	content, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil || string(content) != "new" {
		t.Fatalf("fresh index.html missing or stale: %q, %v", content, err)
	}
}

// Zip-slip: an entry escaping the target root must be rejected, not written.
func TestExtractDashboardZipRejectsTraversal(t *testing.T) {
	zipPath := writeZip(t, map[string]string{"../escaped.html": "<html>"})

	target := t.TempDir()
	if err := extractDashboardZip(zipPath, target); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}
