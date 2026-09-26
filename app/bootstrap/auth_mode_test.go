package bootstrap

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/platform/sysinfo"
)

func TestResolveAuthEnabled(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		systemType sysinfo.SystemType
		want       bool
	}{
		{"auto on debian requires login", appconfig.AuthAuto, sysinfo.SystemDebian, true},
		{"auto on unknown requires login", appconfig.AuthAuto, sysinfo.SystemUnknown, true},
		{"auto on openwrt skips login", appconfig.AuthAuto, sysinfo.SystemOpenWRT, false},
		{"enabled wins on openwrt", appconfig.AuthEnabled, sysinfo.SystemOpenWRT, true},
		{"disabled wins on debian", appconfig.AuthDisabled, sysinfo.SystemDebian, false},
		{"unexpected mode fails safe to enabled", "banana", sysinfo.SystemOpenWRT, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAuthEnabled(tt.mode, tt.systemType); got != tt.want {
				t.Errorf("resolveAuthEnabled(%q, %s) = %v, want %v", tt.mode, tt.systemType, got, tt.want)
			}
		})
	}
}
