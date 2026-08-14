package settings

import (
	"strings"
	"testing"
)

// TestValidateInfoKeywords verifies the storage boundary check: sane lists pass,
// oversized/blank/control-character entries are rejected with a clear message.
func TestValidateInfoKeywords(t *testing.T) {
	if err := validateInfoKeywords(nil); err != nil {
		t.Errorf("empty list should be valid (clears the override): %v", err)
	}
	if err := validateInfoKeywords([]string{"流量", "expire"}); err != nil {
		t.Errorf("normal list should be valid: %v", err)
	}

	cases := []struct {
		name string
		in   []string
	}{
		{"blank entry", []string{"traffic", "   "}},
		{"too long", []string{strings.Repeat("a", MaxInfoKeywordLen+1)}},
		{"control character", []string{"traf\nfic"}},
		{"too many", make([]string, MaxInfoKeywords+1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := validateInfoKeywords(c.in); err == nil {
				t.Errorf("validateInfoKeywords(%q) = nil, want an error", c.in)
			}
		})
	}
}
