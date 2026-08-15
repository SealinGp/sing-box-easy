package appupdate

import (
	"os"
	"path/filepath"
	"testing"
)

// opkg records no provenance: a package pulled from a feed and one installed
// from a downloaded .ipk produce byte-identical stanzas. So the panel cannot
// ask "how was this installed" — it asks "can a configured feed upgrade it",
// which these tests pin down.

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInspectOpkg(t *testing.T) {
	// Verbatim shape from an ImmortalWrt 23.05.1 box.
	const realistic = `Package: dnsmasq-full
Version: 2.90-2
Status: install user installed
Architecture: x86_64
Installed-Time: 1772940607

Package: sing-box-easy
Version: 1.2.4
Depends: libc
Status: install user installed
Architecture: x86_64
Installed-Time: 1773000000

Package: luci-app-passwall
Version: 25.8.5-1
Status: install user installed
Architecture: all
`

	t.Run("reads architecture and version from our own stanza", func(t *testing.T) {
		path := writeTestFile(t, t.TempDir(), "status", realistic)
		got := inspectOpkg(path)

		if !got.Managed {
			t.Error("Managed = false, want true")
		}
		if got.Architecture != "x86_64" {
			t.Errorf("Architecture = %q, want %q", got.Architecture, "x86_64")
		}
		if got.Version != "1.2.4" {
			t.Errorf("Version = %q, want %q", got.Version, "1.2.4")
		}
	})

	t.Run("neighbouring stanza fields do not leak", func(t *testing.T) {
		// If stanza state is not reset at the boundary, sing-box-easy would
		// inherit "all" from the package declared after it.
		const leaky = `Package: sing-box-easy
Version: 1.2.4
Status: install user installed
Architecture: x86_64

Package: luci-app-something
Architecture: all
Status: install user installed
`
		got := inspectOpkg(writeTestFile(t, t.TempDir(), "status", leaky))
		if got.Architecture != "x86_64" {
			t.Errorf("Architecture = %q, want x86_64", got.Architecture)
		}
	})

	t.Run("not-installed stanza is not managed", func(t *testing.T) {
		const removed = `Package: sing-box-easy
Version: 1.2.4
Status: install prefer,user not-installed
Architecture: x86_64
`
		got := inspectOpkg(writeTestFile(t, t.TempDir(), "status", removed))
		if got.Managed {
			t.Error("Managed = true for a not-installed stanza")
		}
	})

	t.Run("duplicate stanzas prefer the installed one", func(t *testing.T) {
		// Real ImmortalWrt boxes carry these: an installed stanza plus a
		// leftover not-installed one for the same package.
		const dup = `Package: sing-box-easy
Status: install user installed
Architecture: x86_64

Package: sing-box-easy
Version: release-2024031600
Status: install prefer,user not-installed
Architecture: all
`
		got := inspectOpkg(writeTestFile(t, t.TempDir(), "status", dup))
		if !got.Managed {
			t.Fatal("Managed = false, want true")
		}
		if got.Architecture != "x86_64" {
			t.Errorf("Architecture = %q, want x86_64 (the installed stanza)", got.Architecture)
		}
	})

	t.Run("missing file is not managed", func(t *testing.T) {
		got := inspectOpkg(filepath.Join(t.TempDir(), "absent"))
		if got.Managed || got.Architecture != "" {
			t.Errorf("inspectOpkg(missing) = %+v, want zero value", got)
		}
	})
}

func TestOpkgListsDir(t *testing.T) {
	tests := []struct {
		name string
		conf string
		want string
	}{
		{
			name: "explicit lists_dir",
			conf: "dest root /\ndest ram /tmp\nlists_dir ext /var/opkg-lists\noption overlay_root /overlay\n",
			want: "/var/opkg-lists",
		},
		{
			name: "extra whitespace",
			conf: "  lists_dir   ext    /custom/lists  \n",
			want: "/custom/lists",
		},
		{
			name: "no lists_dir falls back to the opkg default",
			conf: "dest root /\n",
			want: defaultOpkgListsDir,
		},
		{
			name: "missing file falls back to the opkg default",
			conf: "",
			want: defaultOpkgListsDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "opkg.conf")
			if tt.conf != "" {
				writeTestFile(t, filepath.Dir(path), "opkg.conf", tt.conf)
			}
			if got := opkgListsDir(path); got != tt.want {
				t.Errorf("opkgListsDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFeedProvides(t *testing.T) {
	t.Run("empty lists dir means unknown, not absent", func(t *testing.T) {
		// /var/opkg-lists is tmpfs and starts empty every boot. Reporting
		// "no feed has it" from an unpopulated cache would be a lie.
		provides, known := feedProvides(t.TempDir(), "sing-box-easy")
		if known {
			t.Error("known = true for an empty lists dir, want false")
		}
		if provides {
			t.Error("provides = true for an empty lists dir")
		}
	})

	t.Run("missing lists dir means unknown", func(t *testing.T) {
		_, known := feedProvides(filepath.Join(t.TempDir(), "absent"), "sing-box-easy")
		if known {
			t.Error("known = true for a missing lists dir")
		}
	})

	t.Run("package present in a feed index", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, dir, "immortalwrt_core", "Package: busybox\nVersion: 1.36\n\n")
		writeTestFile(t, dir, "custom_feed", "Package: sing-box-easy\nVersion: 1.3.0\nArchitecture: x86_64\n\n")

		provides, known := feedProvides(dir, "sing-box-easy")
		if !known {
			t.Fatal("known = false, want true")
		}
		if !provides {
			t.Error("provides = false, want true")
		}
	})

	t.Run("populated feeds without the package", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, dir, "immortalwrt_core", "Package: busybox\nVersion: 1.36\n\n")

		provides, known := feedProvides(dir, "sing-box-easy")
		if !known {
			t.Fatal("known = false, want true")
		}
		if provides {
			t.Error("provides = true, want false")
		}
	})

	t.Run("prefix collisions do not match", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, dir, "feed", "Package: sing-box-easy-extras\nPackage: sing-box\n")

		if provides, _ := feedProvides(dir, "sing-box-easy"); provides {
			t.Error("provides = true for sing-box / sing-box-easy-extras")
		}
	})

	t.Run("subdirectories are ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeTestFile(t, dir, "sub/nested", "Package: sing-box-easy\n")

		provides, known := feedProvides(dir, "sing-box-easy")
		if provides {
			t.Error("provides = true from a nested file")
		}
		if known {
			t.Error("known = true when only subdirectories exist")
		}
	})
}

func TestIpkAssetName(t *testing.T) {
	tests := []struct {
		tag, arch, want string
	}{
		// The release workflow strips the leading "v" for the ipk filename.
		{"v1.2.4", "x86_64", "sing-box-easy_1.2.4_x86_64.ipk"},
		{"1.2.4", "x86_64", "sing-box-easy_1.2.4_x86_64.ipk"},
		{"v1.3.0", "aarch64_generic", "sing-box-easy_1.3.0_aarch64_generic.ipk"},
		{"v2.0.0-rc.1", "arm_cortex-a7", "sing-box-easy_2.0.0-rc.1_arm_cortex-a7.ipk"},
	}

	for _, tt := range tests {
		if got := IpkAssetName(tt.tag, tt.arch); got != tt.want {
			t.Errorf("IpkAssetName(%q, %q) = %q, want %q", tt.tag, tt.arch, got, tt.want)
		}
	}
}

func TestIpkAssetURL(t *testing.T) {
	got := IpkAssetURL("v1.2.4", "x86_64")
	want := releaseDownloadBase + "/v1.2.4/sing-box-easy_1.2.4_x86_64.ipk"
	if got != want {
		t.Errorf("IpkAssetURL() = %q, want %q", got, want)
	}
	if IpkChecksumURL("v1.2.4", "x86_64") != want+".sha256" {
		t.Errorf("IpkChecksumURL() = %q, want %q", IpkChecksumURL("v1.2.4", "x86_64"), want+".sha256")
	}
}
