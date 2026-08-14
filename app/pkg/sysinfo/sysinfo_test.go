package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseDistribution(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "openwrt pretty name",
			content: `NAME="OpenWrt"
VERSION="23.05.2"
ID="openwrt"
PRETTY_NAME="OpenWrt 23.05.2"
`,
			want: "OpenWrt 23.05.2",
		},
		{
			name: "debian pretty name",
			content: `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
`,
			want: "Debian GNU/Linux 12 (bookworm)",
		},
		{
			name: "falls back to name and version",
			content: `NAME="Alpine Linux"
VERSION="3.19"
`,
			want: "Alpine Linux 3.19",
		},
		{
			name:    "falls back to name alone",
			content: "NAME=Gentoo\n",
			want:    "Gentoo",
		},
		{
			name:    "single quoted values are trimmed",
			content: "PRETTY_NAME='Some Distro 1.0'\n",
			want:    "Some Distro 1.0",
		},
		{
			name:    "comments and blank lines are ignored",
			content: "# a comment\n\nPRETTY_NAME=\"Router OS\"\n",
			want:    "Router OS",
		},
		{
			name:    "empty file yields empty name",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDistribution(tt.content); got != tt.want {
				t.Errorf("parseDistribution() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectReadsFiles(t *testing.T) {
	dir := t.TempDir()
	osRelease := filepath.Join(dir, "os-release")
	kernel := filepath.Join(dir, "osrelease")

	if err := os.WriteFile(osRelease, []byte("PRETTY_NAME=\"OpenWrt 23.05.2\"\n"), 0o600); err != nil {
		t.Fatalf("write os-release: %v", err)
	}
	if err := os.WriteFile(kernel, []byte("5.15.137\n"), 0o600); err != nil {
		t.Fatalf("write kernel file: %v", err)
	}

	info := collect(osRelease, kernel)

	if info.Distribution != "OpenWrt 23.05.2" {
		t.Errorf("Distribution = %q, want %q", info.Distribution, "OpenWrt 23.05.2")
	}
	if info.Kernel != "5.15.137" {
		t.Errorf("Kernel = %q, want %q", info.Kernel, "5.15.137")
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.CPUCores < 1 {
		t.Errorf("CPUCores = %d, want >= 1", info.CPUCores)
	}
}

// Missing files are the norm on macOS dev machines, so this must not panic or
// report bogus data.
func TestCollectToleratesMissingFiles(t *testing.T) {
	dir := t.TempDir()

	info := collect(filepath.Join(dir, "absent"), filepath.Join(dir, "absent-too"))

	if info.Distribution != "" {
		t.Errorf("Distribution = %q, want empty", info.Distribution)
	}
	if info.Kernel != "" {
		t.Errorf("Kernel = %q, want empty", info.Kernel)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
}
