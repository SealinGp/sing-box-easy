package installer

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

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
