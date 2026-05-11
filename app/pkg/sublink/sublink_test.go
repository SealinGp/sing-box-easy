package sublink

import "testing"

func TestValidateSubscriptionURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Allowed
		{"https public host", "https://example.com/sub", false},
		{"http public host", "http://example.com/sub", false},
		{"https with port", "https://example.com:8443/sub", false},
		{"https public ip", "https://93.184.216.34/sub", false},

		// Schemes
		{"empty rejected", "", true},
		{"ftp rejected", "ftp://example.com/sub", true},
		{"file rejected", "file:///etc/passwd", true},
		{"gopher rejected", "gopher://example.com/", true},
		{"no scheme rejected", "example.com/sub", true},

		// Loopback / private / link-local hostnames and IPs
		{"localhost rejected", "http://localhost/", true},
		{"127.0.0.1 rejected", "http://127.0.0.1/", true},
		{"::1 rejected", "http://[::1]/", true},
		{"private 10.x rejected", "http://10.0.0.1/", true},
		{"private 192.168.x rejected", "http://192.168.1.1/", true},
		{"private 172.16.x rejected", "http://172.16.0.1/", true},
		{"link-local 169.254 rejected", "http://169.254.169.254/", true},
		{"unspecified 0.0.0.0 rejected", "http://0.0.0.0/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSubscriptionURL(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("validateSubscriptionURL(%q) expected error, got nil", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateSubscriptionURL(%q) unexpected error: %v", tt.url, err)
			}
		})
	}
}
