package installer

import (
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/service"
)

func TestBuildInstallCommandLinux(t *testing.T) {
	tests := []struct {
		name    string
		version string
		beta    bool
		want    string
	}{
		{name: "latest", want: "https://sing-box.app/install.sh | sh"},
		{name: "beta", beta: true, want: "install.sh | sh -s -- --beta"},
		{name: "pinned", version: "1.12.12", want: "install.sh | sh -s -- --version 1.12.12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildInstallCommand(service.SystemDebian, tt.version, tt.beta)
			if err != nil {
				t.Fatalf("buildInstallCommand() error = %v", err)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("buildInstallCommand() = %q, want it to contain %q", got, tt.want)
			}
			if !strings.HasPrefix(got, "curl ") {
				t.Errorf("linux install must go through curl, got %q", got)
			}
		})
	}
}

func TestBuildInstallCommandOpenwrt(t *testing.T) {
	t.Run("latest uses opkg", func(t *testing.T) {
		got, err := buildInstallCommand(service.SystemOpenWRT, "", false)
		if err != nil {
			t.Fatalf("buildInstallCommand() error = %v", err)
		}
		if got != "opkg update && opkg install sing-box" {
			t.Errorf("buildInstallCommand() = %q", got)
		}
	})

	t.Run("pinned version downloads release tarball via wget", func(t *testing.T) {
		got, err := buildInstallCommand(service.SystemOpenWRT, "1.12.12", false)
		if err != nil {
			t.Fatalf("buildInstallCommand() error = %v", err)
		}
		for _, want := range []string{
			"wget -qO /tmp/sing-box.tar.gz",
			"github.com/SagerNet/sing-box/releases/download/v1.12.12/",
			"install -m 755",
			"/usr/bin/sing-box",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("buildInstallCommand() = %q, want it to contain %q", got, want)
			}
		}
		if strings.Contains(got, "curl") {
			t.Errorf("OpenWrt install must not depend on curl, got %q", got)
		}
	})

	t.Run("beta is rejected", func(t *testing.T) {
		if _, err := buildInstallCommand(service.SystemOpenWRT, "", true); err == nil {
			t.Error("buildInstallCommand() with beta on OpenWrt should error")
		}
	})
}
