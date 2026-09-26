package config

import (
	"context"
	"reflect"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// TestDNSRuleActionTypesAreRegistered 守住 DNSRuleActionTypes 与
// CreateDNSRuleActionOptions 的一致性, 与 TestDNSTypesAreRegistered 同一契约。
func TestDNSRuleActionTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(DNSRuleActionTypes) == 0 {
		t.Fatal("DNSRuleActionTypes is empty")
	}

	seen := make(map[string]bool, len(DNSRuleActionTypes))
	for _, action := range DNSRuleActionTypes {
		if seen[action] {
			t.Errorf("duplicate entry %q in DNSRuleActionTypes", action)
		}
		seen[action] = true

		options, ok := r.CreateDNSRuleActionOptions(action)
		if !ok {
			t.Errorf("CreateDNSRuleActionOptions(%q) = not registered", action)
			continue
		}

		rt := reflect.TypeOf(options)
		if rt == nil || rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("CreateDNSRuleActionOptions(%q) returned %v; want pointer to struct", action, rt)
		}

		if !IsKnownDNSRuleAction(action) {
			t.Errorf("IsKnownDNSRuleAction(%q) = false for a listed type", action)
		}
	}
}

// TestRouteOnlyActionsAreNotDNSActions pins the distinction that makes this list
// necessary at all.
//
// option/rule_action.go declares _RuleAction (route rules) and _DNSRuleAction
// (DNS rules) side by side, and both switch on constants from the same
// constant.RuleActionType* namespace. But the value SETS differ: `direct`,
// `hijack-dns`, `sniff` and `resolve` exist only for route rules, and a DNS rule
// naming one fails to decode with "unknown DNS rule action". Offering one in the
// DNS rule form would be a save that cannot succeed.
func TestRouteOnlyActionsAreNotDNSActions(t *testing.T) {
	r := &Registry{}

	for _, action := range []string{
		C.RuleActionTypeDirect,
		C.RuleActionTypeHijackDNS,
		C.RuleActionTypeSniff,
		C.RuleActionTypeResolve,
	} {
		if _, ok := r.CreateDNSRuleActionOptions(action); ok {
			t.Errorf("CreateDNSRuleActionOptions(%q) = registered; it is a route-rule action only", action)
		}
		if IsKnownDNSRuleAction(action) {
			t.Errorf("IsKnownDNSRuleAction(%q) = true; it is a route-rule action only", action)
		}
	}
}

// TestDNSRuleActionRoundTrip parses one rule per action through the same context
// the config manager uses, then marshals it back.
//
// This is the test that pins the generated inventory to reality. Two fields
// carry a Go type whose underlying kind lies about the wire format —
// option.DNSRCode is an int that marshals as "NXDOMAIN", and DNSRecordOptions
// embeds dns.RR but marshals as an RR string — so a generator that classified
// them by reflect.Kind alone would render a number spinner and a JSON textarea
// for values that are plainly strings.
func TestDNSRuleActionRoundTrip(t *testing.T) {
	cases := map[string]string{
		C.RuleActionTypeRoute: `{"domain":["a.example"],"action":"route",` +
			`"server":"dns-remote","strategy":"ipv4_only","disable_cache":true,` +
			`"rewrite_ttl":60,"client_subnet":"1.2.3.0/24"}`,
		C.RuleActionTypeRouteOptions: `{"domain":["a.example"],"action":"route-options",` +
			`"strategy":"prefer_ipv4","disable_cache":true,"rewrite_ttl":30}`,
		C.RuleActionTypeReject: `{"domain":["a.example"],"action":"reject","method":"default","no_drop":true}`,
		C.RuleActionTypePredefined: `{"domain":["a.example"],"action":"predefined","rcode":"NXDOMAIN",` +
			`"answer":["a.example. 3600 IN A 192.0.2.1"]}`,
	}

	ctx := CreateContext(context.Background())

	for action, raw := range cases {
		t.Run(action, func(t *testing.T) {
			var rule option.DNSRule
			if err := json.UnmarshalContext(ctx, []byte(raw), &rule); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if rule.DefaultOptions.Action != action {
				t.Errorf("Action = %q, want %q", rule.DefaultOptions.Action, action)
			}

			out, err := json.MarshalContext(ctx, rule)
			if err != nil {
				t.Fatalf("marshal back: %v", err)
			}
			if len(out) == 0 {
				t.Fatal("marshalled to nothing")
			}
		})
	}
}

// TestDNSRuleActionRoundTripPreservesFields guards the data loss the schema form
// is meant to end: every field below is one the hand-written form had no control
// for, and several were destroyed by opening a rule and pressing Update.
func TestDNSRuleActionRoundTripPreservesFields(t *testing.T) {
	ctx := CreateContext(context.Background())

	var rule option.DNSRule
	raw := `{"domain":["a.example"],"action":"route","server":"dns-remote",` +
		`"strategy":"ipv4_only","disable_cache":true,"rewrite_ttl":60,"client_subnet":"1.2.3.0/24"}`
	if err := json.UnmarshalContext(ctx, []byte(raw), &rule); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	route := rule.DefaultOptions.RouteOptions
	if route.Server != "dns-remote" {
		t.Errorf("Server = %q, want dns-remote", route.Server)
	}
	if route.Strategy.String() != "ipv4_only" {
		t.Errorf("Strategy = %q, want ipv4_only", route.Strategy.String())
	}
	if !route.DisableCache {
		t.Error("DisableCache = false, want true")
	}
	if route.RewriteTTL == nil || *route.RewriteTTL != 60 {
		t.Errorf("RewriteTTL = %v, want 60", route.RewriteTTL)
	}
	if route.ClientSubnet == nil {
		t.Error("ClientSubnet = nil, want 1.2.3.0/24")
	}
}

// TestPredefinedAnswerIsAString pins the wire format the generator must emit for
// answer/ns/extra. DNSRecordOptions embeds dns.RR — an interface — so reflection
// alone would classify the list element as an object and the form would offer a
// JSON textarea. sing-box reads and writes a plain RR string.
func TestPredefinedAnswerIsAString(t *testing.T) {
	ctx := CreateContext(context.Background())

	var rule option.DNSRule
	raw := `{"domain":["a.example"],"action":"predefined","rcode":"NOERROR",` +
		`"answer":["a.example. 3600 IN A 192.0.2.1"],"ns":[],"extra":[]}`
	if err := json.UnmarshalContext(ctx, []byte(raw), &rule); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	predefined := rule.DefaultOptions.PredefinedOptions
	if len(predefined.Answer) != 1 {
		t.Fatalf("Answer has %d entries, want 1", len(predefined.Answer))
	}

	out, err := json.MarshalContext(ctx, rule)
	if err != nil {
		t.Fatalf("marshal back: %v", err)
	}

	var back map[string]any
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("re-parse: %v", err)
	}

	// badoption.Listable collapses a single-entry list to a bare scalar, so the
	// wire format is scalar-or-array — exactly like the domain/rule_set matchers
	// the conditions form already coerces with toArrayField. The frontend must
	// normalise on load or a one-answer rule round-trips into a chips box holding
	// one character per array index.
	switch answer := back["answer"].(type) {
	case string:
	case []any:
		if _, ok := answer[0].(string); !ok {
			t.Errorf("answer[0] marshalled as %T, want a string", answer[0])
		}
	default:
		t.Fatalf("answer marshalled as %T, want a string or an array of strings", answer)
	}

	// An omitted rcode means NOERROR. The form must not seed NXDOMAIN on open.
	if _, present := back["rcode"]; present && back["rcode"] != "NOERROR" {
		t.Errorf("rcode = %v, want NOERROR or absent", back["rcode"])
	}
}
