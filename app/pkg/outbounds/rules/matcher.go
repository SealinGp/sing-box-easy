package noderules

import (
	"sort"
	"strings"
)

// matcherMatches reports whether a single matcher hits the (already lowercased)
// tag. The original tag is also passed because emoji synonyms are matched
// verbatim (lowercasing an emoji is a no-op but we keep the literal form clear).
func matcherMatches(m Matcher, tag, lowerTag string) bool {
	switch m.Type {
	case MatcherKeyword:
		raw := strings.TrimSpace(m.Value)
		v := strings.ToLower(raw)
		if v == "" {
			return false
		}
		if strings.Contains(lowerTag, v) {
			return true
		}
		// A value pasted from the UI before endpoints were hashed still names
		// exactly one node; convert it rather than letting the exclude quietly
		// stop matching.
		if canon := legacyTagValue(raw); canon != "" {
			return strings.Contains(lowerTag, strings.ToLower(canon))
		}
		return false
	case MatcherEmoji:
		v := strings.TrimSpace(m.Value)
		return v != "" && strings.Contains(tag, v)
	case MatcherCode:
		// A code describes where the node IS, which only the provider's display
		// name says. Matching the rest of the tag made the server's hostname
		// vote: nodes on "s4.usghq.ps1ksydn.com" all matched US via "usghq",
		// so Hong Kong and Taiwan joined a US-only filter and a urltest then
		// sent that traffic out through Hong Kong.
		name := strings.ToLower(displayName(tag))
		for _, syn := range CodeSynonyms(m.Value) {
			s := strings.ToLower(strings.TrimSpace(syn))
			if s == "" {
				continue
			}
			// ASCII synonyms must sit on word boundaries — "us" is a substring
			// of Belarus and Cyprus, "in" of Singapore. CJK synonyms and emoji
			// flags have no boundaries to find and are plain substrings.
			if isASCII(s) {
				if containsWord(name, s) {
					return true
				}
				continue
			}
			if strings.Contains(name, s) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// filterMatches reports whether a tag satisfies any of a Filter's matchers (OR
// semantics within a Filter).
func filterMatches(f *Filter, tag, lowerTag string) bool {
	for _, m := range f.Matchers {
		if matcherMatches(m, tag, lowerTag) {
			return true
		}
	}
	return false
}

// filterExcluded reports whether a tag matches any of a Filter's exclude rules
// (OR semantics). A tag that is excluded is kept out of the Filter even when it
// satisfied a matcher.
func filterExcluded(f *Filter, tag, lowerTag string) bool {
	for _, m := range f.Excludes {
		if matcherMatches(m, tag, lowerTag) {
			return true
		}
	}
	return false
}

// filterClaims reports whether a Filter should take ownership of a tag: it must
// match at least one matcher AND not match any exclude.
func filterClaims(f *Filter, tag, lowerTag string) bool {
	return filterMatches(f, tag, lowerTag) && !filterExcluded(f, tag, lowerTag)
}

// NodePool is the candidate set the matcher runs over.
//
// The split exists because not every legal group member may be collected
// automatically. Endpoints are real exit nodes: one that matches no Filter falls
// through to the fallback. OptIn tags (today the `direct` outbounds) join a
// Filter ONLY when its matchers name them, and never reach the fallback — a
// urltest that quietly acquired `direct` would elect it on every probe and route
// everything unproxied.
type NodePool struct {
	Endpoints []string
	OptIn     []string
}

// AssignFilters maps endpoint tags to Filters using multi-match semantics: an
// endpoint joins EVERY non-fallback Filter whose matchers it satisfies AND whose
// excludes it does not (an excluded tag is treated as not claimed by that
// Filter). Any endpoint that matches no non-fallback Filter is assigned to the fallback
// Filter (and also reported in `others` for preview/diagnostics).
//
// The returned membership is keyed by Filter ID; each value is the list of
// endpoint tags assigned to that Filter, in input order. Non-fallback Filters
// are evaluated in ascending (Priority, ID) order — order does not change
// membership under multi-match, but it keeps output deterministic and mirrors
// the UI ordering.
//
// Opt-in tags (pool.OptIn) are evaluated against the same matchers but are
// excluded from the fallback: an opt-in tag no Filter claims simply joins
// nothing, and is not reported in `others` either (it is not "unmatched" in the
// sense the preview means — it was never up for automatic collection).
//
// The function is pure: no I/O, no config dependency. It is safe to call for a
// dry-run preview.
func AssignFilters(pool NodePool, filters []*Filter) (membership map[string][]string, others []string) {
	membership = make(map[string][]string)

	// Partition into fallback (there should be exactly one) and the rest.
	var fallback *Filter
	ranked := make([]*Filter, 0, len(filters))
	for _, f := range filters {
		if f == nil {
			continue
		}
		if f.IsFallback {
			fallback = f
			continue
		}
		ranked = append(ranked, f)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Priority != ranked[j].Priority {
			return ranked[i].Priority < ranked[j].Priority
		}
		return ranked[i].ID < ranked[j].ID
	})

	for _, tag := range pool.Endpoints {
		lower := strings.ToLower(tag)
		matchedAny := false
		for _, f := range ranked {
			if filterClaims(f, tag, lower) {
				membership[f.ID] = append(membership[f.ID], tag)
				matchedAny = true
			}
		}
		if !matchedAny {
			others = append(others, tag)
			if fallback != nil {
				membership[fallback.ID] = append(membership[fallback.ID], tag)
			}
		}
	}

	// Opt-in tags: claimed only on an explicit match, never by the fallback.
	for _, tag := range pool.OptIn {
		lower := strings.ToLower(tag)
		for _, f := range ranked {
			if filterClaims(f, tag, lower) {
				membership[f.ID] = append(membership[f.ID], tag)
			}
		}
	}
	return membership, others
}
