package dnsprobe

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sagernet/sing-box/option"
)

// maxDomainLength is the DNS limit for a name. Enforced so a probe cannot be
// used to push arbitrary payloads at the configured resolvers.
const maxDomainLength = 253

// logSettleDelay gives sing-box a moment to flush its decision lines before
// the log window is read back.
const logSettleDelay = 250 * time.Millisecond

// logWindowLines is how much log to snapshot either side of the query. Large
// enough to survive a burst of concurrent DNS traffic between the two reads.
const logWindowLines = 300

// SupportedQueryTypes are the record types the UI may ask for. Restricted on
// purpose: these are the ones that answer "where does this domain point".
var SupportedQueryTypes = []string{"A", "AAAA", "CNAME", "TXT", "MX", "NS"}

// Result is the full answer to one probe.
type Result struct {
	Domain    string `json:"domain"`
	QueryType string `json:"query_type"`

	// Live is what sing-box itself resolved. Nil when it could not be asked,
	// with LiveError explaining why.
	Live      *LiveResult `json:"live,omitempty"`
	LiveError string      `json:"live_error,omitempty"`

	// Attribution is the offline reconstruction of the routing decision.
	Attribution Attribution `json:"attribution"`

	// LoggedMatches is sing-box's own record of the decision, present only
	// when debug logging was enabled at probe time.
	LoggedMatches []LoggedMatch `json:"logged_matches"`
	// LogStatus is a machine-readable note about the log evidence, translated
	// by the UI. Prose here would arrive in one language regardless of the
	// viewer's locale.
	LogStatus LogStatus `json:"log_status,omitempty"`
	// LogError carries the underlying message when LogStatus is "read_error".
	LogError string `json:"log_error,omitempty"`

	// Servers holds each directly reachable upstream's answer, for comparison.
	Servers []ServerResult `json:"servers"`
	// Disagreement is true when two reachable servers returned different
	// records — the signal that something is rewriting answers.
	Disagreement bool `json:"disagreement"`
}

// LogStatus explains what the log evidence amounts to.
type LogStatus string

const (
	// LogStatusOK means exactly one decision was found and it is this query's.
	LogStatusOK LogStatus = ""
	// LogStatusNoLines means nothing was logged: no rule matched, the answer
	// was cached, or sing-box is not running at debug level.
	LogStatusNoLines LogStatus = "no_lines"
	// LogStatusAmbiguous means other DNS traffic shared the log window, so the
	// decisions cannot all be attributed to this query.
	LogStatusAmbiguous LogStatus = "ambiguous"
	// LogStatusReadError means the log could not be read at all.
	LogStatusReadError LogStatus = "read_error"
)

// LogTailer supplies the log window used for exact attribution. It matches the
// service controller's TailLogs shape so the handler can pass it straight
// through without the probe depending on the service package.
type LogTailer func(lines int, afterCursor string) ([]string, string, error)

// Options controls one probe run.
type Options struct {
	Domain    string
	QueryType string
	// CompareServers runs the per-upstream comparison. It issues real queries
	// to every reachable configured resolver, so it is opt-in.
	CompareServers bool
	// Tailer enables exact attribution from sing-box's debug log. Optional.
	Tailer LogTailer
}

// Run performs a probe against the supplied config.
//
// It never fails wholesale: each source (live answer, attribution, comparison)
// degrades independently so a probe still explains what it can when sing-box
// is stopped or the Clash API is off.
func Run(cfg *option.Options, opts Options) (*Result, error) {
	domain, err := NormalizeQueryDomain(opts.Domain)
	if err != nil {
		return nil, err
	}

	queryType := strings.ToUpper(strings.TrimSpace(opts.QueryType))
	if queryType == "" {
		queryType = "A"
	}
	if !isSupportedType(queryType) {
		return nil, fmt.Errorf("unsupported query type %q", queryType)
	}

	result := &Result{
		Domain:        domain,
		QueryType:     queryType,
		LoggedMatches: []LoggedMatch{},
		Servers:       []ServerResult{},
	}

	var dns *option.DNSOptions
	if cfg != nil {
		dns = cfg.DNS
	}
	result.Attribution = Attribute(dns, domain)

	// Snapshot the log before the query. Only lines that appear afterwards can
	// belong to this probe — see collectLoggedMatches for why the tailer's
	// cursor cannot be relied on for this.
	var before []string
	if opts.Tailer != nil {
		if lines, _, err := opts.Tailer(logWindowLines, ""); err == nil {
			before = lines
		}
	}

	result.Live, result.LiveError = queryLive(cfg, domain, queryType)

	if opts.Tailer != nil {
		result.LoggedMatches, result.LogStatus, result.LogError =
			collectLoggedMatches(opts.Tailer, before, result.Attribution)
	}

	if opts.CompareServers && dns != nil {
		result.Servers = QueryServers(dns.Servers, domain, queryType)
		result.Disagreement = hasDisagreement(result.Servers)
	}

	return result, nil
}

// queryLive asks sing-box for the authoritative answer.
func queryLive(cfg *option.Options, domain, queryType string) (*LiveResult, string) {
	if cfg == nil {
		return nil, "no configuration loaded"
	}

	client, err := NewClashClient(cfg.Experimental)
	if err != nil {
		if errors.Is(err, ErrClashAPIDisabled) {
			return nil, "sing-box cannot be queried: enable experimental.clash_api to see the live answer"
		}
		return nil, err.Error()
	}

	live, err := client.Query(domain, queryType)
	if err != nil {
		return nil, err.Error()
	}
	return live, ""
}

// collectLoggedMatches extracts sing-box's own rule decisions for this query.
//
// The window is established by diffing the log against a snapshot taken before
// the query, rather than by the tailer's cursor: only the systemd backend
// implements `afterCursor`, so on the file and logread backends (macOS, and
// every OpenWrt install) a cursor is silently ignored and the tailer returns
// the same trailing lines every time. Relying on it meant a probe that matched
// no rule inherited the previous probe's match and displayed it as confirmed.
func collectLoggedMatches(
	tailer LogTailer, before []string, attribution Attribution,
) ([]LoggedMatch, LogStatus, string) {
	time.Sleep(logSettleDelay)

	after, _, err := tailer(logWindowLines, "")
	if err != nil {
		return []LoggedMatch{}, LogStatusReadError, err.Error()
	}

	matches := VerifyMatches(ParseMatchLines(NewLines(before, after)), attribution)
	if len(matches) == 0 {
		return []LoggedMatch{}, LogStatusNoLines, ""
	}
	if len(matches) > 1 {
		// sing-box logs one match per query, so extra lines mean other DNS
		// traffic landed in the same window. The Clash API queries on a
		// background context, so the lines carry no ID to filter by.
		return matches, LogStatusAmbiguous, ""
	}
	return matches, LogStatusOK, ""
}

// NewLines returns the lines present in `after` but not yet in `before`.
//
// Logs only ever append, so the new lines are whatever follows the last line
// the snapshot ended with. When that anchor cannot be found — the log rotated,
// or the window scrolled past it — everything is treated as new, which is the
// safe direction: an over-wide window is filtered further by the caller, while
// a missed one would hide the very evidence being sought.
func NewLines(before, after []string) []string {
	if len(before) == 0 {
		return after
	}

	anchor := before[len(before)-1]
	for i := len(after) - 1; i >= 0; i-- {
		if after[i] == anchor {
			return after[i+1:]
		}
	}

	return after
}

// hasDisagreement reports whether two servers that both answered returned
// different record sets. Errors and skips are ignored: silence is not
// disagreement.
func hasDisagreement(results []ServerResult) bool {
	var reference []string
	found := false

	for _, result := range results {
		if result.Skipped != "" || result.Error != "" || len(result.Records) == 0 {
			continue
		}
		if !found {
			reference = result.Records
			found = true
			continue
		}
		if !equalRecords(reference, result.Records) {
			return true
		}
	}

	return false
}

func equalRecords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NormalizeQueryDomain validates and canonicalises a user-supplied domain.
//
// The probe sends this name to configured resolvers, so it is validated at the
// boundary: a URL, a host:port pair or an address literal is rejected rather
// than silently coerced.
func NormalizeQueryDomain(input string) (string, error) {
	domain := normalizeDomain(input)
	if domain == "" {
		return "", errors.New("domain is required")
	}
	if len(domain) > maxDomainLength {
		return "", fmt.Errorf("domain is longer than %d characters", maxDomainLength)
	}
	if strings.ContainsAny(domain, " \t/\\?#@:") {
		return "", errors.New("enter a bare domain name, without a scheme, port or path")
	}
	if net.ParseIP(domain) != nil {
		return "", errors.New("enter a domain name, not an IP address")
	}
	if !strings.Contains(domain, ".") {
		return "", errors.New("enter a fully qualified domain, e.g. example.com")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" {
			return "", errors.New("domain contains an empty label")
		}
		if len(label) > 63 {
			return "", errors.New("domain label is longer than 63 characters")
		}
	}

	return domain, nil
}

func isSupportedType(queryType string) bool {
	for _, supported := range SupportedQueryTypes {
		if supported == queryType {
			return true
		}
	}
	return false
}
