package subscription

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
)

// Account metadata (traffic left, expiry, reset countdown) reaches us through
// three provider-dependent channels, none of which is guaranteed:
//
//  1. the standard `Subscription-Userinfo` HTTP response header (parseUserinfo),
//  2. "info nodes" pointed at a loopback address (isLoopbackNode),
//  3. "info nodes" that are ordinary, fully-routable proxy entries — usually
//     byte-identical clones of the first real node — distinguished only by their
//     display name (isInfoLabel). 良心云 does this.
//
// Cases 2 and 3 must be kept out of the config: they are not usable exits, and
// their names change on every refresh (the countdown ticks down), which would
// otherwise churn the outbound list on every update.

// loopbackServers are the non-routable hosts that mark a parsed node as an
// "info node" rather than a real proxy. A node pointed at one of these can never
// be a usable exit, so this is a robust, provider/language-agnostic signal —
// no need to match the (localized) metadata field names themselves.
var loopbackServers = map[string]struct{}{
	"127.0.0.1": {},
	"::1":       {},
	"localhost": {},
	"0.0.0.0":   {},
}

// DefaultInfoLabelKeywords are the words providers put in a metadata label.
// Matching is substring-based on the lowercased label (the part BEFORE the
// colon), so "剩余流量", "距离下次重置剩余" and "Remaining Traffic" all hit.
// Entries are lowercase; they are only ever consulted for names that already
// have the "<label><colon><value>" shape, which is what keeps region names such
// as "🇭🇰香港：01" out.
//
// This is only the built-in starting point: operators can override the list per
// deployment (Subscriptions page → info keywords), because every provider
// invents its own labels. An empty override falls back to these defaults.
var DefaultInfoLabelKeywords = []string{
	// Chinese
	"流量", "到期", "过期", "有效期", "重置", "套餐", "剩余", "已用", "用完",
	"总量", "订阅", "官网", "网址", "客服", "购买", "续费", "余额", "更新时间",
	// English
	"traffic", "expire", "expiry", "expiration", "reset", "remaining", "used",
	"usage", "total", "quota", "bandwidth", "package", "plan", "balance",
	"renew", "official", "website", "homepage", "subscription",
}

// InfoKeywordsProvider supplies the operator-configured metadata labels. It is
// an interface (satisfied by the settings manager) so this package stays free of
// a storage dependency and the updater remains testable. A nil provider — or one
// returning an empty list — means "use DefaultInfoLabelKeywords".
type InfoKeywordsProvider interface {
	GetSubscriptionInfoKeywords() []string
}

// NormalizeInfoKeywords returns a new, storage-ready list: trimmed, lowercased
// (matching is case-insensitive), de-duplicated, with empties dropped. The input
// slice is never modified.
func NormalizeInfoKeywords(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, kw := range in {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" {
			continue
		}
		if _, dup := seen[kw]; dup {
			continue
		}
		seen[kw] = struct{}{}
		out = append(out, kw)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// EffectiveInfoKeywords resolves the list actually used for matching: the
// operator's override when it holds anything, the built-in defaults otherwise.
func EffectiveInfoKeywords(configured []string) []string {
	if normalized := NormalizeInfoKeywords(configured); len(normalized) > 0 {
		return normalized
	}
	return DefaultInfoLabelKeywords
}

// isInfoNode reports whether a parsed node carries account metadata instead of a
// usable proxy endpoint — either by pointing at a loopback address or by wearing
// a metadata label drawn from keywords.
func isInfoNode(n *node.SubNode, keywords []string) bool {
	if n == nil {
		return false
	}
	return isLoopbackNode(n) || isInfoLabel(n.Tag, keywords)
}

// isLoopbackNode reports whether the node's server is a loopback/unspecified
// address, which no real exit ever uses.
func isLoopbackNode(n *node.SubNode) bool {
	server := config.GetOutboundServer(config.Outbound{Type: n.Type, Options: n.Options})
	_, ok := loopbackServers[strings.ToLower(strings.TrimSpace(server))]
	return ok
}

// isInfoLabel reports whether a display name reads as an account-metadata line.
// Two conditions must BOTH hold, which is what keeps real proxy names safe:
//
//   - the name splits into a non-empty "<label><colon><value>" pair, and
//   - the label mentions one of keywords (already lowercase).
//
// "剩余流量：4.7 TB" qualifies; "🇭🇰香港：01" (no keyword) and "高速流量节点"
// (no key/value shape) do not.
func isInfoLabel(name string, keywords []string) bool {
	entry := parseInfoEntry(name)
	if entry.Key == "" || entry.Value == "" {
		return false
	}
	label := strings.ToLower(entry.Key)
	for _, kw := range keywords {
		if strings.Contains(label, kw) {
			return true
		}
	}
	return false
}

// parseInfoEntry turns an info-node display name into a generic key/value pair by
// splitting on the FIRST colon — fullwidth "：" (U+FF1A) or ASCII ":". With no
// colon the whole name becomes the key and the value is empty. This keeps the
// feature provider-agnostic: any label ("剩余流量", "Expire", ...) is preserved
// verbatim rather than matched against a hardcoded set.
func parseInfoEntry(name string) SubInfo {
	idx := -1
	sepLen := 0
	for i, r := range name {
		if r == '：' || r == ':' {
			idx = i
			sepLen = len(string(r)) // 1 for ASCII ':'; 3 for fullwidth '：'
			break
		}
	}
	if idx < 0 {
		return SubInfo{Key: strings.TrimSpace(name)}
	}
	key := strings.TrimSpace(name[:idx])
	value := strings.TrimSpace(name[idx+sepLen:])
	return SubInfo{Key: key, Value: value}
}

// partitionNodes splits a fetched feed into real proxy nodes (kept for the
// config diff) and the metadata entries parsed from its info nodes. Operates on
// node.Tag, which at this point is the raw provider name (the "<server:port> |
// <subID>" suffix is only appended later inside diffNodes).
func partitionNodes(nodes []*node.SubNode, keywords []string) (real []*node.SubNode, info []SubInfo) {
	for _, n := range nodes {
		if isInfoNode(n, keywords) {
			info = append(info, parseInfoEntry(n.Tag))
			continue
		}
		real = append(real, n)
	}
	return real, info
}

// parseUserinfo turns a Subscription-Userinfo header into human-readable info
// entries. The header is the cross-provider standard
//
//	upload=<bytes>; download=<bytes>; total=<bytes>; expire=<unix-seconds>
//
// (any subset may be present). The four keys are a fixed spec — not localized
// per-provider labels — so formatting them here is safe. Returns nil for an
// empty/unparseable header.
func parseUserinfo(header string) []SubInfo {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil
	}

	vals := make(map[string]int64)
	for _, part := range strings.Split(header, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		n, err := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)
		if err != nil {
			continue
		}
		vals[key] = n
	}

	var out []SubInfo
	used := vals["upload"] + vals["download"]
	total, hasTotal := vals["total"]
	if used > 0 || hasTotal {
		out = append(out, SubInfo{Key: "Used", Value: humanizeBytes(used)})
	}
	// total == 0 conventionally means "unlimited" — show neither total nor
	// remaining in that case.
	if hasTotal && total > 0 {
		out = append(out, SubInfo{Key: "Total", Value: humanizeBytes(total)})
		if rem := total - used; rem >= 0 {
			out = append(out, SubInfo{Key: "Remaining", Value: humanizeBytes(rem)})
		}
	}
	if exp, ok := vals["expire"]; ok && exp > 0 {
		out = append(out, SubInfo{Key: "Expires", Value: time.Unix(exp, 0).Format("2006-01-02")})
	}
	return out
}

// humanizeBytes formats a byte count with binary (1024) units, matching how
// airports report quota (e.g. total=805306368000 → "750.00 GB").
func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
