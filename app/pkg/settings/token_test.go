package settings

import "testing"

func TestValidateGitHubToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"classic PAT", "ghp_" + repeat("a", 36), false},
		{"fine-grained PAT", "github_pat_" + repeat("b", 60), false},
		{"opaque but plausible", "0123456789abcdef", false},
		{"too short", "ghp_", true},
		{"too long", repeat("a", maxTokenLen+1), true},
		{"embedded space", "ghp_abc def12345", true},
		{"embedded newline", "ghp_abcdef123\n", true},
		{"embedded tab", "ghp_abc\tdef12345", true},
		{"control character", "ghp_abc\x7fdef1234", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitHubToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitHubToken(%q) error = %v, wantErr = %v", tt.token, err, tt.wantErr)
			}
		})
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"ab", "••••"},
		{"abcd", "••••"},
		{"ghp_secret1234", "••••••••1234"},
	}

	for _, tt := range tests {
		if got := MaskSecret(tt.in); got != tt.want {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// MaskSecret must never echo enough of the credential to be usable.
func TestMaskSecretHidesMostOfTheToken(t *testing.T) {
	token := "ghp_" + repeat("s", 36)
	masked := MaskSecret(token)

	if len(masked) >= len(token) {
		t.Errorf("MaskSecret(%q) = %q, want it shorter than the input", token, masked)
	}
	if masked[len(masked)-4:] != token[len(token)-4:] {
		t.Errorf("MaskSecret should keep the last 4 characters, got %q", masked)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
