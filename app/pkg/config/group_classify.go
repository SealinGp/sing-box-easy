package config

import "strings"

// groupNameMarkers are the (temporary) substrings that mark a selector/urltest
// as a node *group* rather than a flat node *collection*.
//
// Distinction:
//   - Node collection: a flat aggregate of final exit nodes (real proxies).
//     Subscription updates keep it in sync — new nodes are added, gone nodes
//     removed. Examples: "♻️ 自动选择", "🌍 其他节点".
//   - Node group: a curated list that includes non-final outbounds (other
//     selectors/urltests). It must NOT receive raw subscription nodes; only
//     existing references are pruned/renamed. Example: "🎬 流媒体分组" which
//     references "♻️ 自动选择", "🤖 AI", etc.
//
// TODO: replace this name-based heuristic with a structural check — a group is
// any selector/urltest whose member list references other selector/urltest
// tags (i.e. contains non-final outbounds). The name match is a placeholder.
var groupNameMarkers = []string{"分组", "group"}

// IsNodeGroup reports whether an outbound tag denotes a node *group* (a curated
// list that may include non-final outbounds) as opposed to a node *collection*
// (a flat aggregate of final exit nodes). Group tags are skipped when syncing
// freshly-added subscription nodes into selector/urltest member lists.
func IsNodeGroup(tag string) bool {
	lower := strings.ToLower(tag)
	for _, marker := range groupNameMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
