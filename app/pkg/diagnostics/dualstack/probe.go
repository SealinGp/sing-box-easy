package dualstack

// The orchestrator: resolve, predict, dial, observe — per domain, per family.

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
)

// Limits on what one request may ask for.
//
// This endpoint makes the SERVER dial addresses chosen, indirectly, by the
// caller. That is a deliberate and necessary capability — it is the only way
// to test the router's own egress — but it is also an SSRF surface, so its
// bounds are explicit rather than implied by whatever the caller sends.
const (
	// MaxDomains caps one request. Each domain costs two DNS queries and up
	// to two dials, on a device that is also routing traffic.
	MaxDomains = 10
	// DefaultPort matches what the script tests and what a browser opens.
	DefaultPort uint16 = 443
	// observeAttempts is how many times /connections is polled while the
	// probe's connection is held open.
	observeAttempts = 4
	// observeInterval spaces those polls.
	observeInterval = 150 * time.Millisecond
)

// Family is one address family under test.
type Family string

const (
	FamilyV4 Family = "ipv4"
	FamilyV6 Family = "ipv6"
)

// queryType is the record type that produces this family's addresses.
func (f Family) queryType() string {
	if f == FamilyV6 {
		return "AAAA"
	}
	return "A"
}

// Agreement compares the prediction with the observation.
type Agreement string

const (
	// AgreementUnknown means one side is missing, so there is nothing to
	// compare — not that they disagree.
	AgreementUnknown Agreement = "unknown"
	// AgreementYes means traffic went where the config said it would.
	AgreementYes Agreement = "agree"
	// AgreementNo means it did not. This is the headline: it is the config bug
	// the whole feature exists to surface.
	AgreementNo Agreement = "differ"
)

// Prediction is where the offline route walk says this address would go.
type Prediction struct {
	Outbound string `json:"outbound"`
	// Source is which config key produced it: "rule", "route.final", …
	Source string `json:"source,omitempty"`
	// MatchedIndex is the deciding route rule, or -1 for a fall-through.
	MatchedIndex int `json:"matched_index"`
	// Exact is false when an undecidable rule sat ahead of the decision, so
	// the prediction is a best guess and a disagreement may be the
	// prediction's fault rather than the config's.
	Exact bool   `json:"exact"`
	Error string `json:"error,omitempty"`
}

// FamilyResult is one domain's answer for one address family.
type FamilyResult struct {
	Family    Family `json:"family"`
	QueryType string `json:"query_type"`
	// Addresses is everything sing-box returned for this family.
	Addresses []string `json:"addresses"`
	// Tested is the address actually dialled — the first returned, which is
	// also the one a client would use.
	Tested string `json:"tested,omitempty"`
	// DNSElapsedMS is the resolve round trip, and DNSError why it failed.
	DNSElapsedMS int64  `json:"dns_elapsed_ms"`
	DNSError     string `json:"dns_error,omitempty"`

	Predicted *Prediction  `json:"predicted,omitempty"`
	Dial      *DialResult  `json:"dial,omitempty"`
	Observed  *Observed    `json:"observed,omitempty"`
	Status    Reachability `json:"status"`
	Agreement Agreement    `json:"agreement"`
}

// DomainResult is one domain across both families.
type DomainResult struct {
	Domain   string         `json:"domain"`
	Families []FamilyResult `json:"families"`
	Error    string         `json:"error,omitempty"`
}

// Result is the whole run.
type Result struct {
	Domains []DomainResult `json:"domains"`
	// DNSEndpoint is where a LAN client's DNS lands, per the route rules.
	// Reported because a config whose hijack is missing explains every
	// surprising answer at once.
	DNSEndpoint DNSEndpoint `json:"dns_endpoint"`
	Port        uint16      `json:"port"`
	// ClientV4/ClientV6 say what this host could test at all. A panel with no
	// global IPv6 reports `untested`, never `unreachable`.
	ClientV4 bool `json:"client_ipv4"`
	ClientV6 bool `json:"client_ipv6"`
	// ObserveError explains why correlation was unavailable — typically the
	// Clash API being off. Separate from a per-family failure because it
	// applies to every row at once.
	ObserveError string `json:"observe_error,omitempty"`
}

// Resolver asks sing-box to resolve a name. Satisfied by the DNS probe's
// Clash client.
type Resolver interface {
	Query(name, qType string) (addresses []string, elapsedMS int64, err error)
}

// Snapshotter reads sing-box's live connection table.
type Snapshotter interface {
	Connections(ctx context.Context) (*clashapi.Snapshot, error)
}

// Predictor predicts where one literal address would be routed.
type Predictor func(address netip.Addr, port uint16) (*Prediction, error)

// Options controls one run.
type Options struct {
	Domains []string
	Port    uint16
	// TLS completes a handshake after connecting. Off makes the test a plain
	// reachability check, which is the right thing for a non-TLS port.
	TLS bool
	// SkipDial reports the DNS and prediction halves only, sending no
	// traffic. Useful on a device where the operator does not want the panel
	// opening connections at all.
	SkipDial bool
	Timeout  time.Duration

	Resolve Resolver
	Predict Predictor
	Observe Snapshotter
}

// Stage names the phases, in order, for progress reporting. Each is a real
// source of latency — see the DNS probe's Stage for why none is invented.
type Stage string

const (
	// StageEndpoint is the config read that locates the DNS hijack: instant,
	// emitted first so the UI has a frame to draw into.
	StageEndpoint Stage = "endpoint"
	// StageResolve is two Clash API queries per domain.
	StageResolve Stage = "resolve"
	// StagePredict is the offline route walk: microseconds, but it may read
	// rule sets from sing-box's cache.
	StagePredict Stage = "predict"
	// StageTraffic is the dial plus the connection-table polls.
	StageTraffic Stage = "traffic"
)

// StageFunc is called as each stage completes, with the result so far. The
// pointer keeps being written after the call returns, so an implementation
// must serialize what it needs rather than retaining it.
//
// Returning an error aborts the run — used when the client has disconnected
// and the remaining work would open connections for nobody.
type StageFunc func(stage Stage, partial *Result) error

// Run performs a probe with no progress reporting.
func Run(ctx context.Context, endpoint DNSEndpoint, opts Options) (*Result, error) {
	return RunStaged(ctx, endpoint, opts, nil)
}

// RunStaged is Run with progress reporting. onStage may be nil.
//
// It fails only on input it cannot interpret. Everything else — a resolver
// that is down, a rule set that cannot be read, an address that will not
// answer — lands in the result, where the operator can see it.
func RunStaged(
	ctx context.Context, endpoint DNSEndpoint, opts Options, onStage StageFunc,
) (*Result, error) {
	domains, err := NormalizeDomains(opts.Domains)
	if err != nil {
		return nil, err
	}
	port := opts.Port
	if port == 0 {
		port = DefaultPort
	}

	result := &Result{
		Domains:     make([]DomainResult, 0, len(domains)),
		DNSEndpoint: endpoint,
		Port:        port,
		ClientV4:    HasGlobalAddress(false),
		ClientV6:    HasGlobalAddress(true),
	}

	report := func(stage Stage) error {
		if onStage == nil {
			return nil
		}
		return onStage(stage, result)
	}
	if err := report(StageEndpoint); err != nil {
		return result, nil
	}

	for _, domain := range domains {
		entry := DomainResult{Domain: domain, Families: []FamilyResult{}}
		for _, family := range []Family{FamilyV4, FamilyV6} {
			entry.Families = append(entry.Families, resolveFamily(domain, family, opts))
		}
		result.Domains = append(result.Domains, entry)
	}
	if err := report(StageResolve); err != nil {
		return result, nil
	}

	for i := range result.Domains {
		for j := range result.Domains[i].Families {
			predictFamily(&result.Domains[i].Families[j], port, opts)
		}
	}
	if err := report(StagePredict); err != nil {
		return result, nil
	}

	// Serially, on purpose. Each dial holds a connection open while the
	// connection table is polled, and correlation keys on (address, port,
	// time) — concurrent dials to the same CDN address would be
	// indistinguishable from each other. The cost is bounded by MaxDomains
	// and the per-dial timeout, on a device that is also routing traffic.
	if !opts.SkipDial {
		for i := range result.Domains {
			domain := result.Domains[i].Domain
			for j := range result.Domains[i].Families {
				testFamily(ctx, domain, &result.Domains[i].Families[j], port, opts, result)
			}
		}
	} else {
		markUntested(result, "dialling was not requested")
	}

	for i := range result.Domains {
		for j := range result.Domains[i].Families {
			family := &result.Domains[i].Families[j]
			family.Agreement = compare(family)
		}
	}
	if err := report(StageTraffic); err != nil {
		return result, nil
	}

	return result, nil
}

// resolveFamily asks sing-box for one family's addresses.
func resolveFamily(domain string, family Family, opts Options) FamilyResult {
	entry := FamilyResult{
		Family:    family,
		QueryType: family.queryType(),
		Addresses: []string{},
		Status:    ReachabilityNoAddress,
	}

	if opts.Resolve == nil {
		entry.DNSError = "sing-box cannot be queried: enable experimental.clash_api to resolve through it"
		entry.Status = ReachabilityUntested
		return entry
	}

	addresses, elapsed, err := opts.Resolve.Query(domain, entry.QueryType)
	entry.DNSElapsedMS = elapsed
	if err != nil {
		entry.DNSError = err.Error()
		// A resolver that could not be asked is not a domain with no address.
		entry.Status = ReachabilityUntested
		return entry
	}

	for _, address := range addresses {
		parsed, parseErr := netip.ParseAddr(address)
		if parseErr != nil {
			continue
		}
		// sing-box answers an A query with A records, but a defensive filter
		// costs nothing and keeps an IPv4-mapped answer out of the v6 row —
		// where it would look like working IPv6 and is not.
		if wantsV6 := family == FamilyV6; parsed.Unmap().Is6() != wantsV6 {
			continue
		}
		entry.Addresses = append(entry.Addresses, parsed.String())
	}

	if len(entry.Addresses) > 0 {
		// The first is what a client would use, so it is what gets tested.
		entry.Tested = entry.Addresses[0]
	}
	return entry
}

// predictFamily fills in where the tested address would be routed.
func predictFamily(entry *FamilyResult, port uint16, opts Options) {
	if entry.Tested == "" || opts.Predict == nil {
		return
	}
	address, err := netip.ParseAddr(entry.Tested)
	if err != nil {
		return
	}
	prediction, predictErr := opts.Predict(address, port)
	if predictErr != nil {
		entry.Predicted = &Prediction{MatchedIndex: -1, Error: predictErr.Error()}
		return
	}
	entry.Predicted = prediction
}

// testFamily dials the tested address and correlates the connection.
func testFamily(
	ctx context.Context, domain string, entry *FamilyResult,
	port uint16, opts Options, result *Result,
) {
	if entry.Tested == "" {
		// No address for this family. For a proxied domain's AAAA that is the
		// intended outcome, so the status stands as resolveFamily left it.
		return
	}
	address, err := netip.ParseAddr(entry.Tested)
	if err != nil {
		return
	}

	// The honest SKIP. Dialling IPv6 from a host with no global IPv6 fails
	// for a reason that has nothing to do with sing-box's egress, and
	// reporting that as "unreachable" blames the config for the client.
	if entry.Family == FamilyV6 && !result.ClientV6 {
		entry.Status = ReachabilityUntested
		entry.Dial = &DialResult{Status: ReachabilityUntested,
			Error: "this host has no global IPv6 address, so IPv6 egress cannot be tested from here"}
		return
	}
	if entry.Family == FamilyV4 && !result.ClientV4 {
		entry.Status = ReachabilityUntested
		entry.Dial = &DialResult{Status: ReachabilityUntested,
			Error: "this host has no global IPv4 address, so IPv4 egress cannot be tested from here"}
		return
	}

	serverName := ""
	if opts.TLS {
		serverName = domain
	}

	dialedAt := time.Now()
	dialResult := Dial(ctx, DialRequest{
		Address:    address,
		Port:       port,
		ServerName: serverName,
		Timeout:    opts.Timeout,
		// Correlation runs while the connection is still open, because
		// /connections lists active connections only.
		Hold: func(string) {
			observed, observeErr := observe(ctx, opts.Observe, address, port, dialedAt)
			if observeErr != nil && result.ObserveError == "" {
				result.ObserveError = observeErr.Error()
			}
			entry.Observed = observed
		},
	})

	entry.Dial = &dialResult
	entry.Status = dialResult.Status
}

// observe polls the connection table until the probe's connection shows up.
//
// It polls rather than reading once because sing-box registers a connection
// when it routes it, which is a moment after the TCP handshake the panel just
// completed. A single read immediately after connecting loses the race often
// enough to be reported as "no connection observed" on a working path.
func observe(
	ctx context.Context, source Snapshotter, address netip.Addr, port uint16, dialedAt time.Time,
) (*Observed, error) {
	if source == nil {
		return nil, errors.New(
			"connections cannot be read: enable experimental.clash_api to see which rule carried the traffic")
	}

	var lastErr error
	for attempt := 0; attempt < observeAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(observeInterval):
			}
		}
		snapshot, err := source.Connections(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		if observed := Correlate(snapshot, address, port, dialedAt); observed != nil {
			return observed, nil
		}
	}
	return nil, lastErr
}

// markUntested records that nothing was dialled, for a SkipDial run.
func markUntested(result *Result, reason string) {
	for i := range result.Domains {
		for j := range result.Domains[i].Families {
			entry := &result.Domains[i].Families[j]
			if entry.Tested == "" {
				continue
			}
			entry.Status = ReachabilityUntested
			entry.Dial = &DialResult{Status: ReachabilityUntested, Error: reason}
		}
	}
}

// compare decides whether prediction and observation agree.
//
// Missing evidence on either side is AgreementUnknown, never AgreementNo: a
// disagreement is the report's headline finding and must mean what it says.
func compare(entry *FamilyResult) Agreement {
	if entry.Predicted == nil || entry.Observed == nil {
		return AgreementUnknown
	}
	if entry.Predicted.Error != "" || entry.Predicted.Outbound == "" || entry.Observed.Outbound == "" {
		return AgreementUnknown
	}
	if entry.Predicted.Outbound == entry.Observed.Outbound {
		return AgreementYes
	}
	// An inexact prediction that differs is the PREDICTION's problem, not the
	// config's — an undecidable rule ahead of the decision could have matched
	// first. Calling that a disagreement would cry wolf on every config whose
	// rule sets cannot be read.
	if !entry.Predicted.Exact {
		return AgreementUnknown
	}
	return AgreementNo
}

// NormalizeDomains validates the request's domain list.
//
// Each name is validated the way the DNS probe validates its own, because both
// send the name onward: a URL, a host:port pair or an address literal is
// rejected rather than silently coerced into something that resolves.
func NormalizeDomains(input []string) ([]string, error) {
	seen := map[string]struct{}{}
	domains := make([]string, 0, len(input))

	for _, raw := range input {
		domain := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), ".")))
		if domain == "" {
			continue
		}
		if err := validateDomain(domain); err != nil {
			return nil, fmt.Errorf("%q: %w", raw, err)
		}
		if _, duplicate := seen[domain]; duplicate {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}

	if len(domains) == 0 {
		return nil, errors.New("at least one domain is required")
	}
	if len(domains) > MaxDomains {
		return nil, fmt.Errorf("at most %d domains per request, got %d", MaxDomains, len(domains))
	}
	return domains, nil
}

func validateDomain(domain string) error {
	if len(domain) > 253 {
		return errors.New("domain is longer than 253 characters")
	}
	if strings.ContainsAny(domain, " \t/\\?#@:") {
		return errors.New("enter a bare domain name, without a scheme, port or path")
	}
	if _, err := netip.ParseAddr(domain); err == nil {
		return errors.New("enter a domain name, not an IP address")
	}
	if !strings.Contains(domain, ".") {
		return errors.New("enter a fully qualified domain, e.g. example.com")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" {
			return errors.New("domain contains an empty label")
		}
		if len(label) > 63 {
			return errors.New("domain label is longer than 63 characters")
		}
	}
	return nil
}
