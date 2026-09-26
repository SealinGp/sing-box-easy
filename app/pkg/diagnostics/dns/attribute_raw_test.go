package dns

import (
	"encoding/json"
	"testing"
)

// attribute walks a DNS section given as raw JSON — the form a config is
// actually stored in, and the only form that survives a host running a newer
// sing-box than this panel is compiled against.
func attribute(t *testing.T, section string, query Query) Attribution {
	t.Helper()
	if !json.Valid([]byte(section)) {
		t.Fatalf("fixture is not valid JSON: %s", section)
	}
	return AttributeRaw(json.RawMessage(section), query)
}

// A 1.14 DNS section: `evaluate`, `respond`, `match_response`, `race` and
// `optimistic` all appear, and none of them exists in the pinned 1.12.12
// schema. Before the raw walk this produced an empty ladder — zero rules out
// of however many the config had — and the whole attribution was disabled.
const section114 = `{
  "servers":[{"type":"udp","tag":"local","server":"223.5.5.5"}],
  "optimistic": true,
  "rules":[
    {"domain_suffix":["owolist.cn"],"action":"evaluate","server":"dns_router","tag":"d1","race":true},
    {"match_response":"d1","response_rcode":"NXDOMAIN","action":"respond"},
    {"domain":["blocked.test"],"action":"reject"},
    {"domain_suffix":["owolist.cn"],"action":"route","server":"local"}
  ],
  "final":"local"
}`

func TestAttributeRawWalksA114Section(t *testing.T) {
	got := attribute(t, section114, Query{Domain: "www.owolist.cn"})

	if len(got.Rules) != 4 {
		t.Fatalf("rules = %d, want 4 — the ladder must not be empty on a 1.14 config", len(got.Rules))
	}
}

func TestAttributeRawEvaluateDoesNotTerminate(t *testing.T) {
	// dns/router.go's matchDNS switch handles only route, reject and
	// predefined. `evaluate` matches, is logged, and falls through the switch
	// — so the walk CONTINUES. Treating it as terminal (which first-match-wins
	// does) reports the wrong server for every config built this way.
	got := attribute(t, section114, Query{Domain: "www.owolist.cn"})

	if got.Rules[0].State != MatchStateMatched {
		t.Fatalf("rule 0 state = %q, want matched", got.Rules[0].State)
	}
	if got.Rules[0].Terminal {
		t.Fatal("evaluate must not terminate the walk")
	}
	if got.MatchedIndex != 3 {
		t.Fatalf("matched = %d, want 3 — the route rule decides, not the evaluate", got.MatchedIndex)
	}
	if got.Server != "local" {
		t.Fatalf("server = %q, want local", got.Server)
	}
}

func TestAttributeRawRejectTerminates(t *testing.T) {
	got := attribute(t, section114, Query{Domain: "blocked.test"})
	if got.MatchedIndex != 2 {
		t.Fatalf("matched = %d, want 2", got.MatchedIndex)
	}
	if !got.Rules[2].Terminal {
		t.Fatal("reject must terminate")
	}
	if got.FinalUsed {
		t.Fatal("a terminal match must not fall through to dns.final")
	}
}

func TestAttributeRawRouteOptionsDoesNotTerminate(t *testing.T) {
	// route-options matches, mutates the query options and continues
	// (dns/router.go: the RuleActionDNSRouteOptions case has no return).
	section := `{"rules":[
	  {"domain_suffix":["a.test"],"action":"route-options","disable_cache":true},
	  {"domain_suffix":["a.test"],"action":"route","server":"local"}
	],"final":"fallback"}`

	got := attribute(t, section, Query{Domain: "x.a.test"})
	if got.Rules[0].State != MatchStateMatched || got.Rules[0].Terminal {
		t.Fatalf("rule 0 = %+v, want a non-terminal match", got.Rules[0])
	}
	if got.MatchedIndex != 1 || got.Server != "local" {
		t.Fatalf("matched = %d server = %q, want 1/local", got.MatchedIndex, got.Server)
	}
}

func TestAttributeRawUnknownConditionIsUndecidableNotIgnored(t *testing.T) {
	// The safety property. A condition key this build has never heard of must
	// make the rule undecidable — silently dropping it would turn a
	// conditional rule into one that matches everything, and report a
	// confident, wrong server.
	section := `{"rules":[
	  {"some_future_condition":["x"],"action":"route","server":"wrong"}
	],"final":"right"}`

	got := attribute(t, section, Query{Domain: "example.com"})
	if got.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", got.Rules[0].State)
	}
	if got.Exact {
		t.Fatal("an undecidable rule ahead of the decision must clear Exact")
	}
	if !contains(got.Rules[0].Unevaluated, "some_future_condition") {
		t.Fatalf("unevaluated = %v, want the unknown key named", got.Rules[0].Unevaluated)
	}
}

func TestAttributeRawKnownActionOptionsAreNotConditions(t *testing.T) {
	// The counterweight to the test above. Action options are FLAT alongside
	// conditions in a DNS rule, so a walk that treated every unrecognised key
	// as a condition would mark every 1.14 rule undecidable and achieve
	// nothing. These keys carry the action, not a condition.
	section := `{"rules":[
	  {"domain":["a.test"],"action":"route","server":"s","strategy":"prefer_ipv4",
	   "disable_cache":true,"disable_optimistic_cache":true,"rewrite_ttl":60,
	   "client_subnet":"1.2.3.0/24","remove_client_subnet":true,"timeout":"5s",
	   "speculative":true,"race":true,"tag":"t1"}
	],"final":"fallback"}`

	got := attribute(t, section, Query{Domain: "a.test"})
	if got.Rules[0].State != MatchStateMatched {
		t.Fatalf("state = %q (unevaluated: %v), want matched",
			got.Rules[0].State, got.Rules[0].Unevaluated)
	}
	if !got.Exact {
		t.Fatalf("walk should be exact, unevaluated = %v", got.Rules[0].Unevaluated)
	}
}

func TestAttributeRawKnownRuntimeConditionsStayUndecidable(t *testing.T) {
	// 1.14 added a pile of conditions that need runtime state. They must be
	// named individually so the UI can say WHICH one blocked the decision.
	for _, key := range []string{
		"query_dnssec", "source_hostname", "source_mac_address",
		"interface_address", "preferred_by", "match_response", "response_rcode",
	} {
		section := `{"rules":[{"` + key + `":true,"action":"route","server":"s"}],"final":"f"}`
		got := attribute(t, section, Query{Domain: "a.test"})
		if got.Rules[0].State != MatchStateUnevaluated {
			t.Fatalf("%s: state = %q, want unevaluated", key, got.Rules[0].State)
		}
		if !contains(got.Rules[0].Unevaluated, key) {
			t.Fatalf("%s: unevaluated = %v", key, got.Rules[0].Unevaluated)
		}
	}
}

func TestAttributeRawLogicalAnd(t *testing.T) {
	// The shape the user's own config uses for AAAA suppression:
	// query_type AND rule_set. Reported wholesale as "logical rule,
	// unevaluated" before, which made every config using one inexact.
	section := `{"rules":[{
	  "type":"logical","mode":"and","rules":[
	    {"query_type":["AAAA","HTTPS"]},
	    {"domain_suffix":["google.com"]}
	  ],"action":"predefined","rcode":"NOERROR"}],"final":"local"}`

	hit := attribute(t, section, Query{Domain: "www.google.com", Type: "AAAA"})
	if hit.MatchedIndex != 0 {
		t.Fatalf("matched = %d, want 0 (both branches match)", hit.MatchedIndex)
	}

	// One branch fails => the AND fails, decisively, not "unevaluated".
	miss := attribute(t, section, Query{Domain: "www.google.com", Type: "A"})
	if miss.Rules[0].State != MatchStateNotMatched {
		t.Fatalf("state = %q, want not_matched", miss.Rules[0].State)
	}
	if !miss.FinalUsed || !miss.Exact {
		t.Fatalf("expected an exact fall-through, got %+v", miss)
	}
}

func TestAttributeRawLogicalOr(t *testing.T) {
	section := `{"rules":[{
	  "type":"logical","mode":"or","rules":[
	    {"domain_suffix":["a.test"]},
	    {"domain_suffix":["b.test"]}
	  ],"action":"route","server":"hit"}],"final":"miss"}`

	if got := attribute(t, section, Query{Domain: "x.b.test"}); got.Server != "hit" {
		t.Fatalf("server = %q, want hit", got.Server)
	}
	if got := attribute(t, section, Query{Domain: "x.c.test"}); got.Server != "miss" {
		t.Fatalf("server = %q, want miss", got.Server)
	}
}

func TestAttributeRawLogicalOrIsUndecidableWhenABranchIs(t *testing.T) {
	// OR short-circuits on a hit, so an undecidable branch only matters when
	// nothing else matched. Getting this backwards makes every logical rule
	// undecidable and defeats the point of walking into them.
	section := `{"rules":[{
	  "type":"logical","mode":"or","rules":[
	    {"domain_suffix":["a.test"]},
	    {"source_hostname":["nas"]}
	  ],"action":"route","server":"hit"}],"final":"miss"}`

	decided := attribute(t, section, Query{Domain: "x.a.test"})
	if decided.Rules[0].State != MatchStateMatched {
		t.Fatalf("a matching branch settles the OR, got %q", decided.Rules[0].State)
	}

	open := attribute(t, section, Query{Domain: "x.c.test"})
	if open.Rules[0].State != MatchStateUnevaluated {
		t.Fatalf("state = %q, want unevaluated", open.Rules[0].State)
	}
}

func TestAttributeRawInvert(t *testing.T) {
	section := `{"rules":[
	  {"domain_suffix":["a.test"],"invert":true,"action":"route","server":"other"}
	],"final":"fallback"}`

	if got := attribute(t, section, Query{Domain: "x.a.test"}); !got.FinalUsed {
		t.Fatalf("an inverted match must not match, got index %d", got.MatchedIndex)
	}
	if got := attribute(t, section, Query{Domain: "x.b.test"}); got.Server != "other" {
		t.Fatalf("server = %q, want other", got.Server)
	}
}

func TestAttributeRawTolerantOfGarbage(t *testing.T) {
	// Third-party config text reached from an HTTP handler. It must degrade,
	// never panic.
	for _, bad := range []string{`{}`, `null`, `{"rules":"nope"}`, `{"rules":[null,3,"x"]}`, `[]`} {
		got := AttributeRaw(json.RawMessage(bad), Query{Domain: "a.test"})
		if got.Rules == nil {
			t.Fatalf("%s: Rules must never be nil", bad)
		}
	}
}

func TestAttributeRawReportsRouteToAMissingServer(t *testing.T) {
	// sing-box `continue`s when a route action names a transport that does not
	// exist (dns/router.go: "transport not found"), so the rule matches and
	// decides NOTHING. A walk that stopped there would report a server the
	// query never reaches.
	section := `{"servers":[{"type":"udp","tag":"real","server":"1.1.1.1"}],
	  "rules":[
	    {"domain":["a.test"],"action":"route","server":"ghost"},
	    {"domain":["a.test"],"action":"route","server":"real"}
	  ],"final":"fallback"}`

	got := attribute(t, section, Query{Domain: "a.test"})
	if got.MatchedIndex != 1 || got.Server != "real" {
		t.Fatalf("matched = %d server = %q, want 1/real", got.MatchedIndex, got.Server)
	}
	if got.Rules[0].Terminal {
		t.Fatal("a route to a non-existent server does not terminate")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
