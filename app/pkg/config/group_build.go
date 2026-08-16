package config

import (
	"sort"
	"time"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// EndpointTags returns the tags of every real exit-node outbound (skipping
// selector/urltest groups and pseudo-outbounds). This is the input set the
// node-rules matcher is allowed to collect into Filters.
func EndpointTags(outbounds []Outbound) []string {
	out := make([]string, 0, len(outbounds))
	for _, ob := range outbounds {
		if IsEndpointType(ob.Type) {
			out = append(out, ob.Tag)
		}
	}
	return out
}

// FilterSpec describes one Filter to materialize as a generated group outbound.
// MemberTags are the endpoint tags assigned to this Filter (already determined
// by the node-rules matcher). The package is intentionally free of any
// noderules dependency — callers translate their domain types into these specs.
type FilterSpec struct {
	Name         string   // becomes the generated outbound tag
	OutboundType string   // "urltest" (default) or "selector"
	MemberTags   []string // endpoint tags assigned to this Filter
	// urltest health-check settings (ignored for selector). Callers pass concrete
	// values (defaults already applied); empty/zero fields are simply omitted.
	TestURL       string // e.g. "http://www.gstatic.com/generate_204"
	TestInterval  string // duration string, e.g. "10s"
	TestTolerance int    // milliseconds, e.g. 200
}

// GroupSpec describes one Group to materialize as a generated selector outbound.
// FilterNames is the ordered list of member Filter names; only Filters that were
// actually emitted (non-empty) are referenced.
type GroupSpec struct {
	Name        string
	FilterNames []string
}

// BuildGroupOutbounds rebuilds the rule-managed Filter and Group outbounds from
// the given specs, returning a fresh outbound list.
//
// Behavior:
//   - Endpoint outbounds and user-authored (non-managed) outbounds are kept
//     verbatim and in place.
//   - Any existing outbound whose tag equals a managed Filter/Group name is
//     treated as a previously-generated group and rebuilt in place (or dropped
//     if it is now empty). Brand-new managed outbounds are appended at the end,
//     preserving the position of pre-existing groups so route.final / implicit
//     first-outbound semantics stay stable.
//   - A Filter with zero members is SKIPPED (sing-box rejects an empty
//     selector/urltest). Groups drop references to skipped Filters; a Group left
//     with zero members is itself skipped.
//
// Immutability: the input slice and its option structs are never mutated; every
// generated outbound gets a fresh options pointer.
func BuildGroupOutbounds(existing []Outbound, filters []FilterSpec, groups []GroupSpec) []Outbound {
	// 1. Build the emitted Filter outbounds (skip empties) and remember which
	//    Filter names actually produced an outbound, so Groups can reference only
	//    live ones.
	emittedFilter := make(map[string]bool, len(filters))
	managed := make(map[string]Outbound, len(filters)+len(groups))
	// order preserves a stable append order for brand-new managed outbounds.
	order := make([]string, 0, len(filters)+len(groups))

	for _, f := range filters {
		if f.Name == "" || len(f.MemberTags) == 0 {
			continue
		}
		members := dedupeSorted(f.MemberTags)
		ob := Outbound{Tag: f.Name, Type: normalizeFilterType(f.OutboundType)}
		switch ob.Type {
		case "selector":
			ob.Options = &option.SelectorOutboundOptions{Outbounds: members}
		default: // urltest
			ob.Type = "urltest"
			ob.Options = buildURLTestOptions(members, f)
		}
		managed[f.Name] = ob
		emittedFilter[f.Name] = true
		order = append(order, f.Name)
	}

	// 2. Build Group outbounds referencing only emitted Filters.
	for _, g := range groups {
		if g.Name == "" {
			continue
		}
		members := make([]string, 0, len(g.FilterNames))
		seen := make(map[string]struct{}, len(g.FilterNames))
		for _, fn := range g.FilterNames {
			if !emittedFilter[fn] {
				continue
			}
			if _, dup := seen[fn]; dup {
				continue
			}
			seen[fn] = struct{}{}
			members = append(members, fn)
		}
		if len(members) == 0 {
			continue
		}
		managed[g.Name] = Outbound{
			Tag:     g.Name,
			Type:    "selector",
			Options: &option.SelectorOutboundOptions{Outbounds: members},
		}
		order = append(order, g.Name)
	}

	// 3. Rebuild the outbound list: replace previously-generated managed
	//    outbounds in place, drop now-empty ones, keep everything else verbatim.
	result := make([]Outbound, 0, len(existing)+len(managed))
	placed := make(map[string]bool, len(managed))
	managedNames := managedNameSet(filters, groups)

	for _, ob := range existing {
		if _, isManagedName := managedNames[ob.Tag]; isManagedName {
			// This tag is owned by the rule system. Replace with the rebuilt
			// version if it survived; otherwise drop it (became empty).
			if rebuilt, ok := managed[ob.Tag]; ok && !placed[ob.Tag] {
				result = append(result, rebuilt)
				placed[ob.Tag] = true
			}
			continue
		}
		result = append(result, ob)
	}

	// 4. Append brand-new managed outbounds (not previously present), in spec
	//    order: Filters first, then Groups.
	for _, name := range order {
		if placed[name] {
			continue
		}
		if ob, ok := managed[name]; ok {
			result = append(result, ob)
			placed[name] = true
		}
	}

	return result
}

// managedNameSet returns the set of all Filter and Group names, i.e. every tag
// the rule system owns and may add/remove/rebuild.
// ManagedOutboundTags returns the outbound tags the node-rules engine owns.
//
// BuildGroupOutbounds REPLACES any existing outbound whose tag is in this set,
// rather than merging into it — so an edit made through the outbounds form to
// one of these is silently discarded on the next rule apply, and for a Group
// the rebuilt selector carries only its members, dropping any url/interval/
// tolerance that had been set.
//
// Exported so the UI can say so before the operator spends time on an edit
// that will not survive. Deriving the same set in TypeScript would be the drift
// this codebase keeps getting bitten by.
func ManagedOutboundTags(filters []FilterSpec, groups []GroupSpec) []string {
	set := managedNameSet(filters, groups)
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func managedNameSet(filters []FilterSpec, groups []GroupSpec) map[string]struct{} {
	set := make(map[string]struct{}, len(filters)+len(groups))
	for _, f := range filters {
		if f.Name != "" {
			set[f.Name] = struct{}{}
		}
	}
	for _, g := range groups {
		if g.Name != "" {
			set[g.Name] = struct{}{}
		}
	}
	return set
}

// dedupeSorted returns a sorted, de-duplicated copy (deterministic membership).
func dedupeSorted(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func normalizeFilterType(t string) string {
	if t == "selector" {
		return "selector"
	}
	return "urltest"
}

// buildURLTestOptions assembles a urltest outbound's options, attaching the
// health-check url/interval/tolerance when provided. An unparseable interval is
// silently omitted (sing-box then applies its own default) rather than failing
// the whole rebuild.
func buildURLTestOptions(members []string, f FilterSpec) *option.URLTestOutboundOptions {
	opts := &option.URLTestOutboundOptions{Outbounds: members}
	if f.TestURL != "" {
		opts.URL = f.TestURL
	}
	if f.TestInterval != "" {
		if d, err := time.ParseDuration(f.TestInterval); err == nil {
			opts.Interval = badoption.Duration(d)
		}
	}
	if f.TestTolerance > 0 {
		opts.Tolerance = uint16(f.TestTolerance)
	}
	return opts
}
