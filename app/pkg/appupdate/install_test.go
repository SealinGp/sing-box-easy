package appupdate

import (
	"os"
	"path/filepath"
	"testing"
)

// stagedRelease lays out an install directory (with the "running" binary and
// the current frontend) plus an extracted package, and chdirs into the install
// dir so installRelease resolves ./dist the way the server does at runtime.
type stagedRelease struct {
	installDir string
	extractDir string
	liveBinary string
	backupPath string
	liveIndex  string
}

func stageRelease(t *testing.T, pkg map[string]string) stagedRelease {
	t.Helper()

	installDir := t.TempDir()
	t.Chdir(installDir)

	binaryName := currentBinaryName()
	liveBinary := filepath.Join(installDir, binaryName)

	writeFile(t, liveBinary, "old-binary", 0o755)
	writeFile(t, filepath.Join(installDir, "dist", "index.html"), "<old/>", 0o644)

	extractDir := filepath.Join(installDir, ".update-staging", "extract")
	for rel, body := range pkg {
		mode := os.FileMode(0o644)
		if rel == "sing-box-easy" || rel == binaryName {
			mode = 0o755
		}
		writeFile(t, filepath.Join(extractDir, rel), body, mode)
	}

	return stagedRelease{
		installDir: installDir,
		extractDir: extractDir,
		liveBinary: liveBinary,
		backupPath: liveBinary + ".old",
		liveIndex:  filepath.Join(installDir, "dist", "index.html"),
	}
}

func writeFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestInstallReleaseSwapsBinaryAndFrontend(t *testing.T) {
	staged := stageRelease(t, map[string]string{
		"sing-box-easy":         "new-binary",
		"dist/index.html":       "<new/>",
		"dist/assets/js/app.js": "new-js",
		"app.yml.example":       "server:\n",
	})

	if err := installRelease(staged.installDir, staged.extractDir); err != nil {
		t.Fatalf("installRelease: %v", err)
	}

	if got := readFile(t, staged.liveBinary); got != "new-binary" {
		t.Errorf("live binary = %q, want %q", got, "new-binary")
	}
	if got := readFile(t, staged.backupPath); got != "old-binary" {
		t.Errorf("backup = %q, want the previous binary preserved for rollback", got)
	}
	if got := readFile(t, staged.liveIndex); got != "<new/>" {
		t.Errorf("dist/index.html = %q, want the new bundle", got)
	}
	if got := readFile(t, filepath.Join(staged.installDir, "dist", "assets", "js", "app.js")); got != "new-js" {
		t.Errorf("nested asset = %q, want the new bundle", got)
	}

	info, err := os.Stat(staged.liveBinary)
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("binary mode = %v, want it executable", info.Mode().Perm())
	}

	// The stale ".old" bundle must not be left behind.
	if _, err := os.Stat(filepath.Join(staged.installDir, "dist.old")); !os.IsNotExist(err) {
		t.Errorf("dist.old still present after a successful swap (stat err = %v)", err)
	}
}

// Binary-only packages (frontend embedded via go:embed) are valid: the binary
// is swapped and any pre-existing on-disk dist is left untouched.
func TestInstallReleaseAcceptsBinaryOnlyPackage(t *testing.T) {
	staged := stageRelease(t, map[string]string{"sing-box-easy": "new-binary"})

	if err := installRelease(staged.installDir, staged.extractDir); err != nil {
		t.Fatalf("installRelease with a binary-only package failed: %v", err)
	}

	if got := readFile(t, staged.liveBinary); got != "new-binary" {
		t.Errorf("live binary = %q, want the new binary installed", got)
	}
	if got := readFile(t, staged.liveIndex); got != "<old/>" {
		t.Errorf("dist/index.html = %q, want the existing bundle left in place", got)
	}
}

func TestInstallReleaseRejectsPackageWithoutBinary(t *testing.T) {
	staged := stageRelease(t, map[string]string{"dist/index.html": "<new/>"})

	if err := installRelease(staged.installDir, staged.extractDir); err == nil {
		t.Fatal("installRelease succeeded without a binary, want an error")
	}
	if got := readFile(t, staged.liveBinary); got != "old-binary" {
		t.Errorf("live binary = %q, want the original left in place", got)
	}
}

func TestInstallReleaseRejectsIncompleteFrontend(t *testing.T) {
	// dist/ exists but has no index.html — the server would serve 404s.
	staged := stageRelease(t, map[string]string{
		"sing-box-easy":  "new-binary",
		"dist/stray.txt": "nothing useful",
	})

	if err := installRelease(staged.installDir, staged.extractDir); err == nil {
		t.Fatal("installRelease accepted a dist/ without index.html, want an error")
	}
	if got := readFile(t, staged.liveBinary); got != "old-binary" {
		t.Errorf("live binary = %q, want the original left in place", got)
	}
}

func TestInstallReleaseRejectsEmptyBinary(t *testing.T) {
	staged := stageRelease(t, map[string]string{
		"sing-box-easy":   "",
		"dist/index.html": "<new/>",
	})

	if err := installRelease(staged.installDir, staged.extractDir); err == nil {
		t.Fatal("installRelease accepted a zero-byte binary, want an error")
	}
	if got := readFile(t, staged.liveBinary); got != "old-binary" {
		t.Errorf("live binary = %q, want the original left in place", got)
	}
}

func TestWriteAndReadVersionFile(t *testing.T) {
	dir := t.TempDir()

	if err := writeVersionFile(dir, "v1.2.3"); err != nil {
		t.Fatalf("writeVersionFile: %v", err)
	}

	got := readFile(t, filepath.Join(dir, VersionFileName))
	if got != "v1.2.3\n" {
		t.Errorf("version file = %q, want %q", got, "v1.2.3\n")
	}
}
