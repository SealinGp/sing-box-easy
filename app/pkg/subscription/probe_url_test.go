package subscription

import "testing"

// TestNormalizeProbeURL guards the one trap in this feature: sing-box silently
// throws away an http:// delay-test URL and substitutes its own default, so a
// value that looks accepted would produce latency for an endpoint nobody chose.
func TestNormalizeProbeURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "empty means the default", in: "", want: ""},
		{name: "whitespace only means the default", in: "   ", want: ""},
		{name: "https is kept", in: "https://cp.cloudflare.com/generate_204", want: "https://cp.cloudflare.com/generate_204"},
		{name: "surrounding whitespace is trimmed", in: "  https://example.com/204  ", want: "https://example.com/204"},

		// The whole reason this function exists.
		{name: "http is rejected, not silently upgraded", in: "http://www.gstatic.com/generate_204", wantErr: true},

		{name: "other schemes are rejected", in: "ftp://example.com/x", wantErr: true},
		{name: "a bare domain is rejected", in: "example.com/generate_204", wantErr: true},
		{name: "a scheme with no host is rejected", in: "https://", wantErr: true},
		{name: "garbage is rejected", in: "https://exa mple.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeProbeURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeProbeURL(%q) = %q, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeProbeURL(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeProbeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestEffectiveProbeURL — an empty stored value must resolve to the default at
// read time, not be written into every row at save time. Baking it in would
// freeze today's default into subscriptions created today.
func TestEffectiveProbeURL(t *testing.T) {
	if got := EffectiveProbeURL(""); got != DefaultProbeURL {
		t.Errorf("EffectiveProbeURL(\"\") = %q, want the default", got)
	}
	if got := EffectiveProbeURL("https://example.com/204"); got != "https://example.com/204" {
		t.Errorf("EffectiveProbeURL kept = %q", got)
	}
	// A value stored before validation existed (or written by another API
	// client) must not reach the prober: it would be silently swapped by
	// sing-box and the numbers would describe the wrong endpoint.
	if got := EffectiveProbeURL("http://example.com/204"); got != DefaultProbeURL {
		t.Errorf("EffectiveProbeURL(http) = %q, want the default", got)
	}
}
