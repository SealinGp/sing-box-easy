package subscription

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultProbeURL is the target used when a subscription names none.
//
// Mirrors subprobe.DefaultProbeURL and sing-box's own default. Duplicated as a
// constant here rather than imported so the subscription package does not
// depend on the prober — the value is a documented part of sing-box's URL-test
// contract, not a knob either package owns.
const DefaultProbeURL = "https://www.gstatic.com/generate_204"

// NormalizeProbeURL validates an operator-supplied latency-test target and
// returns it trimmed, or "" to mean "use the default".
//
// HTTPS-ONLY, and this is not a security preference. sing-box's delay endpoint
// contains:
//
//	if strings.HasPrefix(url, "http://") { url = "" }
//
// which makes it fall back to its own https default. So an accepted http:// URL
// would be reported back to the operator as their configured target while every
// measurement described a completely different endpoint — the worst kind of
// wrong, because nothing on screen would disagree with itself. Rejecting it at
// the boundary is the only place that can be said out loud.
func NormalizeProbeURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid probe url: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("probe url must start with https:// (sing-box silently ignores http:// targets and tests %s instead)", DefaultProbeURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("probe url must include a host")
	}
	return trimmed, nil
}

// EffectiveProbeURL resolves what will actually be tested.
//
// Resolved at READ time rather than written into the row, so a subscription
// saved today follows a later change of default instead of pinning it. It also
// re-validates: a row written before this rule existed, or by any other client
// of this API, must not reach the prober — the render-time guard that
// safeExternalUrl.ts provides for official_url, for the same reason.
func EffectiveProbeURL(stored string) string {
	normalized, err := NormalizeProbeURL(stored)
	if err != nil || normalized == "" {
		return DefaultProbeURL
	}
	return normalized
}
