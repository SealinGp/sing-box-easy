package subscription

import (
	"encoding/json"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config/outbounds/nodetag"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/model"
	"strings"
)

// collectTags returns the Tag of every outbound in the slice.
func collectTags(outbounds []config.Outbound) []string {
	if len(outbounds) == 0 {
		return nil
	}
	tags := make([]string, 0, len(outbounds))
	for _, o := range outbounds {
		tags = append(tags, o.Tag)
	}
	return tags
}

// collectTagsFromMap returns the Tag of every outbound in the map.
func collectTagsFromMap(m map[string]config.Outbound) []string {
	if len(m) == 0 {
		return nil
	}
	tags := make([]string, 0, len(m))
	for _, o := range m {
		tags = append(tags, o.Tag)
	}
	return tags
}

// collectKeys returns the keys of a set-style map (struct{} values).
func collectKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// subscriptionTagSeparator joins the human-facing unique tag with its owning
// subscription's ID. It is visually distinct and unlikely to appear inside a
// provider-supplied node name, so it can be used to recognize ownership.
const subscriptionTagSeparator = " | "

// subscriptionTagSuffix returns the ownership suffix for a subscription, e.g.
// " | sub_1778669376". The ID is appended (not prepended) because it exists
// only to machine-identify the owning subscription — keeping it at the end
// leaves the human-readable "<name> <server:port>" at the front of the tag.
// Every node fetched from this subscription is tagged as
// "<name> <server:port> | <subID>".
func subscriptionTagSuffix(subID string) string {
	return subscriptionTagSeparator + subID
}

// TagBelongsToSubscription reports whether an outbound tag was minted for the
// given subscription (i.e. it carries that subscription's ID suffix). This is
// the authoritative ownership test: it holds no matter how the provider has
// since renamed the node in front of the suffix.
//
// Exported because it is the ONE rule that decides which outbounds a
// subscription is answerable for, and a second consumer now needs it: the
// quality prober (app/pkg/subscription/internal/probe) groups nodes by subscription to compute
// availability. A prober carrying its own copy of the suffix rule would keep
// measuring the wrong nodes for a while after this one changed.
func TagBelongsToSubscription(tag, subID string) bool {
	return model.TagBelongsToSubscription(tag, subID)
}

// fingerprintLegacyTag rewrites a pre-fingerprint subscription tag
// ("<name> <server:port> | <subID>") into the current form
// ("<name> <fingerprint> | <subID>"), or returns "" when the tag is not in the
// legacy shape and therefore needs no migration.
//
// The endpoint is recovered from the tag itself rather than from the outbound's
// options on purpose: the tag records the endpoint the node had WHEN IT WAS
// MINTED, and matching is against tags the feed produces from those same
// strings. Reading today's options would silently mis-migrate a node whose
// provider moved it to a new host.
func fingerprintLegacyTag(tag, subID string) string {
	suffix := subscriptionTagSuffix(subID)
	if !strings.HasSuffix(tag, suffix) {
		return ""
	}
	head := strings.TrimSuffix(tag, suffix)

	space := strings.LastIndex(head, " ")
	if space == -1 {
		return ""
	}
	endpoint := head[space+1:]
	// Only "host:port" (or a bare host) is a legacy endpoint. Anything else —
	// including a tag already carrying a fingerprint — is left alone.
	if !strings.Contains(endpoint, ":") && !strings.Contains(endpoint, ".") {
		return ""
	}
	fp := nodetag.FingerprintEndpointKey(endpoint)
	if fp == "" || fp == endpoint {
		return ""
	}
	return head[:space+1] + fp + suffix
}

// diffNodes compares current outbounds with the freshly fetched nodes and
// returns the changes, keyed by the existing outbound's tag.
//
// Ownership is determined by the subscription-ID suffix on the tag, NOT by the
// node's server host. Every node fetched from a subscription is tagged as
// "<name> <server:port> | <subID>", so the suffix unambiguously enumerates all
// of a subscription's nodes even when the provider renames them or co-locates
// many distinct nodes behind one relay/CDN endpoint. The leading
// "name server:port" keeps tags unique within the namespace (sing-box requires
// globally unique tags), so it remains the per-node identity in front of the
// suffix.
//
//   - toDelete: tags of this subscription's outbounds no longer in the feed
//     (including nodes whose name changed — the old tag is deleted and the new
//     one is added, since the name is part of the identity).
//   - toUpdate: existing tag -> replacement outbound. The value's Tag differs
//     from the key only for the legacy-migration case (an un-suffixed outbound
//     adopted into the namespace), which is applied as a group-reference rename
//     in applyChanges.
//   - toAdd:    feed nodes with no matching existing outbound.
func (au *Service) diffNodes(cfg *config.SingBoxConfig, sub *Subscription, newNodes []*node.SubNode) (toDelete map[string]struct{}, toAdd []config.Outbound, toUpdate map[string]config.Outbound) {
	toDelete = make(map[string]struct{})
	toUpdate = make(map[string]config.Outbound)

	suffix := subscriptionTagSuffix(sub.ID)

	// Index the feed by its final (suffixed) unique tag. subServers records the
	// feed's server hosts purely so legacy un-suffixed outbounds from this
	// subscription can still be recognized and migrated into the namespace.
	newByTag := make(map[string]config.Outbound, len(newNodes))
	subServers := make(map[string]struct{})
	for _, n := range newNodes {
		outbound := config.Outbound{
			Tag:     n.Tag,
			Type:    n.Type,
			Options: n.Options,
		}
		if svr := nodetag.GetOutboundServer(outbound); svr != "" {
			subServers[svr] = struct{}{}
		}
		if nodetag.GetOutboundServerKey(outbound) == "" {
			continue
		}
		// Final tag = "<name> <endpoint-fingerprint> | <subID>".
		taggedTag := nodetag.GenerateFingerprintedTag(n.Tag, outbound) + suffix
		outbound.Tag = taggedTag
		// Two feed nodes with the same name AND endpoint are genuinely
		// indistinguishable; collapsing them here is correct (one survives).
		newByTag[taggedTag] = outbound
	}

	// consumed tracks which feed nodes (by suffixed tag) were matched to an
	// existing outbound, so the leftovers become additions.
	consumed := make(map[string]bool)

	// Pass 1: outbounds already in this subscription's namespace. The suffix is
	// the sole ownership signal here, so a node that was renamed by the provider
	// (new tag) is correctly seen as a delete of the old tag plus an add.
	for _, outbound := range cfg.Outbounds {
		if !TagBelongsToSubscription(outbound.Tag, sub.ID) {
			continue
		}
		if newOutbound, ok := newByTag[outbound.Tag]; ok && !consumed[outbound.Tag] {
			consumed[outbound.Tag] = true
			if !outboundsDeepEqual(outbound, newOutbound) {
				toUpdate[outbound.Tag] = newOutbound
			}
			continue
		}

		// Tags minted before the endpoint was hashed spell the endpoint out:
		// "<name> <server:port> | <subID>". Convert and retry, so the format
		// change lands as a RENAME rather than as delete-plus-add. The
		// difference is not cosmetic: a rename carries selector/urltest
		// membership across (see renameMap in applyChanges), while a delete
		// would drop every group reference to the node and re-add it bare.
		if migrated := fingerprintLegacyTag(outbound.Tag, sub.ID); migrated != "" {
			if newOutbound, ok := newByTag[migrated]; ok && !consumed[migrated] {
				consumed[migrated] = true
				toUpdate[outbound.Tag] = newOutbound
				continue
			}
		}

		// Carries our suffix but no longer in the feed → delete.
		toDelete[outbound.Tag] = struct{}{}
	}

	// Pass 2: migration for outbounds added before tag-suffixing existed (or
	// added manually). Recognize them by server host and re-tag them into the
	// namespace so future passes can rely solely on the suffix. Running after
	// pass 1 guarantees already-suffixed nodes win the match for a feed entry.
	for _, outbound := range cfg.Outbounds {
		// Any namespace denotes ownership; another subscription is never legacy.
		if strings.Contains(outbound.Tag, subscriptionTagSeparator) {
			continue
		}
		// Only outbounds whose server is present in this feed are candidates,
		// matching the historical host-based ownership heuristic.
		if _, ok := subServers[nodetag.GetOutboundServer(outbound)]; !ok {
			continue
		}

		// The legacy tag is a bare display name, or already carries an endpoint
		// ("name server:port"). The feed index is keyed by the current form
		// ("name fingerprint | subID"), so every candidate is built in that
		// form: the bare tag, the tag plus this outbound's own fingerprint, and
		// the tag with an endpoint it already spells out converted.
		candidates := []string{outbound.Tag + suffix}
		if fp := nodetag.FingerprintEndpointKey(nodetag.GetOutboundServerKey(outbound)); fp != "" {
			candidates = append(candidates, outbound.Tag+" "+fp+suffix)
		}
		if migrated := fingerprintLegacyTag(outbound.Tag+suffix, sub.ID); migrated != "" {
			candidates = append(candidates, migrated)
		}
		matched := false
		for _, candidate := range candidates {
			if newOutbound, ok := newByTag[candidate]; ok && !consumed[candidate] {
				consumed[candidate] = true
				// Rename in place (legacy tag -> suffixed tag) so selector/urltest
				// group memberships are preserved via the rename map in applyChanges.
				toUpdate[outbound.Tag] = newOutbound
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Legacy node from this subscription's host set but absent from the feed → delete.
		toDelete[outbound.Tag] = struct{}{}
	}

	for tag, outbound := range newByTag {
		if !consumed[tag] {
			toAdd = append(toAdd, outbound)
		}
	}

	return toDelete, toAdd, toUpdate
}

// outboundsDeepEqual performs deep comparison of two outbounds via JSON.
// The same options struct may be a typed sing-box option type on one side and
// a map on the other (e.g. existing outbound from typed registry vs. new node
// from parser); marshalling both to JSON normalizes them before compare.
func outboundsDeepEqual(a, b config.Outbound) bool {
	if a.Type != b.Type {
		return false
	}
	aJSON, err1 := json.Marshal(a.Options)
	bJSON, err2 := json.Marshal(b.Options)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}
