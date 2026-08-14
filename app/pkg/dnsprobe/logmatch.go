package dnsprobe

import (
	"regexp"
	"strconv"
	"strings"
)

// LoggedMatch is one rule decision sing-box printed for a query. Unlike the
// offline Attribution this is not a prediction — it is the router's own record
// of which rule fired.
type LoggedMatch struct {
	// LoggedIndex is the number sing-box printed, verbatim.
	LoggedIndex int `json:"logged_index"`
	// ConfigIndex is LoggedIndex decoded back to a dns.rules position, or -1
	// when it could not be decoded.
	ConfigIndex int `json:"config_index"`
	// Description is the rule's conditions as sing-box rendered them, e.g.
	// "domain=tea.tparts.com".
	Description string `json:"description"`
	// Action is the right-hand side, e.g. "route(dns_router)".
	Action string `json:"action"`
	// Verified reports that ConfigIndex was cross-checked against the config
	// rule's own conditions. When false the index is a decode with no
	// corroboration and should be presented as such.
	Verified bool   `json:"verified"`
	Raw      string `json:"raw"`
}

// matchLinePattern parses sing-box's DNS decision line. The description is
// optional: sing-box omits it for rules with no printable conditions.
//
//	dns: match[1] domain=tea.tparts.com => predefined
//	dns: match[3] => route(dns_router)
var matchLinePattern = regexp.MustCompile(`match\[(\d+)\](.*?)=>\s*(.+?)\s*$`)

// decodeLoggedIndex converts the number sing-box prints back to a dns.rules
// index.
//
// sing-box computes the displayed value as `d += d + 1`, i.e. 2*index+1, so
// rule 0 logs as match[1], rule 1 as match[3], and so on (dns/router.go:135).
// That looks like a slip for `d += 1`, but it is what the released binary
// prints, so it is decoded here rather than trusted as an index. Anything that
// does not fit the pattern returns -1 instead of a plausible-looking guess.
func decodeLoggedIndex(logged int) int {
	if logged < 1 || logged%2 == 0 {
		return -1
	}
	return (logged - 1) / 2
}

// ParseMatchLines extracts the rule decisions for one query from a log slice.
//
// The Clash API issues its queries on a background context, so sing-box logs
// them without a connection ID and the lines cannot be correlated by identity.
// Correlation is therefore positional: `lines` must be the window captured
// around a single probe, and the caller is responsible for keeping that window
// tight. Concurrent traffic can still interleave, which is why the result is
// offered as evidence alongside the offline attribution rather than replacing
// it.
func ParseMatchLines(lines []string) []LoggedMatch {
	matches := make([]LoggedMatch, 0, 4)

	for _, line := range lines {
		if !strings.Contains(line, "match[") {
			continue
		}
		// Route decisions print the same shape under a different logger
		// prefix ("router: match[...]"), so anchor on the DNS one.
		if !strings.Contains(line, "dns:") {
			continue
		}
		groups := matchLinePattern.FindStringSubmatch(line)
		if groups == nil {
			continue
		}

		logged, err := strconv.Atoi(groups[1])
		if err != nil {
			continue
		}

		matches = append(matches, LoggedMatch{
			LoggedIndex: logged,
			ConfigIndex: decodeLoggedIndex(logged),
			Description: strings.TrimSpace(groups[2]),
			Action:      strings.TrimSpace(groups[3]),
			Raw:         strings.TrimSpace(line),
		})
	}

	return matches
}

// VerifyMatches cross-checks each decoded index against the attribution's view
// of the same rule, so a decode that points at an unrelated rule is reported as
// unverified instead of silently trusted.
//
// The check is deliberately weak — it compares the leading condition key (e.g.
// "domain_suffix") rather than the whole description, because sing-box
// truncates long lists differently than this package does. It is enough to
// catch an off-by-one or a decoding rule that changed between versions.
func VerifyMatches(matches []LoggedMatch, attribution Attribution) []LoggedMatch {
	verified := make([]LoggedMatch, len(matches))
	copy(verified, matches)

	for i := range verified {
		index := verified[i].ConfigIndex
		if index < 0 || index >= len(attribution.Rules) {
			verified[i].ConfigIndex = -1
			continue
		}
		if verified[i].Description == "" {
			// Nothing to compare against; leave the decode unverified.
			continue
		}
		if conditionKey(verified[i].Description) == conditionKey(attribution.Rules[index].Summary) {
			verified[i].Verified = true
		}
	}

	return verified
}

// conditionKey returns the first condition name in a rule description, e.g.
// "domain_suffix" from "domain_suffix=[a b] rule_set=[c]".
func conditionKey(description string) string {
	description = strings.TrimPrefix(strings.TrimSpace(description), "!(")
	name, _, found := strings.Cut(description, "=")
	if !found {
		return ""
	}
	return strings.TrimSpace(name)
}
