package config

import (
	"context"
	"reflect"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// TestRouteRuleActionTypesAreRegistered guards the list/registry consistency,
// same contract as TestDNSRuleActionTypesAreRegistered.
func TestRouteRuleActionTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(RouteRuleActionTypes) == 0 {
		t.Fatal("RouteRuleActionTypes is empty")
	}

	seen := make(map[string]bool, len(RouteRuleActionTypes))
	for _, action := range RouteRuleActionTypes {
		if seen[action] {
			t.Errorf("duplicate entry %q in RouteRuleActionTypes", action)
		}
		seen[action] = true

		options, ok := r.CreateRouteRuleActionOptions(action)
		if !ok {
			t.Errorf("CreateRouteRuleActionOptions(%q) = not registered", action)
			continue
		}

		// hijack-dns has no options struct at all — RuleAction.MarshalJSON sets
		// v = nil for it — so the registry returns a typed nil marker rather
		// than a struct. Every other action must be a pointer to a struct.
		if action == C.RuleActionTypeHijackDNS {
			continue
		}

		rt := reflect.TypeOf(options)
		if rt == nil || rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("CreateRouteRuleActionOptions(%q) returned %v; want pointer to struct", action, rt)
		}

		if !IsKnownRouteRuleAction(action) {
			t.Errorf("IsKnownRouteRuleAction(%q) = false for a listed type", action)
		}
	}
}

// TestRouteAndDNSActionSetsDiffer pins the distinction that makes two separate
// lists necessary.
//
// _RuleAction (route) and _DNSRuleAction (DNS) sit side by side in
// option/rule_action.go and switch on constants from the same
// C.RuleActionType* namespace, which makes them look interchangeable. The value
// SETS differ in both directions, and using the wrong one is a rule that cannot
// decode.
func TestRouteAndDNSActionSetsDiffer(t *testing.T) {
	r := &Registry{}

	// Route-only: a DNS rule naming one fails with "unknown DNS rule action".
	for _, routeOnly := range []string{
		C.RuleActionTypeDirect,
		C.RuleActionTypeHijackDNS,
		C.RuleActionTypeSniff,
		C.RuleActionTypeResolve,
	} {
		if !IsKnownRouteRuleAction(routeOnly) {
			t.Errorf("IsKnownRouteRuleAction(%q) = false; it is a route action", routeOnly)
		}
		if IsKnownDNSRuleAction(routeOnly) {
			t.Errorf("IsKnownDNSRuleAction(%q) = true; it is route-only", routeOnly)
		}
	}

	// DNS-only: a route rule naming it fails with "unknown rule action".
	if IsKnownRouteRuleAction(C.RuleActionTypePredefined) {
		t.Error("IsKnownRouteRuleAction(predefined) = true; it is DNS-only")
	}
	if _, ok := r.CreateRouteRuleActionOptions(C.RuleActionTypePredefined); ok {
		t.Error("CreateRouteRuleActionOptions(predefined) = registered; it is DNS-only")
	}
}

// TestRouteRuleMatcherTypesAreRegistered covers the single-type matcher domain.
func TestRouteRuleMatcherTypesAreRegistered(t *testing.T) {
	r := &Registry{}

	if len(RouteRuleMatcherTypes) != 1 {
		t.Fatalf("RouteRuleMatcherTypes has %d entries; want exactly 1 (the matchers are not polymorphic)", len(RouteRuleMatcherTypes))
	}

	for _, matcherType := range RouteRuleMatcherTypes {
		options, ok := r.CreateRouteRuleMatcherOptions(matcherType)
		if !ok {
			t.Fatalf("CreateRouteRuleMatcherOptions(%q) = not registered", matcherType)
		}
		if _, isRaw := options.(*option.RawDefaultRule); !isRaw {
			t.Errorf("CreateRouteRuleMatcherOptions(%q) returned %T; want *option.RawDefaultRule", matcherType, options)
		}
	}

	if _, ok := r.CreateRouteRuleMatcherOptions("logical"); ok {
		t.Error(`CreateRouteRuleMatcherOptions("logical") = registered; a logical rule has no matchers of its own, only nested rules`)
	}
}

// TestRouteRuleActionRoundTrip parses one rule per action through the same JSON
// path the handlers use, then marshals it back.
func TestRouteRuleActionRoundTrip(t *testing.T) {
	cases := map[string]string{
		C.RuleActionTypeRoute: `{"domain":["a.example"],"outbound":"direct",` +
			`"override_address":"1.2.3.4","override_port":443,"udp_connect":true,` +
			`"udp_timeout":"5m","tls_fragment":true,"network_strategy":"hybrid"}`,
		C.RuleActionTypeRouteOptions: `{"domain":["a.example"],"action":"route-options",` +
			`"udp_disable_domain_unmapping":true,"tls_record_fragment":true}`,
		C.RuleActionTypeDirect: `{"domain":["a.example"],"action":"direct",` +
			`"bind_interface":"en0","tcp_fast_open":true,"network_type":["wifi"]}`,
		C.RuleActionTypeReject:    `{"domain":["a.example"],"action":"reject","method":"default","no_drop":true}`,
		C.RuleActionTypeHijackDNS: `{"protocol":["dns"],"action":"hijack-dns"}`,
		C.RuleActionTypeSniff:     `{"inbound":["in"],"action":"sniff","sniffer":["http","tls"],"timeout":"300ms"}`,
		C.RuleActionTypeResolve: `{"domain":["a.example"],"action":"resolve","strategy":"ipv4_only",` +
			`"disable_cache":true,"rewrite_ttl":60}`,
	}

	ctx := CreateContext(context.Background())

	for action, raw := range cases {
		t.Run(action, func(t *testing.T) {
			var rule option.Rule
			if err := json.UnmarshalContext(ctx, []byte(raw), &rule); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			got := rule.DefaultOptions.Action
			// An omitted action means "route" — RuleAction.UnmarshalJSON
			// rewrites "" before dispatching.
			if got != action {
				t.Errorf("Action = %q, want %q", got, action)
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

// TestDirectActionRejectsDetour pins the reason `detour` is excluded from the
// generated inventory for the `direct` action.
//
// DirectActionOptions IS DialerOptions, so reflection finds `detour` on it — but
// sing-box refuses to decode it here. A form offering it would produce a save
// that always fails, so the generator drops it via domain.ExcludedFields.
func TestDirectActionRejectsDetour(t *testing.T) {
	ctx := CreateContext(context.Background())

	var rule option.Rule
	raw := `{"domain":["a.example"],"action":"direct","detour":"somewhere"}`
	err := json.UnmarshalContext(ctx, []byte(raw), &rule)
	if err == nil {
		t.Fatal("a direct action with detour decoded successfully; the ExcludedFields entry may no longer be needed")
	}
}

// TestRouteRuleMatcherRoundTripPreservesShapes pins the wire formats the
// generated inventory has to get right. Each of these is a field whose Go type
// misleads: ports are uint16 (NOT strings, so "8080-8090" belongs in
// port_range), user_id is int32, and network_type is a uint8 enum that
// marshals as a name.
func TestRouteRuleMatcherRoundTripPreservesShapes(t *testing.T) {
	ctx := CreateContext(context.Background())

	var rule option.Rule
	raw := `{"port":[443,8080],"port_range":["8080:8090"],"user_id":[1000],` +
		`"network_type":["wifi","cellular"],"ip_version":4,"clash_mode":"Direct",` +
		`"ip_cidr":["10.0.0.0/8"],"invert":true,"outbound":"direct"}`
	if err := json.UnmarshalContext(ctx, []byte(raw), &rule); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	matchers := rule.DefaultOptions.RawDefaultRule
	if len(matchers.Port) != 2 || matchers.Port[0] != 443 {
		t.Errorf("Port = %v, want [443 8080]", matchers.Port)
	}
	if len(matchers.PortRange) != 1 || matchers.PortRange[0] != "8080:8090" {
		t.Errorf("PortRange = %v, want [8080:8090]", matchers.PortRange)
	}
	if len(matchers.NetworkType) != 2 {
		t.Errorf("NetworkType = %v, want 2 entries", matchers.NetworkType)
	}
	if matchers.IPVersion != 4 {
		t.Errorf("IPVersion = %d, want 4", matchers.IPVersion)
	}
	if !matchers.Invert {
		t.Error("Invert = false, want true")
	}

	// A port RANGE in `port` is a hard decode failure — the form's placeholder
	// advertised "8080-8090" for this field, which could never be saved.
	var bad option.Rule
	if err := json.UnmarshalContext(ctx, []byte(`{"port":["8080-8090"],"outbound":"direct"}`), &bad); err == nil {
		t.Error(`port:["8080-8090"] decoded; expected a uint16 failure — ranges belong in port_range as "8080:8090"`)
	}
}
