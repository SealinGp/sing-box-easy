package appupdate

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal plain", "1.2.3", "1.2.3", 0},
		{"equal with v prefix", "v1.2.3", "1.2.3", 0},
		{"equal uppercase V", "V1.2.3", "v1.2.3", 0},
		{"major greater", "v2.0.0", "v1.9.9", 1},
		{"major lesser", "v1.9.9", "v2.0.0", -1},
		{"minor greater", "v1.13.0", "v1.9.0", 1},
		{"patch greater", "v1.2.10", "v1.2.9", 1},
		{"missing patch defaults to zero", "v1.2", "v1.2.0", 0},
		{"missing minor defaults to zero", "v1", "v1.0.0", 0},
		{"release beats prerelease", "v1.2.3", "v1.2.3-rc.1", 1},
		{"prerelease loses to release", "v1.2.3-rc.1", "v1.2.3", -1},
		{"prerelease ordering", "v1.2.3-rc.1", "v1.2.3-rc.2", -1},
		{"build metadata ignored", "v1.2.3+abc", "v1.2.3", 0},
		{"whitespace tolerated", "  v1.2.3 ", "v1.2.3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareVersions(tt.a, tt.b); got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCompareVersionsIsAntisymmetric(t *testing.T) {
	pairs := [][2]string{
		{"v1.2.3", "v1.2.4"},
		{"v2.0.0", "v1.0.0"},
		{"v1.0.0-rc1", "v1.0.0"},
	}

	for _, p := range pairs {
		forward := CompareVersions(p[0], p[1])
		reverse := CompareVersions(p[1], p[0])
		if forward != -reverse {
			t.Errorf("CompareVersions(%q,%q)=%d but reverse=%d; want negation",
				p[0], p[1], forward, reverse)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name               string
		candidate, current string
		want               bool
	}{
		{"strictly newer", "v1.3.0", "v1.2.0", true},
		{"same version", "v1.2.0", "v1.2.0", false},
		{"older candidate", "v1.1.0", "v1.2.0", false},
		{"unstamped dev build accepts anything", "v1.0.0", DevVersion, true},
		{"empty current accepts anything", "v1.0.0", "", true},
		{"empty candidate is never newer", "", "v1.0.0", false},
		{"empty candidate with dev current", "", DevVersion, false},
		{"prerelease over release", "v1.3.0-rc1", "v1.2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNewer(tt.candidate, tt.current); got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.candidate, tt.current, got, tt.want)
			}
		})
	}
}

func TestParseSemverHandlesGarbage(t *testing.T) {
	// Unparseable components must degrade to 0 rather than panic, so an
	// unexpected tag shape can never crash the update check.
	got := parseSemver("vfoo.bar.baz")
	if got.major != 0 || got.minor != 0 || got.patch != 0 {
		t.Errorf("parseSemver(garbage) = %+v, want all zeros", got)
	}
}
