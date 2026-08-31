package noderules

import (
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

// A subscription-minted outbound tag is "<name> <endpoint> | <subID>": the
// provider's display name, a per-node endpoint discriminator that makes the tag
// unique, and the owning subscription's ID. Only the name describes the node's
// region, so that is the part a `code` matcher is allowed to see.
//
// The alternative — matching the whole tag — is what let 🇭🇰 nodes on the host
// s4.usghq.ps1ksydn.com join a US-only filter, because "usghq" contains "us".
// A urltest then elected one of them and AI traffic left through Hong Kong.
const ownershipSeparator = " | "

// displayName strips the machine-generated parts of a subscription-minted tag,
// leaving the provider's own name.
//
// Both halves are optional and are removed only when they are recognizable, so
// a hand-written tag ("relay_bwh_us1") or a bare provider name passes through
// untouched — over-trimming would silently narrow what an operator's matchers
// can see.
func displayName(tag string) string {
	name := tag
	if i := strings.LastIndex(name, ownershipSeparator); i != -1 {
		name = name[:i]
	}
	if space := strings.LastIndex(name, " "); space != -1 {
		if isEndpointToken(name[space+1:]) {
			name = name[:space]
		}
	}
	return strings.TrimSpace(name)
}

// isEndpointToken reports whether a trailing token is the endpoint
// discriminator rather than part of the provider's name. Two shapes exist: the
// legacy "host:port" (or bare host) and the fingerprint that replaced it.
func isEndpointToken(token string) bool {
	if token == "" {
		return false
	}
	if isFingerprint(token) {
		return true
	}
	if host, port, ok := strings.Cut(token, ":"); ok {
		return isHostish(host) && isAllDigits(port)
	}
	return isHostish(token)
}

// isFingerprint reports whether a token is the hex endpoint fingerprint.
func isFingerprint(token string) bool {
	if len(token) < 8 || len(token) > 32 {
		return false
	}
	for _, r := range token {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// isHostish reports whether a token looks like a hostname or IP literal: a dot,
// and nothing but host-legal characters. Requiring the dot is what keeps a
// one-word provider name from being mistaken for a bare host.
func isHostish(token string) bool {
	if !strings.Contains(token, ".") {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// containsWord reports whether needle occurs in haystack delimited by
// non-letters on both sides.
//
// This is the second half of the same bug: a two-letter code is a substring of
// ordinary words, so "us" matched Belarus and Cyprus, "in" matched Singapore
// and link, "de" matched Sweden. Digits and punctuation still count as
// boundaries, so "US-01", "us_west" and "hk01" match as an operator expects.
//
// Only meaningful for ASCII needles: CJK synonyms have no word boundaries to
// find, and are matched as plain substrings by the caller.
func containsWord(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	for offset := 0; ; {
		idx := strings.Index(haystack[offset:], needle)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(needle)
		if !isASCIILetterAt(haystack, start-1) && !isASCIILetterAt(haystack, end) {
			return true
		}
		offset = start + 1
		if offset >= len(haystack) {
			return false
		}
	}
}

// isASCIILetterAt reports whether the byte at i is an ASCII letter. Out-of-range
// indices (either end of the string) are boundaries, not letters.
func isASCIILetterAt(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	c := s[i]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isASCII reports whether a synonym is pure ASCII, and therefore subject to
// word-boundary matching. A synonym carrying CJK or an emoji flag is not.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// legacyTagValue converts a matcher value that IS a full pre-fingerprint tag
// ("<name> <server:port> | <subID>") into the current form
// ("<name> <fingerprint> | <subID>"), or returns "" when the value is not one.
//
// Operators write excludes by pasting a tag out of the UI, so the stored value
// pins one exact node. When the tag format changed, those stored values stopped
// matching anything — and an exclude that matches nothing does not fail loudly,
// it silently re-admits the node it was written to remove. Converting at match
// time keeps them working without rewriting saved rules, and is exact: the same
// node always hashes to the same fingerprint.
func legacyTagValue(value string) string {
	sep := strings.LastIndex(value, ownershipSeparator)
	if sep == -1 {
		return ""
	}
	head := value[:sep]
	space := strings.LastIndex(head, " ")
	if space == -1 {
		return ""
	}
	endpoint := head[space+1:]
	if !isEndpointToken(endpoint) || isFingerprint(endpoint) {
		return ""
	}
	fp := config.FingerprintEndpointKey(endpoint)
	if fp == "" {
		return ""
	}
	return head[:space+1] + fp + value[sep:]
}
