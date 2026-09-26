package dualstack

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

type fakeResolver struct {
	answers map[string][]string
	err     error
}

func (f fakeResolver) Query(name, qType string) ([]string, int64, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.answers[name+"/"+qType], 7, nil
}

func predictTo(outbound string, exact bool) Predictor {
	return func(netip.Addr, uint16) (*Prediction, error) {
		return &Prediction{Outbound: outbound, Source: "rule", MatchedIndex: 3, Exact: exact}, nil
	}
}

// runNoDial exercises everything except the network.
func runNoDial(t *testing.T, opts Options) *Result {
	t.Helper()
	opts.SkipDial = true
	result, err := Run(context.Background(), DNSEndpoint{Source: SourceNone}, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return result
}

func TestRunSeparatesTheTwoFamilies(t *testing.T) {
	result := runNoDial(t, Options{
		Domains: []string{"www.baidu.com"},
		Resolve: fakeResolver{answers: map[string][]string{
			"www.baidu.com/A":    {"110.242.68.66"},
			"www.baidu.com/AAAA": {"240e:ff:e020:9ae:0:ff:b014:8e8b"},
		}},
		Predict: predictTo("➡️ 直连", true),
	})

	families := result.Domains[0].Families
	if len(families) != 2 {
		t.Fatalf("families = %d, want 2", len(families))
	}
	if families[0].Family != FamilyV4 || families[0].QueryType != "A" {
		t.Fatalf("first family = %+v, want ipv4/A", families[0])
	}
	if families[1].Tested != "240e:ff:e020:9ae:0:ff:b014:8e8b" {
		t.Fatalf("v6 tested = %q", families[1].Tested)
	}
}

func TestRunTreatsASuppressedAAAAAsNoAddressNotFailure(t *testing.T) {
	// The single most important behaviour here. A proxied domain with its
	// AAAA suppressed is a CORRECTLY configured IPv6 split; reporting it as
	// unreachable would flag a working config as broken and train the reader
	// to ignore the column.
	result := runNoDial(t, Options{
		Domains: []string{"www.google.com"},
		Resolve: fakeResolver{answers: map[string][]string{
			"www.google.com/A": {"142.250.66.4"},
			// no AAAA entry: NOERROR with zero answers
		}},
		Predict: predictTo("🇭🇰HK", true),
	})

	v6 := result.Domains[0].Families[1]
	if v6.Status != ReachabilityNoAddress {
		t.Fatalf("status = %q, want %q", v6.Status, ReachabilityNoAddress)
	}
	if v6.DNSError != "" {
		t.Fatalf("a suppressed record is not an error, got %q", v6.DNSError)
	}
	if v6.Dial != nil {
		t.Fatal("nothing should be dialled for a family with no address")
	}
}

func TestRunDistinguishesAResolverFailureFromNoAddress(t *testing.T) {
	// "sing-box could not be asked" and "sing-box said there is no AAAA" look
	// identical in a two-state report and have opposite fixes.
	result := runNoDial(t, Options{
		Domains: []string{"www.google.com"},
		Resolve: fakeResolver{err: errors.New("clash api unreachable")},
		Predict: predictTo("x", true),
	})

	for _, family := range result.Domains[0].Families {
		if family.Status != ReachabilityUntested {
			t.Fatalf("%s status = %q, want %q", family.Family, family.Status, ReachabilityUntested)
		}
		if family.DNSError == "" {
			t.Fatalf("%s should carry the resolver's error", family.Family)
		}
	}
}

func TestRunWithoutAResolverIsUntested(t *testing.T) {
	result := runNoDial(t, Options{Domains: []string{"www.google.com"}})
	if result.Domains[0].Families[0].Status != ReachabilityUntested {
		t.Fatalf("status = %q", result.Domains[0].Families[0].Status)
	}
}

func TestRunFiltersAnAnswerOfTheWrongFamily(t *testing.T) {
	// An IPv4-mapped address in the AAAA row would read as working IPv6 and
	// is not — the same confusion that makes `curl -6` lie.
	result := runNoDial(t, Options{
		Domains: []string{"www.google.com"},
		Resolve: fakeResolver{answers: map[string][]string{
			"www.google.com/AAAA": {"::ffff:142.250.66.4"},
		}},
	})

	v6 := result.Domains[0].Families[1]
	if len(v6.Addresses) != 0 {
		t.Fatalf("addresses = %v, want the mapped address rejected", v6.Addresses)
	}
}

type fakeSnapshotter struct {
	snapshot *clashapi.Snapshot
	err      error
	calls    int
}

func (f *fakeSnapshotter) Connections(context.Context) (*clashapi.Snapshot, error) {
	f.calls++
	return f.snapshot, f.err
}

func TestCompareFlagsADisagreement(t *testing.T) {
	entry := &FamilyResult{
		Predicted: &Prediction{Outbound: "➡️ 直连", Exact: true},
		Observed:  &Observed{Outbound: "🇭🇰HK"},
	}
	if got := compare(entry); got != AgreementNo {
		t.Fatalf("agreement = %q, want %q", got, AgreementNo)
	}
}

func TestCompareAgreesWhenTheyMatch(t *testing.T) {
	entry := &FamilyResult{
		Predicted: &Prediction{Outbound: "➡️ 直连", Exact: true},
		Observed:  &Observed{Outbound: "➡️ 直连"},
	}
	if got := compare(entry); got != AgreementYes {
		t.Fatalf("agreement = %q, want %q", got, AgreementYes)
	}
}

func TestCompareWillNotBlameAnInexactPrediction(t *testing.T) {
	// An undecidable rule ahead of the decision could have matched first, so a
	// mismatch is the prediction's fault, not the config's. Reporting it as a
	// disagreement would cry wolf on every config whose rule sets are
	// unreadable — which, before the binary fallback, was most of them.
	entry := &FamilyResult{
		Predicted: &Prediction{Outbound: "➡️ 直连", Exact: false},
		Observed:  &Observed{Outbound: "🇭🇰HK"},
	}
	if got := compare(entry); got != AgreementUnknown {
		t.Fatalf("agreement = %q, want %q", got, AgreementUnknown)
	}
}

func TestCompareNeedsBothSides(t *testing.T) {
	if got := compare(&FamilyResult{Predicted: &Prediction{Outbound: "x", Exact: true}}); got != AgreementUnknown {
		t.Fatalf("agreement = %q, want unknown with no observation", got)
	}
	if got := compare(&FamilyResult{Observed: &Observed{Outbound: "x"}}); got != AgreementUnknown {
		t.Fatalf("agreement = %q, want unknown with no prediction", got)
	}
}

func TestNormalizeDomainsRejectsWhatItCannotSend(t *testing.T) {
	for _, bad := range []string{
		"https://example.com", "example.com:443", "1.2.3.4", "localhost",
		"exa mple.com", "example..com",
	} {
		if _, err := NormalizeDomains([]string{bad}); err == nil {
			t.Fatalf("%q was accepted", bad)
		}
	}
}

func TestNormalizeDomainsDeduplicatesAndLowercases(t *testing.T) {
	domains, err := NormalizeDomains([]string{"WWW.Example.COM", " www.example.com. ", "b.example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 2 || domains[0] != "www.example.com" {
		t.Fatalf("domains = %v", domains)
	}
}

func TestNormalizeDomainsEnforcesTheCap(t *testing.T) {
	// The endpoint makes the SERVER dial; the bound is explicit rather than
	// whatever the caller happens to send.
	many := make([]string, 0, MaxDomains+1)
	for i := 0; i <= MaxDomains; i++ {
		many = append(many, strings.Repeat("a", i+1)+".example.com")
	}
	if _, err := NormalizeDomains(many); err == nil {
		t.Fatal("expected the domain cap to be enforced")
	}
}

func TestNormalizeDomainsRequiresOne(t *testing.T) {
	if _, err := NormalizeDomains([]string{"  ", ""}); err == nil {
		t.Fatal("expected an error for an empty list")
	}
}

func TestRunStagedReportsEveryStage(t *testing.T) {
	var stages []Stage
	_, err := RunStaged(context.Background(), DNSEndpoint{Source: SourceNone}, Options{
		Domains:  []string{"www.example.com"},
		SkipDial: true,
		Resolve:  fakeResolver{answers: map[string][]string{"www.example.com/A": {"1.2.3.4"}}},
		Predict:  predictTo("direct", true),
	}, func(stage Stage, _ *Result) error {
		stages = append(stages, stage)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []Stage{StageEndpoint, StageResolve, StagePredict, StageTraffic}
	if len(stages) != len(want) {
		t.Fatalf("stages = %v, want %v", stages, want)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Fatalf("stages = %v, want %v", stages, want)
		}
	}
}

func TestRunStagedAbortsWhenTheClientLeaves(t *testing.T) {
	// The remaining work opens real connections. Continuing for a browser tab
	// that closed is the leak the log follower's comments warn about.
	called := 0
	result, err := RunStaged(context.Background(), DNSEndpoint{Source: SourceNone}, Options{
		Domains: []string{"www.example.com"},
		Resolve: fakeResolver{answers: map[string][]string{"www.example.com/A": {"1.2.3.4"}}},
		Predict: func(netip.Addr, uint16) (*Prediction, error) {
			t.Fatal("prediction ran after the stream was abandoned")
			return nil, nil
		},
	}, func(Stage, *Result) error {
		called++
		return errors.New("client gone")
	})
	if err != nil {
		t.Fatalf("an abort is not an error: %v", err)
	}
	if called != 1 {
		t.Fatalf("stage callback ran %d times after aborting", called)
	}
	if result == nil {
		t.Fatal("the partial result should still be returned")
	}
}
