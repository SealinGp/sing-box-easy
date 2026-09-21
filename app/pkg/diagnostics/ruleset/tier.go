package ruleset

// Tiered evaluation: decode in-process, else ask the installed binary, else
// say so.
//
// Set.Match remains the single-tier entry point and is unchanged. This adds
// the fallback around it, because the fallback needs a per-TARGET answer while
// a Set is per-tag: the binary is asked "does this set match www.google.com",
// not "give me this set's rules", so its verdict cannot be cached on the Set.

import (
	"strings"
	"time"
)

// Tier names which layer produced a verdict, so the UI can say whether an
// answer came from this build or from the binary that wrote the cache.
type Tier string

const (
	// TierNone means nothing could evaluate the set.
	TierNone Tier = ""
	// TierDecoded means the set was decoded and matched in this process.
	TierDecoded Tier = "decoded"
	// TierBinary means the installed sing-box answered it.
	TierBinary Tier = "sing-box"
)

// MatchResult is one set's verdict for one target, with its provenance.
type MatchResult struct {
	Verdict Verdict `json:"-"`
	// Set is the tag's loaded state, for the tag/type/staleness the UI shows.
	Set *Set `json:"-"`
	// Tier says which layer answered. Empty when none could.
	Tier Tier `json:"tier,omitempty"`
	// Detail carries the failure message when Tier is TierNone and the
	// fallback was tried — "unsupported version: 9" rather than a bare miss.
	Detail string `json:"detail,omitempty"`
}

// EnableCLIFallback lets the loader shell out to `binary` for sets this build
// cannot decode. Passing an empty path leaves the fallback off, which is the
// default: the fallback spawns a process per set and is only worth it when the
// in-process decode has actually failed.
func (l *Loader) EnableCLIFallback(binary string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cliBinary = strings.TrimSpace(binary)
}

// MatchTarget evaluates one tag against one target, falling back to the
// installed sing-box when this build cannot read the set.
//
// It never reports VerdictNo on the strength of a failure. Every path that
// could not reach an answer returns VerdictUnknown with a Detail, because
// "geoip-cn could not be read" and "the address is not in geoip-cn" route
// differently and the caller must be able to tell them apart.
func (l *Loader) MatchTarget(tag string, target Target) MatchResult {
	set := l.Get(tag)

	// Tier 1. An available set is authoritative — it is the same content the
	// binary would read, decoded without a process spawn.
	if set.Available {
		return MatchResult{Verdict: set.Match(target), Set: set, Tier: TierDecoded}
	}

	if !l.fallbackApplies(set) {
		return MatchResult{Verdict: VerdictUnknown, Set: set, Detail: set.Detail}
	}

	query := matchQuery(target)
	if query == "" {
		// The command takes exactly one address-or-domain argument, so a
		// target with neither cannot be expressed to it.
		return MatchResult{Verdict: VerdictUnknown, Set: set, Detail: set.Detail}
	}

	return l.matchViaCLI(set, query)
}

// fallbackApplies reports whether the binary could plausibly do better than
// this build did.
//
// Only a decode failure qualifies. A missing cache entry, a disabled cache or
// an unknown tag all mean there is no content to hand over, so a spawn could
// only waste time and then fail for the same reason.
func (l *Loader) fallbackApplies(set *Set) bool {
	l.mu.Lock()
	binary := l.cliBinary
	l.mu.Unlock()

	if binary == "" || len(set.content) == 0 {
		return false
	}
	switch set.Reason {
	case ReasonUnsupportedVersion, ReasonParseError:
		return true
	default:
		return false
	}
}

// matchViaCLI runs the binary, memoising per (tag, query).
//
// The memo is what keeps this affordable: the production config this was built
// against reaches the same rule set from many rules, and a spawn per REFERENCE
// rather than per distinct question would mean ~19 processes for one probe.
func (l *Loader) matchViaCLI(set *Set, query string) MatchResult {
	key := set.Tag + "\x00" + query

	l.mu.Lock()
	if cached, ok := l.cliResults[key]; ok {
		l.mu.Unlock()
		return cached
	}
	binary, ctx := l.cliBinary, l.ctx
	timeout := l.cliTimeout
	l.mu.Unlock()

	if timeout <= 0 {
		timeout = defaultCLITimeout
	}

	verdict, err := runRuleSetMatch(ctx, binary, timeout, set.content, query)
	result := MatchResult{Verdict: verdict, Set: set, Tier: TierBinary}
	if err != nil {
		result.Tier = TierNone
		result.Detail = err.Error()
	}

	l.mu.Lock()
	if l.cliResults == nil {
		l.cliResults = make(map[string]MatchResult)
	}
	l.cliResults[key] = result
	l.mu.Unlock()

	return result
}

// matchQuery renders a target as the single argument the command accepts.
//
// The domain wins when both are present, mirroring cmd_rule_set_match's own
// order: it parses the argument as an address and falls back to treating it as
// a name. A set keyed on domains is also the case this tier exists for —
// geosite-* is where version skew bites.
func matchQuery(target Target) string {
	if domain := strings.TrimSpace(target.Domain); domain != "" {
		return domain
	}
	if target.IP.IsValid() {
		return target.IP.String()
	}
	return ""
}

// SetCLITimeout overrides the per-invocation bound. Zero restores the default.
func (l *Loader) SetCLITimeout(timeout time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cliTimeout = timeout
}
