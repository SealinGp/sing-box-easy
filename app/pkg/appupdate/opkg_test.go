package appupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledViaOpkg(t *testing.T) {
	writeStatus := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "status")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("missing status file means not opkg-managed", func(t *testing.T) {
		if installedViaOpkg(filepath.Join(t.TempDir(), "nope")) {
			t.Error("installedViaOpkg() = true for a missing status file")
		}
	})

	t.Run("package installed", func(t *testing.T) {
		path := writeStatus(t, "Package: busybox\nStatus: install ok installed\n\n"+
			"Package: sing-box-easy\nVersion: 1.4.0\nStatus: install ok installed\n")
		if !installedViaOpkg(path) {
			t.Error("installedViaOpkg() = false, want true")
		}
	})

	t.Run("similarly named package does not match", func(t *testing.T) {
		path := writeStatus(t, "Package: sing-box\nStatus: install ok installed\n\n"+
			"Package: sing-box-easy-extras\nStatus: install ok installed\n")
		if installedViaOpkg(path) {
			t.Error("installedViaOpkg() = true for sing-box / sing-box-easy-extras")
		}
	})

	t.Run("leftover deinstall stanza does not count as installed", func(t *testing.T) {
		path := writeStatus(t, "Package: sing-box-easy\nVersion: 1.3.0\n"+
			"Status: deinstall ok not-installed\n")
		if installedViaOpkg(path) {
			t.Error("installedViaOpkg() = true for a not-installed stanza")
		}
	})

	t.Run("installed status in a different stanza does not leak", func(t *testing.T) {
		path := writeStatus(t, "Package: busybox\nStatus: install ok installed\n\n"+
			"Package: sing-box-easy\nStatus: deinstall ok not-installed\n")
		if installedViaOpkg(path) {
			t.Error("installedViaOpkg() = true, Status from another stanza leaked")
		}
	})
}
