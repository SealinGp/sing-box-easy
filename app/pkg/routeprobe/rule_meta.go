package routeprobe

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// actionName reports a rule's action, defaulting to "route" — what sing-box
// assumes when the field is omitted. The panel deliberately does not write that
// default back into the config, so it has to be applied on every read.
func actionName(rule option.Rule) string {
	action := ruleAction(rule).Action
	if action == "" {
		return C.RuleActionTypeRoute
	}
	return action
}

func ruleType(rule option.Rule) string {
	if rule.Type == C.RuleTypeLogical {
		return C.RuleTypeLogical
	}
	return C.RuleTypeDefault
}

// ruleAction pulls the action block out of either rule shape.
func ruleAction(rule option.Rule) option.RuleAction {
	if rule.Type == C.RuleTypeLogical {
		return rule.LogicalOptions.RuleAction
	}
	return rule.DefaultOptions.RuleAction
}

// isTerminal reports whether an action stops rule matching.
//
// Only route, reject and hijack-dns do (route/route.go:478-484). Notably
// `direct` does NOT: route/route.go has no case for it, so a rule carrying it
// falls through to the rules below. That looks like an upstream oversight, but
// a diagnostic that silently corrects the engine would be lying about the
// config in force, so it is mirrored exactly.
func isTerminal(action string) bool {
	switch action {
	case C.RuleActionTypeRoute, C.RuleActionTypeReject, C.RuleActionTypeHijackDNS:
		return true
	default:
		return false
	}
}

// actionOutcome renders what the action produces: the outbound for a route,
// the server for a resolve, and so on.
func actionOutcome(action option.RuleAction) string {
	switch action.Action {
	case "", C.RuleActionTypeRoute:
		return action.RouteOptions.Outbound
	case C.RuleActionTypeReject:
		if action.RejectOptions.Method != "" {
			return action.RejectOptions.Method
		}
		return "default"
	case C.RuleActionTypeResolve:
		return action.ResolveOptions.Server
	default:
		return ""
	}
}

// defaultOutbound reproduces sing-box's fall-through, which has three cases
// that are easy to conflate (adapter/outbound/manager.go:59-75, 289):
//
//  1. route.final names an outbound, and that one is used.
//  2. route.final is unset, so the FIRST outbound in the list becomes default.
//  3. there are no outbounds at all, and sing-box synthesises a direct one.
//
// Reporting "final" for all three would send someone hunting for a config key
// that is not there.
func defaultOutbound(options *option.Options) (string, string) {
	if options == nil {
		return "direct", "implicit_direct"
	}
	if options.Route != nil && strings.TrimSpace(options.Route.Final) != "" {
		return options.Route.Final, "route.final"
	}
	for _, outbound := range options.Outbounds {
		if outbound.Tag != "" {
			return outbound.Tag, "first_outbound"
		}
	}
	for _, endpoint := range options.Endpoints {
		if endpoint.Tag != "" {
			return endpoint.Tag, "first_outbound"
		}
	}
	return "direct", "implicit_direct"
}

// summarize renders a rule's conditions compactly, in the order sing-box
// evaluates the groups.
func summarize(rule option.Rule) string {
	if rule.Type == C.RuleTypeLogical {
		return fmt.Sprintf("%s of %d rules", strings.ToUpper(rule.LogicalOptions.Mode), len(rule.LogicalOptions.Rules))
	}

	raw := rule.DefaultOptions.RawDefaultRule
	var parts []string

	appendList := func(name string, values []string) {
		if len(values) > 0 {
			parts = append(parts, name+"="+joinCapped(values, 3))
		}
	}
	appendList("inbound", raw.Inbound)
	appendList("network", raw.Network)
	appendList("protocol", raw.Protocol)
	appendList("domain", raw.Domain)
	appendList("domain_suffix", raw.DomainSuffix)
	appendList("domain_keyword", raw.DomainKeyword)
	appendList("domain_regex", raw.DomainRegex)
	appendList("geosite", raw.Geosite)
	appendList("geoip", raw.GeoIP)
	appendList("ip_cidr", raw.IPCIDR)
	appendList("source_ip_cidr", raw.SourceIPCIDR)
	appendList("rule_set", raw.RuleSet)
	if raw.IPIsPrivate {
		parts = append(parts, "ip_is_private=true")
	}
	if raw.SourceIPIsPrivate {
		parts = append(parts, "source_ip_is_private=true")
	}
	if raw.ClashMode != "" {
		parts = append(parts, "clash_mode="+raw.ClashMode)
	}
	if len(raw.Port) > 0 {
		ports := make([]string, 0, len(raw.Port))
		for _, port := range raw.Port {
			ports = append(ports, strconv.Itoa(int(port)))
		}
		appendList("port", ports)
	}
	appendList("port_range", raw.PortRange)
	if raw.Invert {
		parts = append(parts, "invert=true")
	}

	if len(parts) == 0 {
		return "(no conditions)"
	}
	return strings.Join(parts, " ")
}

// joinCapped renders at most `limit` values so a 40-entry list does not flood
// the display.
func joinCapped(values []string, limit int) string {
	if len(values) == 1 {
		return values[0]
	}
	if len(values) > limit {
		return fmt.Sprintf("[%s ...+%d]", strings.Join(values[:limit], " "), len(values)-limit)
	}
	return "[" + strings.Join(values, " ") + "]"
}

func parseRange(raw string) (uint16, uint16, bool) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	var start, end uint64 = 0, 65535
	var err error
	if parts[0] != "" {
		if start, err = strconv.ParseUint(parts[0], 10, 16); err != nil {
			return 0, 0, false
		}
	}
	if parts[1] != "" {
		if end, err = strconv.ParseUint(parts[1], 10, 16); err != nil {
			return 0, 0, false
		}
	}
	return uint16(start), uint16(end), true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func appendOnce(values []string, candidate string) []string {
	if contains(values, candidate) {
		return values
	}
	return append(values, candidate)
}

func sortStrings(values []string) {
	sort.Strings(values)
}
