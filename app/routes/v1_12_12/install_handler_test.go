package v1_13_0

import "testing"

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty allowed", "", "", false},
		{"simple", "zashboard", "zashboard", false},
		{"trimmed", "  zashboard  ", "zashboard", false},
		{"with dash and dot", "v1.2-final", "v1.2-final", false},
		{"unicode ok", "面板", "面板", false},

		{"slash rejected", "foo/bar", "", true},
		{"backslash rejected", "foo\\bar", "", true},
		{"parent rejected", "..", "", true},
		{"dot rejected", ".", "", true},
		{"traversal rejected", "../etc", "", true},
		{"absolute rejected", "/etc/passwd", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeFolderName(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("sanitizeFolderName(%q) wanted error, got %q", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("sanitizeFolderName(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("sanitizeFolderName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeTargetDir(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty allowed", "", "", false},
		{"absolute clean", "/etc/sing-box/ui", "/etc/sing-box/ui", false},
		{"trailing slash cleaned", "/etc/sing-box/ui/", "/etc/sing-box/ui", false},

		{"relative rejected", "etc/ui", "", true},
		{"dot rejected", "./etc", "", true},
		{"traversal rejected", "/etc/../tmp", "", true},
		{"embedded traversal rejected", "/etc/sing-box/../passwd", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeTargetDir(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("sanitizeTargetDir(%q) wanted error, got %q", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("sanitizeTargetDir(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("sanitizeTargetDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
