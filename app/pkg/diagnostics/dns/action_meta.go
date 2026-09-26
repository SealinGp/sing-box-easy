package dns

// What a DNS rule action does to the walk.
//
// Copied from sing-box's own `Router.matchDNS` (dns/router.go), not reasoned
// about from the action names — the same discipline routeprobe/rule_meta.go
// applies to route rules, and for the same reason: a diagnostic that silently
// corrects the engine is a picture of a config that is not running.
//
// The switch in matchDNS handles exactly three actions. Everything else — and
// that includes `evaluate`, which is the backbone of a modern DNS config —
// falls through it and the loop CONTINUES to the next rule. A first-match-wins
// walk therefore reports the wrong server for any config built with them.

// dnsActionRoute is what sing-box assumes when `action` is omitted.
const dnsActionRoute = "route"

// terminalDNSActions are the actions that end rule matching.
//
// `route` is conditional and handled separately: sing-box `continue`s when the
// named transport does not exist, so a route naming a missing server matches
// and decides nothing.
var terminalDNSActions = map[string]bool{
	"reject":     true,
	"predefined": true,
}

// continuingDNSActions match, may change what the rules below them see, and
// hand over. Listed explicitly rather than inferred from "not terminal", so a
// future action is reported as unrecognised instead of silently assumed to
// continue.
var continuingDNSActions = map[string]bool{
	"route-options": true,
	// evaluate re-enters matchDNS from the rule after this one; from the
	// ladder's point of view it matched and did not decide.
	"evaluate": true,
	// respond answers from a referenced response and does not select a
	// transport, so it does not end transport selection here.
	"respond": true,
}

// dnsActionTerminates reports whether a matched rule ends the walk.
//
// `serverExists` decides the route case. A nil predicate means the server list
// could not be read, in which case a route is assumed to terminate — the
// common case by far, and the assumption that matches a working config.
func dnsActionTerminates(action, server string, serverExists func(string) bool) bool {
	if terminalDNSActions[action] {
		return true
	}
	if continuingDNSActions[action] {
		return false
	}
	if action == dnsActionRoute {
		if server == "" {
			// A route with no server names no transport to switch to.
			return false
		}
		if serverExists == nil {
			return true
		}
		return serverExists(server)
	}
	// An action this build does not recognise. Not terminating is the safe
	// direction: the walk continues and the rules below are still reported,
	// where stopping would hide them behind an action we cannot model.
	return false
}

// dnsActionEffect describes, for the UI, what a non-terminal match changed for
// the rules beneath it. Empty when there is nothing useful to say.
func dnsActionEffect(action string) string {
	switch action {
	case "route-options":
		return "applied query options and continued"
	case "evaluate":
		return "resolved through this server, then continued matching"
	case "respond":
		return "answered from a referenced response"
	}
	return ""
}
