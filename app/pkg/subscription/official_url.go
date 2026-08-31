package subscription

import (
	"net/url"
	"strings"
)

// A subscription's official site is the page an operator actually needs when
// the plan runs out — top up, renew, read the notice board. Providers publish
// it in two places, neither of them mandatory:
//
//  1. the `profile-web-page-url` response header, the convention Clash-family
//     clients read (and the one the panel trusts first, since it is a field
//     rather than a guess);
//  2. an "info node" whose label says so — "官网：https://example.com".
//
// Both are provider-controlled strings that end up in an href, so everything
// here funnels through NormalizeOfficialURL and anything that is not plain
// http(s) is dropped rather than rendered.

// siteLabelKeywords mark an info entry as carrying the official site, matched
// as a lowercase substring of the label (the part before the colon).
//
// Deliberately narrower than DefaultInfoLabelKeywords, which answers a
// different question ("is this entry metadata at all?"). A provider's 客服
// (support) or 群组 (chat group) entry is metadata and often holds a URL too,
// but it is not the site the "top up" click should go to.
var siteLabelKeywords = []string{
	// Chinese
	"官网", "官方网站", "官方站", "网址", "网站", "主页", "首页",
	// English
	"official", "website", "web site", "homepage", "home page", "site url",
}

// NormalizeOfficialURL turns a provider-supplied site string into a link that
// is safe to put in an href, or "" if it cannot be one.
//
// Only http and https survive. That is the security boundary of this feature:
// the value travels from a third-party feed straight into a clickable link, so
// javascript:, data: and friends must never reach the DOM. A bare domain
// ("example.com", "www.example.com") is accepted and promoted to https —
// providers write it that way constantly — but only when it actually looks
// like a hostname, so a stray label is not turned into a link.
func NormalizeOfficialURL(raw string) string {
	candidate := strings.TrimSpace(raw)
	// Providers wrap the value in brackets or trail it with punctuation.
	candidate = strings.Trim(candidate, "<>[](){}\"'` \t\r\n")
	candidate = strings.TrimRight(candidate, ".,;，。；、")
	if candidate == "" {
		return ""
	}
	// A URL never contains whitespace; a sentence that happens to mention one
	// does. Refusing the whole string is right — guessing which word is the
	// link is how a label becomes a wrong link.
	if strings.ContainsAny(candidate, " \t\r\n") {
		return ""
	}

	// Scheme-relative ("//example.com") and bare hosts both mean https here.
	if strings.HasPrefix(candidate, "//") {
		candidate = "https:" + candidate
	} else if !strings.Contains(candidate, "://") {
		// Check the host half only — "example.com/buy" is a bare domain with a
		// path, and the path must not disqualify it.
		host := candidate
		if slash := strings.IndexAny(host, "/?#"); slash != -1 {
			host = host[:slash]
		}
		if !looksLikeHost(host) {
			return ""
		}
		candidate = "https://" + candidate
	}

	u, err := url.Parse(candidate)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	if u.Host == "" {
		return ""
	}
	if !looksLikeHost(u.Hostname()) {
		return ""
	}
	// Lowercase the scheme only: a path can be case-sensitive.
	u.Scheme = strings.ToLower(u.Scheme)
	return u.String()
}

// looksLikeHost reports whether a string is plausibly a hostname: at least one
// dot, no path/scheme punctuation, and a non-numeric-looking last label.
// "example.com" and "sub.example.co.uk" pass; "4.7" and "剩余流量" do not.
func looksLikeHost(host string) bool {
	host = strings.TrimSuffix(host, ".")
	if host == "" || strings.ContainsAny(host, "/\\@ \t") {
		return false
	}
	dot := strings.LastIndex(host, ".")
	if dot <= 0 || dot == len(host)-1 {
		return false
	}
	tld := host[dot+1:]
	// A numeric last label means this is a version or a quota figure, not a
	// host. (A bare IP would also land here, and is not a provider's site.)
	for _, r := range tld {
		if r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

// DetectOfficialURL picks the site URL out of what a refresh learned: the
// `profile-web-page-url` header first, then the info entries whose label says
// they carry it. Returns "" when the feed says nothing usable.
func DetectOfficialURL(headerURL string, info []SubInfo) string {
	if u := NormalizeOfficialURL(headerURL); u != "" {
		return u
	}
	for _, entry := range info {
		if !isSiteLabel(entry.Key) {
			continue
		}
		if u := NormalizeOfficialURL(entry.Value); u != "" {
			return u
		}
	}
	return ""
}

// officialURLToPersist returns the site URL a refresh should write, or "" when
// it should write nothing.
//
// The "don't overwrite" rule lives here rather than inline at the call site
// because it is the surprising half of the feature: providers move domains and
// mirrors keep reporting the old one, so an operator who fixes the link by hand
// must be able to trust that the next refresh leaves it alone.
func officialURLToPersist(current, headerURL string, info []SubInfo) string {
	if strings.TrimSpace(current) != "" {
		return ""
	}
	return DetectOfficialURL(headerURL, info)
}

// isSiteLabel reports whether an info entry's label announces a site URL.
func isSiteLabel(label string) bool {
	lowered := strings.ToLower(strings.TrimSpace(label))
	if lowered == "" {
		return false
	}
	for _, kw := range siteLabelKeywords {
		if strings.Contains(lowered, kw) {
			return true
		}
	}
	return false
}
