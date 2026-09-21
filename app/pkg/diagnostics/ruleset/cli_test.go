package ruleset

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeShim installs a fake `sing-box` whose `rule-set match` behaves as the
// script body says. The body is appended after a preamble that records the
// argument vector, so a test can assert on how the command was built as well
// as on how its output was read.
func writeShim(t *testing.T, body string) (binary string, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	binary = filepath.Join(dir, "sing-box")
	argsFile = filepath.Join(dir, "args.txt")

	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > " + argsFile + "\n" +
		body
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write shim: %v", err)
	}
	return binary, argsFile
}

func TestRunRuleSetMatchReadsTheMatchMarker(t *testing.T) {
	// The real command signals a hit with `println`, which writes to STDERR,
	// and always exits 0. Reading the exit code instead would report every
	// rule set as a match.
	binary, _ := writeShim(t, "echo 'match rules.[3]: domain_suffix=google.com' >&2\nexit 0\n")

	verdict, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte(`{"version":1,"rules":[]}`), "www.google.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != VerdictYes {
		t.Fatalf("verdict = %v, want VerdictYes", verdict)
	}
}

func TestRunRuleSetMatchTreatsSilenceAsNoMatch(t *testing.T) {
	binary, _ := writeShim(t, "exit 0\n")

	verdict, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte(`{"version":1,"rules":[]}`), "www.google.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != VerdictNo {
		t.Fatalf("verdict = %v, want VerdictNo", verdict)
	}
}

func TestRunRuleSetMatchFailureIsUnknownNotNoMatch(t *testing.T) {
	// A non-zero exit means the binary could not answer. Reporting that as
	// "does not match" is the single most dangerous mistake this tier can
	// make: it turns an unreadable set into a confident routing prediction.
	binary, _ := writeShim(t, "echo 'FATAL[0000] unsupported version: 4' >&2\nexit 1\n")

	verdict, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte("SRS\x00"), "www.google.com")
	if verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", verdict)
	}
	if err == nil {
		t.Fatal("expected an error describing the failure")
	}
	if !strings.Contains(err.Error(), "unsupported version") {
		t.Fatalf("error should carry the binary's own message, got %q", err)
	}
}

func TestRunRuleSetMatchMissingBinaryIsUnknown(t *testing.T) {
	verdict, err := runRuleSetMatch(context.Background(),
		filepath.Join(t.TempDir(), "absent"), time.Second,
		[]byte(`{"version":1,"rules":[]}`), "www.google.com")
	if verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", verdict)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestRunRuleSetMatchTimesOutAsUnknown(t *testing.T) {
	binary, _ := writeShim(t, "sleep 5\n")

	started := time.Now()
	verdict, err := runRuleSetMatch(context.Background(), binary, 200*time.Millisecond,
		[]byte(`{"version":1,"rules":[]}`), "www.google.com")
	if verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", verdict)
	}
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("timeout was not enforced: took %s", elapsed)
	}
}

func TestRunRuleSetMatchBuildsTheCommand(t *testing.T) {
	binary, argsFile := writeShim(t, "exit 0\n")

	if _, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte("SRS\x01rest"), "www.google.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("shim recorded no arguments: %v", err)
	}
	args := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(args) != 6 {
		t.Fatalf("args = %q, want 6 elements", args)
	}
	if args[0] != "rule-set" || args[1] != "match" {
		t.Fatalf("args = %q, want the `rule-set match` subcommand", args)
	}
	// Binary content must be declared: the command infers `source` from an
	// unknown extension and would then fail to parse a .srs file.
	if args[2] != "-f" || args[3] != "binary" {
		t.Fatalf("args = %q, want -f binary for SRS content", args)
	}
	if !strings.HasSuffix(args[4], ".srs") {
		t.Fatalf("path = %q, want a .srs extension", args[4])
	}
	if args[5] != "www.google.com" {
		t.Fatalf("query = %q", args[5])
	}
}

func TestRunRuleSetMatchDeclaresSourceFormatForJSON(t *testing.T) {
	binary, argsFile := writeShim(t, "exit 0\n")

	if _, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte(`{"version":1,"rules":[]}`), "www.google.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, _ := os.ReadFile(argsFile)
	args := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if args[3] != "source" || !strings.HasSuffix(args[4], ".json") {
		t.Fatalf("args = %q, want source format with a .json path", args)
	}
}

func TestRunRuleSetMatchRemovesItsTempFile(t *testing.T) {
	binary, argsFile := writeShim(t, "exit 0\n")

	if _, err := runRuleSetMatch(context.Background(), binary, time.Second,
		[]byte("SRS\x01rest"), "www.google.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, _ := os.ReadFile(argsFile)
	args := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if _, err := os.Stat(args[4]); !os.IsNotExist(err) {
		t.Fatalf("extracted rule set %q outlived the call", args[4])
	}
}
