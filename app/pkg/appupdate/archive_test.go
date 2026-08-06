package appupdate

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tarEntry is a single file/dir to write into a test archive.
type tarEntry struct {
	name string
	body string
	dir  bool
	mode int64
}

func writeTestArchive(t *testing.T, entries []tarEntry) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "package.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.body))}
		if hdr.Mode == 0 {
			hdr.Mode = 0o644
		}
		if e.dir {
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		} else {
			hdr.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if !e.dir {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write body %q: %v", e.name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return path
}

func TestExtractTarGzExtractsReleaseLayout(t *testing.T) {
	archive := writeTestArchive(t, []tarEntry{
		{name: "./", dir: true},
		{name: "./sing-box-easy", body: "binary-bytes", mode: 0o755},
		{name: "./dist", dir: true},
		{name: "./dist/index.html", body: "<html></html>"},
		{name: "./dist/assets/js/app.js", body: "console.log(1)"},
		{name: "./app.yml.example", body: "server:\n"},
	})

	target := t.TempDir()
	if err := extractTarGz(archive, target); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	for path, want := range map[string]string{
		"sing-box-easy":         "binary-bytes",
		"dist/index.html":       "<html></html>",
		"dist/assets/js/app.js": "console.log(1)",
		"app.yml.example":       "server:\n",
	} {
		got, err := os.ReadFile(filepath.Join(target, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}

	info, err := os.Stat(filepath.Join(target, "sing-box-easy"))
	if err != nil {
		t.Fatalf("stat binary: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binary mode = %v, want the owner-execute bit set", info.Mode().Perm())
	}
}

func TestExtractTarGzRejectsPathTraversal(t *testing.T) {
	for _, name := range []string{"../escaped.txt", "dist/../../escaped.txt", "/etc/passwd"} {
		t.Run(name, func(t *testing.T) {
			archive := writeTestArchive(t, []tarEntry{{name: name, body: "pwned"}})

			target := t.TempDir()
			err := extractTarGz(archive, target)
			if err == nil {
				t.Fatalf("extractTarGz(%q) succeeded, want a path-traversal rejection", name)
			}
			if !strings.Contains(err.Error(), "illegal path") {
				t.Errorf("error = %v, want an 'illegal path' rejection", err)
			}
		})
	}
}

func TestExtractTarGzSkipsSymlinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "link.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "evil-link",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("write symlink header: %v", err)
	}
	tw.Close()
	gz.Close()
	f.Close()

	target := t.TempDir()
	if err := extractTarGz(path, target); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(target, "evil-link")); !os.IsNotExist(err) {
		t.Errorf("symlink was extracted; want it skipped (lstat err = %v)", err)
	}
}

func TestSafeJoin(t *testing.T) {
	root := filepath.Clean(t.TempDir())

	if got, err := safeJoin(root, "./"); err != nil || got != root {
		t.Errorf(`safeJoin(root, "./") = %q, %v; want %q, nil`, got, err, root)
	}
	if got, err := safeJoin(root, "dist/index.html"); err != nil || got != filepath.Join(root, "dist/index.html") {
		t.Errorf("safeJoin nested = %q, %v; want the joined path", got, err)
	}
	if _, err := safeJoin(root, "../outside"); err == nil {
		t.Error("safeJoin(root, \"../outside\") succeeded, want rejection")
	}
}
