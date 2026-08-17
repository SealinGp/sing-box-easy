// Package noderules implements "Outbound Node Rules": a two-level grouping
// system that automatically organizes subscription-fetched endpoint outbounds
// into Filters and Groups on every subscription update.
//
// Concepts:
//   - Endpoint: a real exit node (vmess/trojan/...). Tagged elsewhere as
//     "<name> <server:port> | <subID>".
//   - Filter (user term: "Region Node"): a named, keyword/tag-matched bucket of
//     endpoints. Materializes as one urltest (default) or selector outbound.
//   - Group (user term: "Group Node"): a named, ordered set of Filters.
//     Materializes as one selector outbound whose members are Filter tags.
//   - "Other Nodes": a mandatory, undeletable fallback Filter that catches every
//     endpoint matching zero non-fallback Filters.
//
// Membership is multi-match: an endpoint joins every Filter whose matchers it
// satisfies; only endpoints matching no Filter fall through to "Other".
package noderules

import "time"

// MatcherType enumerates how a matcher value is interpreted.
type MatcherType string

const (
	// MatcherKeyword is a literal, case-insensitive substring match against the
	// endpoint tag (e.g. "Streaming", "YouTube", "IPLC").
	MatcherKeyword MatcherType = "keyword"
	// MatcherCode expands a short country/region code (e.g. "HK") to a curated
	// synonym set covering Chinese, English, code, and emoji-flag spellings.
	MatcherCode MatcherType = "code"
	// MatcherEmoji matches an exact emoji flag (e.g. "🇭🇰").
	MatcherEmoji MatcherType = "emoji"
)

const (
	// OutboundTypeURLTest is the default Filter materialization: auto-select the
	// lowest-latency member.
	OutboundTypeURLTest = "urltest"
	// OutboundTypeSelector materializes a Filter as a manual-pick selector.
	OutboundTypeSelector = "selector"
)

// urltest health-check defaults, applied when a Filter leaves them unset.
const (
	DefaultURLTestURL = "http://www.gstatic.com/generate_204"
	// DefaultURLTestInterval matches sing-box's own default. It used to be
	// "10s", which meant every Filter probed all of its members six times a
	// minute — with several large Filters that is tens of proxy handshakes per
	// second, sustained, and it starved a router badly enough that sing-box's
	// startup rule-set downloads timed out and procd crash-looped.
	//
	// The value is only a default; config.effectiveURLTestInterval additionally
	// clamps the resulting probe rate, which is what repairs Filters that
	// already have "10s" persisted.
	DefaultURLTestInterval  = "3m"
	DefaultURLTestTolerance = 200
)

const (
	// FallbackFilterID is the fixed primary key of the mandatory "Other" Filter.
	FallbackFilterID = "filter_other"
	// FallbackFilterName is the default display name / outbound tag of the
	// mandatory fallback Filter.
	FallbackFilterName = "Other Nodes"
	// FallbackPriority forces the fallback to be evaluated last regardless of its
	// stored priority (it only ever receives zero-match leftovers anyway).
	FallbackPriority = 1 << 30
)

// Matcher is one rule for assigning endpoints to a Filter.
type Matcher struct {
	Type  MatcherType `json:"type"`
	Value string      `json:"value"`
}

// Filter is the domain representation of a FilterRule (matchers parsed).
type Filter struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Matchers []Matcher `json:"matchers"`
	// Excludes is an optional deny-list evaluated AFTER Matchers: an endpoint
	// that satisfies a Matcher is still kept OUT of this Filter if it also
	// satisfies any Exclude (e.g. match code "US" but exclude the node tagged
	// "relay_bwh_us1"). Excluded endpoints fall through to the fallback Filter
	// unless another Filter still claims them.
	Excludes     []Matcher `json:"excludes"`
	OutboundType string    `json:"outbound_type"`
	Priority     int       `json:"priority"`
	IsFallback   bool      `json:"is_fallback"`
	// urltest health-check settings (ignored when OutboundType is "selector").
	TestURL       string    `json:"test_url"`
	TestInterval  string    `json:"test_interval"` // duration string, e.g. "10s"
	TestTolerance int       `json:"test_tolerance"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// URLTestSettings returns the effective urltest health-check settings, applying
// built-in defaults for any field the Filter leaves unset.
func (f *Filter) URLTestSettings() (url, interval string, tolerance int) {
	url = f.TestURL
	if url == "" {
		url = DefaultURLTestURL
	}
	interval = f.TestInterval
	if interval == "" {
		interval = DefaultURLTestInterval
	}
	tolerance = f.TestTolerance
	if tolerance <= 0 {
		tolerance = DefaultURLTestTolerance
	}
	return url, interval, tolerance
}

// Group is the domain representation of a GroupRule (filter IDs parsed).
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FilterIDs []string  `json:"filter_ids"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NormalizeOutboundType returns a valid Filter outbound type, defaulting to
// urltest for anything unrecognized.
func NormalizeOutboundType(t string) string {
	if t == OutboundTypeSelector {
		return OutboundTypeSelector
	}
	return OutboundTypeURLTest
}
