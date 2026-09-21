package ruleset

// Tier 2 of rule-set evaluation: ask the INSTALLED sing-box binary.
//
// WHY THIS EXISTS
// ───────────────
// The in-process matcher decodes .srs with the sing-box library this panel is
// COMPILED against, which the repo pins (1.12.12). The binary on the host is
// whatever the operator installed, and a newer one writes a newer .srs version
// into the same cache file. srs.Read then refuses it outright
// (common/srs/binary.go: "unsupported version: N"), and every rule carrying
// that set becomes undecidable — on a real config that is most of them.
//
// Shelling out closes exactly that gap, because the binary that WROTE the cache
// is the binary being asked to read it. It is a fallback and not the primary
// path: it costs a process spawn and a temp file per set, on a device that is
// also routing traffic.
//
// WHAT THE COMMAND ACTUALLY DOES — three details, each of which silently
// inverts the result if assumed rather than checked
// (cmd/sing-box/cmd_rule_set_match.go):
//
//  1. It ALWAYS exits 0 on a successful read, whether or not anything matched.
//     Branching on the exit code reports every rule set as a match.
//  2. A hit is announced with the builtin `println`, which writes to STDERR,
//     not stdout. Reading only stdout reports every rule set as a miss.
//  3. The format is inferred from the file EXTENSION when -f is omitted, and
//     an unrecognised extension leaves it as "source" — so binary content in a
//     file the command cannot classify fails to parse instead of matching.
//
// So: the marker on the combined output decides yes/no, and only a non-zero
// exit (or a failure to run at all) yields VerdictUnknown. That asymmetry is
// deliberate — an unreadable set must never collapse into "does not match".

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// matchMarker is what cmd_rule_set_match prints for every rule that matched.
// Matched on text because the command has no machine-readable output mode.
const matchMarker = "match rules.["

// defaultCLITimeout bounds one invocation. The command reads one file and
// walks it in memory; anything slower is a stuck process, not a slow one.
const defaultCLITimeout = 3 * time.Second

// runRuleSetMatch evaluates `content` against `query` using the installed
// sing-box binary.
//
// It returns VerdictUnknown together with an error whenever the binary could
// not answer — missing, timed out, or refused the file. The caller must
// propagate that as undecidable rather than as a miss.
func runRuleSetMatch(
	ctx context.Context, binary string, timeout time.Duration, content []byte, query string,
) (Verdict, error) {
	if strings.TrimSpace(binary) == "" {
		return VerdictUnknown, errors.New("no sing-box binary configured")
	}
	if len(content) == 0 {
		return VerdictUnknown, errors.New("rule set has no content to match against")
	}
	if strings.TrimSpace(query) == "" {
		return VerdictUnknown, errors.New("no domain or address to match")
	}
	if timeout <= 0 {
		timeout = defaultCLITimeout
	}

	format, extension := "source", ".json"
	if looksBinary(content) {
		format, extension = "binary", ".srs"
	}

	path, cleanup, err := writeTempRuleSet(content, extension)
	if err != nil {
		return VerdictUnknown, err
	}
	// The file is this process's, not the user's cache: it must go whether the
	// command succeeded, failed, or was killed by the timeout.
	defer cleanup()

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(runCtx, binary, "rule-set", "match", "-f", format, path, query)
	// Kill the child on every return path. An ignored one is an orphaned
	// process per probe — the same leak the log follower's comments warn about.
	command.Cancel = func() error { return command.Process.Kill() }

	// The marker goes to stderr and diagnostics may go to either, so both are
	// read as one stream. Nothing here is parsed positionally.
	output, runErr := command.CombinedOutput()

	if runErr != nil {
		if runCtx.Err() != nil {
			return VerdictUnknown, fmt.Errorf(
				"sing-box rule-set match timed out after %s", timeout)
		}
		return VerdictUnknown, fmt.Errorf(
			"sing-box rule-set match failed: %s", firstMeaningfulLine(string(output), runErr))
	}

	if strings.Contains(string(output), matchMarker) {
		return VerdictYes, nil
	}
	return VerdictNo, nil
}

// writeTempRuleSet materialises cached rule-set bytes as a file, because the
// command takes a path and has no stdin mode for a named format.
//
// The extension is not cosmetic: see detail (3) above.
func writeTempRuleSet(content []byte, extension string) (string, func(), error) {
	file, err := os.CreateTemp("", "sbe-ruleset-*"+extension)
	if err != nil {
		return "", func() {}, fmt.Errorf("failed to stage rule set for matching: %w", err)
	}
	path := file.Name()
	cleanup := func() { os.Remove(path) }

	if _, err := file.Write(content); err != nil {
		file.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("failed to stage rule set for matching: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("failed to stage rule set for matching: %w", err)
	}

	// CreateTemp already guarantees the suffix, but a caller that passed an
	// empty one would silently lose format detection.
	if filepath.Ext(path) != extension {
		cleanup()
		return "", func() {}, fmt.Errorf("staged rule set has no %s extension", extension)
	}
	return path, cleanup, nil
}

// firstMeaningfulLine picks the line most likely to explain a failure, so the
// UI shows "unsupported version: 4" rather than "exit status 1".
func firstMeaningfulLine(output string, fallback error) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback.Error()
}
