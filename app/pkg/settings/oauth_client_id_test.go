package settings

import "testing"

func TestValidateOAuthClientID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		// Shapes GitHub has actually issued: the modern "Ov23li…" prefix and
		// the older 20-character hex form.
		{name: "modern client id", id: "Ov23liAbCdEf01234567"},
		{name: "legacy hex client id", id: "1a2b3c4d5e6f70819243"},
		{name: "with separators", id: "Iv1.8a61f9b3a7aba766"},

		{name: "empty is rejected here", id: "", wantErr: true},
		{name: "too short", id: "abc", wantErr: true},
		{name: "embedded space", id: "Ov23li AbCdEf012345", wantErr: true},
		{name: "trailing newline", id: "Ov23liAbCdEf01234567\n", wantErr: true},
		{name: "control character", id: "Ov23liAbCdEf0123\x7f45", wantErr: true},
		// A client *secret* must never be accepted here — the device flow has
		// none, so a long opaque paste is almost certainly the wrong value.
		{name: "absurdly long", id: string(make([]byte, 200)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOAuthClientID(tt.id)
			if tt.wantErr && err == nil {
				t.Errorf("validateOAuthClientID(%q) = nil, want an error", tt.id)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateOAuthClientID(%q) = %v, want nil", tt.id, err)
			}
		})
	}
}
