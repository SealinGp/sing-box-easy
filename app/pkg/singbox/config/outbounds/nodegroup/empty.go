package nodegroup

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing-box/option"
)

// PruneEmptyGroups removes groups that cannot be initialized and recursively
// prunes their parent memberships. before identifies groups removed by a rules
// rebuild; unknown tags are retained because they may name opaque outbounds or
// top-level endpoints. It never substitutes unrelated nodes or a direct route.
// Like PruneGroupReferences, it leaves the input slices and options untouched.
func PruneEmptyGroups(before, after []config.Outbound) []config.Outbound {
	present := make(map[string]bool, len(after))
	for _, ob := range after {
		present[ob.Tag] = true
	}
	removed := make(map[string]struct{})
	for _, ob := range before {
		if (ob.Type == "selector" || ob.Type == "urltest") && !present[ob.Tag] {
			removed[ob.Tag] = struct{}{}
		}
	}
	result := after
	for {
		result = PruneGroupReferences(result, removed, nil, nil)
		removed = make(map[string]struct{})
		kept := make([]config.Outbound, 0, len(result))
		for _, ob := range result {
			empty := false
			switch opts := ob.Options.(type) {
			case *option.SelectorOutboundOptions:
				empty = ob.Type == "selector" && opts != nil && len(opts.Outbounds) == 0
			case *option.URLTestOutboundOptions:
				empty = ob.Type == "urltest" && opts != nil && len(opts.Outbounds) == 0
			}
			if empty {
				removed[ob.Tag] = struct{}{}
			} else {
				kept = append(kept, ob)
			}
		}
		if len(removed) == 0 {
			return result
		}
		result = kept
	}
}
