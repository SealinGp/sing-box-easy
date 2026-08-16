package config

import C "github.com/sagernet/sing-box/constant"

// RouteRuleActionTypes lists every route rule action
// CreateRouteRuleActionOptions can construct.
//
// Same contract as DNSRuleActionTypes: the schema generator reflects over
// exactly these, and TestRouteRuleActionTypesAreRegistered fails the build if
// one is listed here without a matching case.
//
// NOT THE SAME SET AS DNSRuleActionTypes
// ──────────────────────────────────────
// option/rule_action.go declares _RuleAction (route rules) and _DNSRuleAction
// (DNS rules) side by side, and both switch on constants from the same
// C.RuleActionType* namespace — which makes them look interchangeable. They
// differ in BOTH directions:
//
//	route-only:  direct, hijack-dns, sniff, resolve
//	DNS-only:    predefined
//	shared:      route, route-options, reject
//
// A rule naming an action from the other family fails to decode.
// TestRouteAndDNSActionSetsDiffer pins it.
//
// As with DNS rules, an omitted `action` means "route" — RuleAction.UnmarshalJSON
// rewrites "" to C.RuleActionTypeRoute before dispatching.
//
// Order is the order fields appear in the generated TypeScript.
var RouteRuleActionTypes = []string{
	// Pick the outbound, and how the connection is dialed.
	C.RuleActionTypeRoute,
	// The same dial knobs, without choosing an outbound.
	C.RuleActionTypeRouteOptions,
	// Dial from here, with this rule's own dialer options.
	C.RuleActionTypeDirect,

	// Refuse the connection.
	C.RuleActionTypeReject,

	// Non-terminal actions: these annotate the connection and let matching
	// continue, rather than deciding where it goes.
	C.RuleActionTypeHijackDNS,
	C.RuleActionTypeSniff,
	C.RuleActionTypeResolve,
}

// RouteRuleMatcherTypes is the single "type" of the route rule matcher domain.
//
// Unlike every other domain the generator covers, route rule MATCHERS are not
// polymorphic: `option.RawDefaultRule` is one flat struct of ~37 conditions,
// with no discriminator selecting between variants. The generator's shape
// (a list of types plus a constructor) still fits — there is simply one entry.
//
// "default" is `C.RuleTypeDefault`, the value of a route rule's `type` field.
// The other value, "logical", is deliberately absent: a logical rule has no
// matchers of its own, only a `mode` and a nested `rules` array, so there is no
// field inventory to generate for it.
var RouteRuleMatcherTypes = []string{
	C.RuleTypeDefault,
}

// IsKnownRouteRuleAction reports whether the action can be constructed by the
// registry. Used by request validation so an unknown action is a field-level
// error rather than a raw sing-box decode string.
func IsKnownRouteRuleAction(action string) bool {
	for _, known := range RouteRuleActionTypes {
		if known == action {
			return true
		}
	}
	return false
}
