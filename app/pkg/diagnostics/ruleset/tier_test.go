package ruleset

import (
	"os"
	"path/filepath"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// localSetLoader builds a loader over one local rule set holding `content`.
func localSetLoader(t *testing.T, tag string, content []byte) *Loader {
	t.Helper()
	path := filepath.Join(t.TempDir(), tag+".srs")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write rule set: %v", err)
	}
	return NewLoader(nil, &option.Options{
		Route: &option.RouteOptions{RuleSet: []option.RuleSet{{
			Type: C.RuleSetTypeLocal,
			Tag:  tag,
			LocalOptions: option.LocalRuleSet{Path: path},
		}}},
	})
}

// undecodableSRS is a well-formed SRS header this build cannot finish reading,
// which is the shape a newer sing-box's cache entry takes here.
var undecodableSRS = []byte("SRS\x01\xff\xff\xff\xff")

func TestMatchTargetFallsBackToTheBinary(t *testing.T) {
	binary, _ := writeShim(t, "echo 'match rules.[0]: domain_suffix=google.com' >&2\nexit 0\n")

	loader := localSetLoader(t, "geosite-google", undecodableSRS)
	defer loader.Close()
	loader.EnableCLIFallback(binary)

	result := loader.MatchTarget("geosite-google", Target{Domain: "www.google.com"})
	if result.Verdict != VerdictYes {
		t.Fatalf("verdict = %v (%s), want VerdictYes", result.Verdict, result.Detail)
	}
	if result.Tier != TierBinary {
		t.Fatalf("tier = %q, want %q", result.Tier, TierBinary)
	}
}

func TestMatchTargetPrefersTheDecodedSet(t *testing.T) {
	// A shim that would claim a match, over a set this build CAN decode and
	// which does not match. Tier 1 must win, and nothing may be spawned.
	binary, argsFile := writeShim(t, "echo 'match rules.[0]: everything' >&2\nexit 0\n")

	loader := NewLoader(nil, &option.Options{
		Route: &option.RouteOptions{RuleSet: []option.RuleSet{{
			Type: C.RuleSetTypeInline,
			Tag:  "pinned",
			InlineOptions: option.PlainRuleSet{Rules: []option.HeadlessRule{{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultHeadlessRule{
					DomainSuffix: []string{"example.net"},
				},
			}}},
		}}},
	})
	defer loader.Close()
	loader.EnableCLIFallback(binary)

	result := loader.MatchTarget("pinned", Target{Domain: "www.google.com"})
	if result.Verdict != VerdictNo {
		t.Fatalf("verdict = %v, want VerdictNo", result.Verdict)
	}
	if result.Tier != TierDecoded {
		t.Fatalf("tier = %q, want %q", result.Tier, TierDecoded)
	}
	if _, err := os.Stat(argsFile); err == nil {
		t.Fatal("the binary was invoked for a set that decoded cleanly")
	}
}

func TestMatchTargetWithoutFallbackStaysUnknown(t *testing.T) {
	loader := localSetLoader(t, "geosite-google", undecodableSRS)
	defer loader.Close()

	result := loader.MatchTarget("geosite-google", Target{Domain: "www.google.com"})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", result.Verdict)
	}
	if result.Tier != TierNone {
		t.Fatalf("tier = %q, want %q", result.Tier, TierNone)
	}
}

func TestMatchTargetBinaryFailureStaysUnknown(t *testing.T) {
	// The whole point of the tier: a binary that cannot read the set must not
	// turn an undecidable rule into a confident miss.
	binary, _ := writeShim(t, "echo 'FATAL unsupported version: 9' >&2\nexit 1\n")

	loader := localSetLoader(t, "geosite-google", undecodableSRS)
	defer loader.Close()
	loader.EnableCLIFallback(binary)

	result := loader.MatchTarget("geosite-google", Target{Domain: "www.google.com"})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", result.Verdict)
	}
	if result.Detail == "" {
		t.Fatal("expected the binary's own message to be carried through")
	}
}

func TestMatchTargetDoesNotRespawnForTheSameTarget(t *testing.T) {
	// A production config reaches the same set from many rules. One spawn per
	// rule would mean ~19 processes per probe on the config this was built for.
	counter := filepath.Join(t.TempDir(), "count")
	binary, _ := writeShim(t, "echo x >> "+counter+"\nexit 0\n")

	loader := localSetLoader(t, "geosite-google", undecodableSRS)
	defer loader.Close()
	loader.EnableCLIFallback(binary)

	target := Target{Domain: "www.google.com"}
	loader.MatchTarget("geosite-google", target)
	loader.MatchTarget("geosite-google", target)
	loader.MatchTarget("geosite-google", target)

	raw, err := os.ReadFile(counter)
	if err != nil {
		t.Fatalf("shim never ran: %v", err)
	}
	if got := len(raw); got != 2 {
		t.Fatalf("binary ran %d times, want 1", got/2)
	}
}

func TestMatchTargetUnknownTagNeverSpawns(t *testing.T) {
	binary, argsFile := writeShim(t, "exit 0\n")

	loader := NewLoader(nil, &option.Options{})
	defer loader.Close()
	loader.EnableCLIFallback(binary)

	result := loader.MatchTarget("absent", Target{Domain: "www.google.com"})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %v, want VerdictUnknown", result.Verdict)
	}
	// unknown_tag is a config error, not a decode failure: there is no content
	// to hand the binary, so invoking it could only waste a spawn.
	if _, err := os.Stat(argsFile); err == nil {
		t.Fatal("the binary was invoked for a tag the config does not declare")
	}
}
