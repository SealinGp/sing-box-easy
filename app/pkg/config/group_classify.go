package config

import "strings"

// nonEndpointTypes are outbound types that are NOT real exit nodes: group
// constructs (selector/urltest) and the built-in pseudo-outbounds. Everything
// else (vmess, trojan, shadowsocks, ...) is a final endpoint that Filters may
// match and collect.
var nonEndpointTypes = map[string]struct{}{
	"selector": {},
	"urltest":  {},
	"direct":   {},
	"block":    {},
	"dns":      {},
}

// IsEndpointType reports whether an outbound type denotes a real exit node (as
// opposed to a group or pseudo-outbound). Used to decide which outbounds the
// node-rules matcher is allowed to collect into Filters.
func IsEndpointType(outboundType string) bool {
	_, special := nonEndpointTypes[outboundType]
	return !special
}

// groupNameMarkers are the substrings that mark a selector/urltest as a node
// *group* rather than a flat node *collection*.
//
// DEPRECATED / LEGACY: this name heuristic is only used by PruneGroupReferences'
// addTags path, which is itself a fallback for when the Outbound Node Rules
// engine (app/pkg/noderules) is not configured. When rules are active (the
// default in production), node placement is owned by the rules engine and this
// heuristic is never consulted. Kept so direct AutoUpdater usage without a rules
// provider still degrades gracefully.
var groupNameMarkers = []string{"分组", "group"}

// IsNodeGroup reports whether an outbound tag denotes a node *group* (a curated
// list that may include non-final outbounds) as opposed to a node *collection*
// (a flat aggregate of final exit nodes). Group tags are skipped when syncing
// freshly-added subscription nodes into selector/urltest member lists.
//
// LEGACY: superseded by the Outbound Node Rules engine (see
// app/pkg/noderules); retained only for the no-rules fallback path.
func IsNodeGroup(tag string) bool {
	lower := strings.ToLower(tag)
	for _, marker := range groupNameMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
