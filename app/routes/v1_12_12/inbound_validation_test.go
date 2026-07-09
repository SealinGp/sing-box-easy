package v1_13_0

import (
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestValidateInboundVMess(t *testing.T) {
	tests := []struct {
		name    string
		options *option.VMessInboundOptions
		wantErr string
	}{
		{
			name: "missing users",
			options: &option.VMessInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
			},
			wantErr: "users is required",
		},
		{
			name: "missing uuid",
			options: &option.VMessInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
				Users:         []option.VMessUser{{Name: "sekai"}},
			},
			wantErr: "users[0].uuid is required",
		},
		{
			name: "valid",
			options: &option.VMessInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
				Users: []option.VMessUser{{
					Name: "sekai",
					UUID: "bf000d23-0752-40b4-affe-68f7707a9661",
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInbound(option.Inbound{
				Type:    C.TypeVMess,
				Tag:     "vmess-in",
				Options: tt.options,
			})
			assertValidationError(t, err, tt.wantErr)
		})
	}
}

func TestValidateInboundShadowsocks(t *testing.T) {
	tests := []struct {
		name    string
		options *option.ShadowsocksInboundOptions
		wantErr string
	}{
		{
			name: "missing method",
			options: &option.ShadowsocksInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
			},
			wantErr: "method is required",
		},
		{
			name: "missing password",
			options: &option.ShadowsocksInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
				Method:        "aes-128-gcm",
			},
			wantErr: "password is required",
		},
		{
			name: "none method allows empty password",
			options: &option.ShadowsocksInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
				Method:        "none",
			},
		},
		{
			name: "valid",
			options: &option.ShadowsocksInboundOptions{
				ListenOptions: option.ListenOptions{ListenPort: 10000},
				Method:        "aes-128-gcm",
				Password:      "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInbound(option.Inbound{
				Type:    C.TypeShadowsocks,
				Tag:     "ss-in",
				Options: tt.options,
			})
			assertValidationError(t, err, tt.wantErr)
		})
	}
}

func TestValidateInboundRequiresListenPort(t *testing.T) {
	err := validateInbound(option.Inbound{
		Type:    C.TypeVMess,
		Tag:     "vmess-in",
		Options: &option.VMessInboundOptions{},
	})
	assertValidationError(t, err, "listen_port is required")
}

func assertValidationError(t *testing.T, err error, want string) {
	t.Helper()

	if want == "" {
		if err != nil {
			t.Fatalf("validateInbound returned error %q, want nil", err)
		}
		return
	}

	if err == nil {
		t.Fatalf("validateInbound returned nil error, want %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("validateInbound error = %q, want contains %q", err, want)
	}
}
