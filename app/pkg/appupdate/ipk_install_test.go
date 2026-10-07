package appupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShellQuoteNeutralisesQuotesAndExpansion(t *testing.T) {
	cases := map[string]string{
		"":             "''",
		"/tmp/a.ipk":   "'/tmp/a.ipk'",
		"a b":          "'a b'",
		"it's":         `'it'\''s'`,
		"$(reboot)":    "'$(reboot)'",
		"`reboot`; rm": "'`reboot`; rm'",
	}
	for input, want := range cases {
		if got := shellQuote(input); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestPackageInstallScriptLeavesTheCgroupBeforeAnythingElse(t *testing.T) {
	// Order is the whole point: a helper that starts opkg while still in
	// procd's service cgroup is killed together with the panel.
	script := buildPackageInstallScript(packageInstallParams{
		IpkPath: "/tmp/sing-box-easy_v1.5.4_x86_64.ipk", FromVersion: "v1.5.3", ToVersion: "v1.5.4",
	})

	leave := strings.Index(script, "cgroup.procs")
	install := strings.Index(script, "opkg install")
	if leave < 0 || install < 0 || leave > install {
		t.Fatalf("the script must leave the cgroup before running opkg (leave=%d install=%d)", leave, install)
	}
	if !strings.HasPrefix(script, "#!/bin/sh\n") {
		t.Fatal("the script must start with a /bin/sh shebang")
	}
}

func TestPackageInstallScriptAddsTheDowngradeFlagOnlyWhenAsked(t *testing.T) {
	upgrade := buildPackageInstallScript(packageInstallParams{IpkPath: "/tmp/a.ipk"})
	if !strings.Contains(upgrade, "FLAGS=''\n") {
		t.Errorf("an upgrade must pass no flags:\n%s", upgrade)
	}
	downgrade := buildPackageInstallScript(packageInstallParams{IpkPath: "/tmp/a.ipk", Downgrade: true})
	if !strings.Contains(downgrade, "FLAGS='--force-downgrade'\n") {
		t.Errorf("a downgrade must pass --force-downgrade:\n%s", downgrade)
	}
}

func TestPackageInstallScriptQuotesEveryValue(t *testing.T) {
	script := buildPackageInstallScript(packageInstallParams{
		IpkPath: "/tmp/x'; reboot; '.ipk", FromVersion: "$(reboot)", ToVersion: "`reboot`",
	})
	for _, want := range []string{
		`IPK='/tmp/x'\''; reboot; '\''.ipk'`,
		"FROM='$(reboot)'",
		"TO='`reboot`'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script is missing the quoted value %s", want)
		}
	}
}

func TestPackageInstallScriptIsValidShell(t *testing.T) {
	// `sh -n` parses without executing. A syntax error here would only show up
	// on a router, after the panel had already been stopped.
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on this host")
	}
	path := filepath.Join(t.TempDir(), "helper.sh")
	script := buildPackageInstallScript(packageInstallParams{
		IpkPath: "/tmp/a b.ipk", Downgrade: true, FromVersion: "v1.5.5", ToVersion: "v1.5.4",
	})
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(shell, "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("generated script does not parse: %v\n%s", err, output)
	}
}

func TestParsePackageInstallState(t *testing.T) {
	now := time.Unix(1_800_000_100, 0)

	t.Run("succeeded", func(t *testing.T) {
		result := parsePackageInstallState([]byte(
			"state=succeeded\nexit_code=0\nfrom=v1.5.3\nto=v1.5.4\nstarted_at=1800000000\nfinished_at=1800000009\n"),
			"== done", now)
		if result == nil || result.State != PackageInstallSucceeded {
			t.Fatalf("result = %+v, want succeeded", result)
		}
		if result.ExitCode != 0 || result.FromVersion != "v1.5.3" || result.ToVersion != "v1.5.4" {
			t.Errorf("unexpected fields: %+v", result)
		}
		if result.StartedAt == "" || result.FinishedAt == "" || result.LogTail != "== done" {
			t.Errorf("timestamps and log must be carried: %+v", result)
		}
	})

	t.Run("failed keeps opkg's exit code", func(t *testing.T) {
		result := parsePackageInstallState([]byte(
			"state=failed\nexit_code=255\nfrom=a\nto=b\nstarted_at=1800000000\nfinished_at=1800000004\n"), "", now)
		if result == nil || result.State != PackageInstallFailed || result.ExitCode != 255 {
			t.Fatalf("result = %+v, want failed/255", result)
		}
	})

	t.Run("running while fresh", func(t *testing.T) {
		result := parsePackageInstallState([]byte(
			"state=running\nexit_code=\nfrom=a\nto=b\nstarted_at=1800000090\nfinished_at=\n"), "", now)
		if result == nil || result.State != PackageInstallRunning || result.ExitCode != -1 {
			t.Fatalf("result = %+v, want running with unknown exit code", result)
		}
	})

	t.Run("running long ago is interrupted, not running forever", func(t *testing.T) {
		stale := now.Add(packageInstallStaleAfter + time.Minute)
		result := parsePackageInstallState([]byte(
			"state=running\nexit_code=\nfrom=a\nto=b\nstarted_at=1800000090\nfinished_at=\n"), "", stale)
		if result == nil || result.State != PackageInstallInterrupted {
			t.Fatalf("result = %+v, want interrupted", result)
		}
	})

	t.Run("garbage is no result", func(t *testing.T) {
		for _, data := range []string{"", "state=\n", "state=weird\n", "not a state file"} {
			if result := parsePackageInstallState([]byte(data), "", now); result != nil {
				t.Errorf("parse(%q) = %+v, want nil", data, result)
			}
		}
	})
}

func TestCheckInstallablePlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkg.ipk")
	if err := os.WriteFile(path, []byte("package bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := checkInstallablePlan(&IpkPlan{Path: path, SHA256: sum, Verified: true}); err != nil {
		t.Fatalf("a verified, unchanged package must be installable: %v", err)
	}
	if err := checkInstallablePlan(nil); err == nil {
		t.Error("a missing plan must be refused")
	}
	// The manual path tolerates an unverified download because a person reads
	// the warning. Nobody reads it here.
	if err := checkInstallablePlan(&IpkPlan{Path: path, SHA256: sum, Verified: false}); err == nil {
		t.Error("an unverified package must not be installed automatically")
	}
	if err := checkInstallablePlan(&IpkPlan{Path: filepath.Join(dir, "gone.ipk"), SHA256: sum, Verified: true}); err == nil {
		t.Error("a package that is no longer on disk must be refused")
	}

	// Swapped in /tmp between prepare and install.
	if err := os.WriteFile(path, []byte("something else"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkInstallablePlan(&IpkPlan{Path: path, SHA256: sum, Verified: true}); err == nil {
		t.Error("a package that changed on disk after verification must be refused")
	}
}

func TestStartIpkInstallRefusesOffOpkg(t *testing.T) {
	// This host is not opkg-managed, so nothing may be launched.
	if InstalledViaOpkg() {
		t.Skip("running on an opkg-managed host")
	}
	if _, err := NewUpdater("", nil).StartIpkInstall("ipk_1"); err == nil {
		t.Fatal("expected a refusal on a host that is not opkg-managed")
	}
}
