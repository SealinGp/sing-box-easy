package rules

import "github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds/nodegroup"

// BuildSpecs runs the matcher over the current node pool and translates the
// rules into config build specs, returning the raw membership/others for
// preview and diagnostics.
//
//   - filters/groups should be passed in their display order (the manager
//     already sorts by priority); FilterSpec order mirrors it.
//   - membership is keyed by Filter ID; others lists endpoints that matched no
//     non-fallback Filter (already folded into the fallback Filter's members).
//
// This function is pure (no I/O). Callers wrap nodegroup.BuildGroupOutbounds with
// the returned specs inside a config.Manager.UpdateConfig closure to apply, or
// just read membership for a dry-run preview.
func BuildSpecs(filters []*Filter, groups []*Group, pool NodePool) (filterSpecs []nodegroup.FilterSpec, groupSpecs []nodegroup.GroupSpec, membership map[string][]string, others []string) {
	membership, others = AssignFilters(pool, filters)

	byID := make(map[string]*Filter, len(filters))
	for _, f := range filters {
		if f != nil {
			byID[f.ID] = f
		}
	}

	filterSpecs = make([]nodegroup.FilterSpec, 0, len(filters))
	for _, f := range filters {
		if f == nil {
			continue
		}
		spec := nodegroup.FilterSpec{
			Name:         f.Name,
			OutboundType: f.OutboundType,
			MemberTags:   membership[f.ID],
		}
		// urltest filters carry health-check settings (with defaults applied).
		if f.OutboundType == OutboundTypeURLTest {
			spec.TestURL, spec.TestInterval, spec.TestTolerance = f.URLTestSettings()
		}
		filterSpecs = append(filterSpecs, spec)
	}

	groupSpecs = make([]nodegroup.GroupSpec, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		names := make([]string, 0, len(g.FilterIDs))
		for _, fid := range g.FilterIDs {
			if f, ok := byID[fid]; ok {
				names = append(names, f.Name)
			}
		}
		groupSpecs = append(groupSpecs, nodegroup.GroupSpec{
			Name:        g.Name,
			FilterNames: names,
			ExtraTags:   g.ExtraTags,
		})
	}
	return filterSpecs, groupSpecs, membership, others
}
