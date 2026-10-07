package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// The reference check: the part of "is this config loadable" that
// `sing-box check` leaves out.
//
// `sing-box check` builds the outbounds and parses the route, but it does not
// START the router, and it is at start that sing-box resolves the tags the
// route names. A `route.final`, a route rule or a detour naming an outbound
// that does not exist therefore PASSES the check and kills the service when it
// starts — "default outbound not found: Other Nodes". On a host whose init
// system respawns, one such edit is a crash loop, and on a router that is the
// whole network down. Operators met this exactly as described: the check said
// fine, and only the runtime log said otherwise (#14).
//
// So the panel resolves those references itself, at the two places every
// config passes through:
//
//   - saveDocumentLocked — every write. A save is refused when it would ADD a
//     problem. Problems that were already there do not block it, or a config
//     with two broken references could never be repaired one edit at a time.
//   - ValidateCurrentConfig — before start, restart and reload. Here ANY
//     problem is fatal, because the core is about to fail on it anyway and
//     "route.final names a missing outbound" is a better thing to be told than
//     a respawn loop.
//
// Everything is read from raw JSON, for the reason the rest of this package
// gives: decoding `route` or `dns` through the pinned option structs would fail
// on fields a newer core added. Anything that does not have the expected shape
// is skipped rather than reported — a guard that cannot read a section must not
// invent a problem in it.

// ErrOutboundReference marks a config refused by the reference check.
var ErrOutboundReference = errors.New("config references an unusable outbound")

// ReferenceKind says what is wrong with the outbound a setting names.
type ReferenceKind string

const (
	// ReferenceMissing: no outbound or endpoint has that tag.
	ReferenceMissing ReferenceKind = "missing_outbound"
	// ReferenceEmptyGroup: the tag is a selector/urltest with no members. The
	// core rejects that too; it is reported here in the operator's terms.
	ReferenceEmptyGroup ReferenceKind = "empty_group"
)

// Reference fields. They are part of the API: the frontend maps each one to the
// page where it is edited.
const (
	FieldRouteFinal    = "route.final"
	FieldRouteRules    = "route.rules"
	FieldRouteRuleSet  = "route.rule_set"
	FieldDNSServers    = "dns.servers"
	referenceListLimit = 8
)

// ReferenceProblem is one setting that names an outbound sing-box cannot use.
type ReferenceProblem struct {
	Kind ReferenceKind `json:"kind"`
	// Field is the setting family, one of the Field* constants.
	Field string `json:"field"`
	// Where is the exact config path: "route.final", "route.rules[3]".
	Where string `json:"where"`
	Tag   string `json:"tag"`
}

// key identifies a problem across two versions of a document. The index in
// `Where` is deliberately not part of it: reordering rules moves a broken rule
// to another index without making it a new problem.
func (p ReferenceProblem) key() string {
	return string(p.Kind) + "\x00" + p.Field + "\x00" + p.Tag
}

// ReferenceError carries the problems to the API layer, which serialises them
// so the frontend can say what to fix and where.
type ReferenceError struct {
	Problems []ReferenceProblem
}

func (e *ReferenceError) Error() string {
	parts := make([]string, 0, len(e.Problems))
	for index, problem := range e.Problems {
		if index == referenceListLimit {
			parts = append(parts, fmt.Sprintf("and %d more", len(e.Problems)-index))
			break
		}
		switch problem.Kind {
		case ReferenceEmptyGroup:
			parts = append(parts, fmt.Sprintf("%s points at %q, a group with no members", problem.Where, problem.Tag))
		default:
			parts = append(parts, fmt.Sprintf("%s points at %q, which is not an outbound in this config", problem.Where, problem.Tag))
		}
	}
	return "sing-box would refuse to start with this config: " + strings.Join(parts, "; ") +
		". `sing-box check` does not detect this; choose an existing outbound for each setting"
}

func (e *ReferenceError) Is(target error) bool { return target == ErrOutboundReference }

// CheckReferences returns every reference problem in a config document.
func CheckReferences(raw []byte) []ReferenceProblem {
	var sections map[string]json.RawMessage
	if json.Unmarshal(raw, &sections) != nil {
		return nil
	}
	return referenceProblems(sections)
}

type outboundKind struct {
	group   bool
	members int
}

// knownOutbounds maps every tag a route may name to what it is. Endpoints
// (wireguard, tailscale) count: since sing-box 1.11 they are routable targets
// that live in their own section, and leaving them out would report a working
// config as broken.
func knownOutbounds(sections map[string]json.RawMessage) (map[string]outboundKind, bool) {
	known := make(map[string]outboundKind)
	var outbounds []map[string]any
	if raw, present := sections["outbounds"]; present {
		if json.Unmarshal(raw, &outbounds) != nil {
			// The list itself is unreadable: nothing can be concluded about
			// what exists, so nothing is reported as missing.
			return nil, false
		}
	}
	for _, outbound := range outbounds {
		tag, _ := outbound["tag"].(string)
		if tag == "" {
			continue
		}
		kind := outboundKind{}
		if family, _ := outbound["type"].(string); family == "selector" || family == "urltest" {
			kind.group = true
			if members, ok := outbound["outbounds"].([]any); ok {
				kind.members = len(members)
			}
		}
		known[tag] = kind
	}
	var endpoints []map[string]any
	if json.Unmarshal(sections["endpoints"], &endpoints) == nil {
		for _, endpoint := range endpoints {
			if tag, _ := endpoint["tag"].(string); tag != "" {
				known[tag] = outboundKind{}
			}
		}
	}
	return known, true
}

func referenceProblems(sections map[string]json.RawMessage) []ReferenceProblem {
	known, readable := knownOutbounds(sections)
	if !readable {
		return nil
	}
	var problems []ReferenceProblem
	check := func(field, where string, value any) {
		tag, ok := value.(string)
		if !ok || tag == "" {
			return
		}
		kind, exists := known[tag]
		switch {
		case !exists:
			problems = append(problems, ReferenceProblem{Kind: ReferenceMissing, Field: field, Where: where, Tag: tag})
		case kind.group && kind.members == 0:
			problems = append(problems, ReferenceProblem{Kind: ReferenceEmptyGroup, Field: field, Where: where, Tag: tag})
		}
	}

	var route map[string]any
	if json.Unmarshal(sections["route"], &route) == nil {
		check(FieldRouteFinal, "route.final", route["final"])
		walkRuleOutbounds("route.rules", route["rules"], func(where string, value any) {
			check(FieldRouteRules, where, value)
		})
		if ruleSets, ok := route["rule_set"].([]any); ok {
			for index, entry := range ruleSets {
				if ruleSet, ok := entry.(map[string]any); ok {
					check(FieldRouteRuleSet, fmt.Sprintf("route.rule_set[%d].download_detour", index), ruleSet["download_detour"])
				}
			}
		}
	}

	var dns map[string]any
	if json.Unmarshal(sections["dns"], &dns) == nil {
		if servers, ok := dns["servers"].([]any); ok {
			for index, entry := range servers {
				if server, ok := entry.(map[string]any); ok {
					check(FieldDNSServers, fmt.Sprintf("dns.servers[%d].detour", index), server["detour"])
				}
			}
		}
	}
	return problems
}

// walkRuleOutbounds visits the `outbound` of every rule, descending into
// logical rules, whose sub-rules live under their own `rules` key.
func walkRuleOutbounds(path string, value any, visit func(where string, value any)) {
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
		visit(where, rule["outbound"])
		walkRuleOutbounds(where+".rules", rule["rules"], visit)
	}
}

// introducedProblems returns the problems in `after` that `before` did not
// already have.
func introducedProblems(before, after []ReferenceProblem) []ReferenceProblem {
	had := make(map[string]struct{}, len(before))
	for _, problem := range before {
		had[problem.key()] = struct{}{}
	}
	var introduced []ReferenceProblem
	for _, problem := range after {
		if _, existed := had[problem.key()]; !existed {
			introduced = append(introduced, problem)
		}
	}
	return introduced
}

// guardNewReferences refuses a document that adds a reference problem to the
// one currently on disk. A missing or unreadable current file has no problems
// to grandfather, so the new document must then be clean.
func (m *Manager) guardNewReferences(next []byte) error {
	current, _ := os.ReadFile(m.configPath)
	if introduced := introducedProblems(CheckReferences(current), CheckReferences(next)); len(introduced) > 0 {
		return &ValidationError{Stage: ValidationStagePanelGuard, Err: &ReferenceError{Problems: introduced}}
	}
	return nil
}
