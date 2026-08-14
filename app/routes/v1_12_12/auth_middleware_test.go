package v1_13_0

import (
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/appconfig"
	"github.com/SealinGp/sing-box-easy/app/pkg/service"
)

func TestResolveAuthEnabled(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		systemType service.SystemType
		want       bool
	}{
		{"auto on debian requires login", appconfig.AuthAuto, service.SystemDebian, true},
		{"auto on unknown requires login", appconfig.AuthAuto, service.SystemUnknown, true},
		{"auto on openwrt skips login", appconfig.AuthAuto, service.SystemOpenWRT, false},
		{"enabled wins on openwrt", appconfig.AuthEnabled, service.SystemOpenWRT, true},
		{"disabled wins on debian", appconfig.AuthDisabled, service.SystemDebian, false},
		{"unexpected mode fails safe to enabled", "banana", service.SystemOpenWRT, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveAuthEnabled(tt.mode, tt.systemType); got != tt.want {
				t.Errorf("ResolveAuthEnabled(%q, %s) = %v, want %v", tt.mode, tt.systemType, got, tt.want)
			}
		})
	}
}
