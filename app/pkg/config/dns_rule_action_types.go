package config

import C "github.com/sagernet/sing-box/constant"

// DNSRuleActionTypes lists every DNS rule action CreateDNSRuleActionOptions can
// construct.
//
// Same contract as DNSTypes: the schema generator reflects over exactly these,
// and TestDNSRuleActionTypesAreRegistered fails the build if one is listed here
// without a matching case.
//
// THE DISCRIMINATOR IS `action`, NOT `type`
// ─────────────────────────────────────────
// Every other domain the generator covers keys its option struct off a `type`
// field. A DNS rule instead carries `type` for its MATCHER shape ("default" or
// "logical") and `action` for what to do once it matches. The two are
// independent: a logical rule has an action just like a default one does.
//
// WHY THIS IS NOT THE ROUTE-RULE ACTION LIST
// ──────────────────────────────────────────
// option/rule_action.go declares _RuleAction (route rules) and _DNSRuleAction
// (DNS rules) side by side, and both switch on constants from the same
// C.RuleActionType* namespace — which makes them look interchangeable. They are
// not. `direct`, `hijack-dns`, `sniff` and `resolve` are route-rule actions
// only; a DNS rule naming one fails to decode with "unknown DNS rule action".
// TestRouteOnlyActionsAreNotDNSActions pins this.
//
// An omitted `action` means "route" — DNSRuleAction.UnmarshalJSONContext
// (option/rule_action.go) rewrites "" to C.RuleActionTypeRoute before
// dispatching, so a rule with no action is a route rule and the form must show
// it as one.
//
// Order is the order fields appear in the generated TypeScript — keep related
// actions adjacent, it is the diff people read on a sing-box upgrade.
var DNSRuleActionTypes = []string{
	// Pick the DNS server that answers, and how the answer is treated.
	C.RuleActionTypeRoute,
	// The same treatment knobs, without changing which server answers.
	C.RuleActionTypeRouteOptions,

	// Answer nothing.
	C.RuleActionTypeReject,
	// Answer this, without asking any server.
	C.RuleActionTypePredefined,
}

// IsKnownDNSRuleAction reports whether the action can be constructed by the
// registry. Used by request validation so an unknown action is a field-level
// error rather than a raw sing-box decode string.
func IsKnownDNSRuleAction(action string) bool {
	for _, known := range DNSRuleActionTypes {
		if known == action {
			return true
		}
	}
	return false
}
