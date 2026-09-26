package bootstrap

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox"
)

func TestResolveAuthEnabled(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		systemType singbox.SystemType
		want       bool
	}{
		{"auto on debian requires login", appconfig.AuthAuto, singbox.SystemDebian, true},
		{"auto on unknown requires login", appconfig.AuthAuto, singbox.SystemUnknown, true},
		{"auto on openwrt skips login", appconfig.AuthAuto, singbox.SystemOpenWRT, false},
		{"enabled wins on openwrt", appconfig.AuthEnabled, singbox.SystemOpenWRT, true},
		{"disabled wins on debian", appconfig.AuthDisabled, singbox.SystemDebian, false},
		{"unexpected mode fails safe to enabled", "banana", singbox.SystemOpenWRT, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAuthEnabled(tt.mode, tt.systemType); got != tt.want {
				t.Errorf("resolveAuthEnabled(%q, %s) = %v, want %v", tt.mode, tt.systemType, got, tt.want)
			}
		})
	}
}
