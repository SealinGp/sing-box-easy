package config

import "github.com/sagernet/sing-box/option"

// PruneGroupReferences rewrites all selector/urltest outbound groups so they no
// longer reference deleted tags, pick up any renames produced by the same edit
// (e.g. a subscription update that re-tags an existing server), and sync newly
// added nodes into node *collections* (but never into node *groups*).
//
//   - deletedTags: tags that have been removed from the outbound list; any
//     reference to one of these tags is stripped from group `outbounds` lists
//     and from selector `default` fields.
//   - renameMap:   old-tag -> new-tag rewrites for outbounds whose identity
//     (server:port) survives but whose tag changed; references are rewritten
//     before the delete check so a renamed-and-then-deleted tag still wins.
//   - addTags:     tags of freshly-added outbounds (e.g. new subscription
//     nodes) to append to every node *collection*'s member list. Node *groups*
//     (see IsNodeGroup) are skipped — they curate other selectors and must not
//     have raw nodes dumped into them. Pass nil to add nothing.
//
// Immutability contract:
//   - The input []Outbound slice is not mutated.
//   - For each group outbound this function rewrites, a *fresh* options-struct
//     pointer is installed (the original `*option.SelectorOutboundOptions` /
//     `*option.URLTestOutboundOptions` stays untouched).
//   - The `Outbounds []string` field on those copies is a fresh slice; the
//     `Default` string field is a value type. All *other* scalar fields are
//     value-copied (safe). Be aware: if upstream sing-box ever adds another
//     slice or map field to these option types, this helper would need an
//     explicit clone for that field — the current set is value-only.
//   - Non-group outbounds (vmess, trojan, ...) are returned by value; their
//     Options pointer is the same one the caller passed in. The helper never
//     touches them.
//
// `sing-box check` validates protocol-level fields, not whether group outbounds
// reference live tags, so leaving stale references in place silently breaks
// selectors at runtime even though the config "validates".
func PruneGroupReferences(outbounds []Outbound, deletedTags map[string]struct{}, renameMap map[string]string, addTags []string) []Outbound {
	if len(outbounds) == 0 {
		return outbounds
	}
	if len(deletedTags) == 0 && len(renameMap) == 0 && len(addTags) == 0 {
		return outbounds
	}

	rewriteList := func(list []string) []string {
		if len(list) == 0 {
			return list
		}
		newList := make([]string, 0, len(list))
		seen := make(map[string]struct{}, len(list))
		for _, tag := range list {
			if newTag, ok := renameMap[tag]; ok {
				tag = newTag
			}
			if _, deleted := deletedTags[tag]; deleted {
				continue
			}
			if _, dup := seen[tag]; dup {
				// Renames can collapse two distinct entries onto the same tag.
				continue
			}
			seen[tag] = struct{}{}
			newList = append(newList, tag)
		}
		return newList
	}

	rewriteDefault := func(def string) string {
		if def == "" {
			return def
		}
		if newTag, ok := renameMap[def]; ok {
			def = newTag
		}
		if _, deleted := deletedTags[def]; deleted {
			return ""
		}
		return def
	}

	// appendNewNodes returns a fresh list with addTags appended (deduped against
	// the existing members). Always allocates a new slice so the caller's input
	// is never mutated. Only invoked for node collections, never for groups.
	appendNewNodes := func(list []string) []string {
		seen := make(map[string]struct{}, len(list)+len(addTags))
		out := make([]string, 0, len(list)+len(addTags))
		for _, tag := range list {
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
		for _, tag := range addTags {
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
		return out
	}

	result := make([]Outbound, len(outbounds))
	for i, ob := range outbounds {
		switch ob.Type {
		case "selector":
			if opts, ok := ob.Options.(*option.SelectorOutboundOptions); ok && opts != nil {
				newOpts := *opts
				newOpts.Outbounds = rewriteList(opts.Outbounds)
				newOpts.Default = rewriteDefault(opts.Default)
				// Sync new nodes into collections only; groups curate other
				// selectors and must not receive raw nodes.
				if len(addTags) > 0 && !IsNodeGroup(ob.Tag) {
					newOpts.Outbounds = appendNewNodes(newOpts.Outbounds)
				}
				ob.Options = &newOpts
			}
		case "urltest":
			if opts, ok := ob.Options.(*option.URLTestOutboundOptions); ok && opts != nil {
				newOpts := *opts
				newOpts.Outbounds = rewriteList(opts.Outbounds)
				if len(addTags) > 0 && !IsNodeGroup(ob.Tag) {
					newOpts.Outbounds = appendNewNodes(newOpts.Outbounds)
				}
				ob.Options = &newOpts
			}
		}
		result[i] = ob
	}
	return result
}
