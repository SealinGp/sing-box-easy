package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrDanglingOutboundReference marks an outbound edit refused because it would
// remove a tag something else in the config still routes to.
var ErrDanglingOutboundReference = errors.New("outbound is still referenced")

// OutboundReference is one place outside the outbound list that names an
// outbound by tag.
type OutboundReference struct {
	// Where is the config path of the reference, written the way an operator
	// would look it up: "route.final", "route.rules[3]".
	Where string
	Tag   string
}

// outboundReferences lists every reference to an outbound tag from the
// sections an outbound edit does not rewrite.
//
// These are exactly the references `sing-box check` does NOT validate. It
// parses the config; it does not build the router. A `route.final`, a route
// rule or a detour naming an outbound that does not exist therefore passes the
// check and is only discovered when the service STARTS — "default outbound not
// found" — at which point an init system that respawns turns one bad edit into
// a crash loop on a device that is also the network's router.
//
// Read from raw JSON for the reason the rest of this package gives: decoding
// `route` or `dns` through the pinned option structs would fail on fields a
// newer core added, and this must work on a host running any version.
// Anything that does not have the expected shape is skipped, not reported: a
// guard that cannot read a section must not block edits to another one.
func outboundReferences(sections map[string]json.RawMessage) []OutboundReference {
	var references []OutboundReference
	add := func(where string, tag any) {
		if name, ok := tag.(string); ok && name != "" {
			references = append(references, OutboundReference{Where: where, Tag: name})
		}
	}

	var route map[string]any
	if json.Unmarshal(sections["route"], &route) == nil {
		add("route.final", route["final"])
		collectRuleOutbounds("route.rules", route["rules"], add)
		if ruleSets, ok := route["rule_set"].([]any); ok {
			for index, entry := range ruleSets {
				if ruleSet, ok := entry.(map[string]any); ok {
					add(fmt.Sprintf("route.rule_set[%d].download_detour", index), ruleSet["download_detour"])
				}
			}
		}
	}

	var dns map[string]any
	if json.Unmarshal(sections["dns"], &dns) == nil {
		if servers, ok := dns["servers"].([]any); ok {
			for index, entry := range servers {
				if server, ok := entry.(map[string]any); ok {
					add(fmt.Sprintf("dns.servers[%d].detour", index), server["detour"])
				}
			}
		}
	}
	return references
}

// collectRuleOutbounds walks a rule list, descending into logical rules, whose
// sub-rules live under their own `rules` key.
func collectRuleOutbounds(path string, value any, add func(where string, tag any)) {
	rules, ok := value.([]any)
	if !ok {
		return
	}
	for index, entry := range rules {
		rule, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		where := fmt.Sprintf("%s[%d]", path, index)
		add(where, rule["outbound"])
		collectRuleOutbounds(where+".rules", rule["rules"], add)
	}
}

// danglingAfter returns the references an outbound edit would leave pointing
// at nothing: those whose tag was in the list BEFORE the edit and is gone
// AFTER it.
//
// Only references this edit breaks are reported. One that already named a
// missing outbound is somebody else's problem — the Overview diagram flags it —
// and refusing every later outbound edit because of it would leave the
// operator unable to fix anything from the panel.
func danglingAfter(before, after []Outbound, sections map[string]json.RawMessage) []OutboundReference {
	had := make(map[string]struct{}, len(before))
	for _, outbound := range before {
		had[outbound.Tag] = struct{}{}
	}
	has := make(map[string]struct{}, len(after))
	for _, outbound := range after {
		has[outbound.Tag] = struct{}{}
	}

	var dangling []OutboundReference
	for _, reference := range outboundReferences(sections) {
		_, existed := had[reference.Tag]
		_, exists := has[reference.Tag]
		if existed && !exists {
			dangling = append(dangling, reference)
		}
	}
	return dangling
}

// danglingReferenceError explains a refused edit in terms the operator can act
// on: which outbound, who still points at it, and the two ways out.
func danglingReferenceError(dangling []OutboundReference) error {
	places := make(map[string][]string)
	var order []string
	for _, reference := range dangling {
		if _, seen := places[reference.Tag]; !seen {
			order = append(order, reference.Tag)
		}
		places[reference.Tag] = append(places[reference.Tag], reference.Where)
	}
	parts := make([]string, 0, len(order))
	for _, tag := range order {
		parts = append(parts, fmt.Sprintf("%q (used by %s)", tag, strings.Join(places[tag], ", ")))
	}
	return &ValidationError{
		Stage: ValidationStagePanelGuard,
		Err: fmt.Errorf(
			"%w: this change would remove %s, and sing-box would then refuse to start. "+
				"Point those settings at another outbound first, or keep this one. "+
				"A node-rules filter or group is removed when it matches no nodes",
			ErrDanglingOutboundReference, strings.Join(parts, "; ")),
	}
}
