package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapPasswordFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	password := " spaces '$\\quotes\"% "
	if err := os.WriteFile(path, []byte(password), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadBootstrapPassword("", path)
	if err != nil || got != password {
		t.Fatalf("password not preserved: %v", err)
	}
	if _, err := ReadBootstrapPassword("conflict", path); err == nil {
		t.Fatal("expected conflicting inputs to fail")
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBootstrapPassword("", path); err == nil {
		t.Fatal("empty file must not fall back to admin")
	}
	if _, err := ReadBootstrapPassword("", path+"missing"); err == nil {
		t.Fatal("missing file must fail")
	}
}
func TestBootstrapPasswordFlagCompatibility(t *testing.T) {
	got, err := ReadBootstrapPassword("existing", "")
	if err != nil || got != "existing" {
		t.Fatal("legacy flag broken")
	}
}
