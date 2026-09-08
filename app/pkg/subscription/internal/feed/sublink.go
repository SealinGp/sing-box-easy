package sublink

import (
	"context"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/fetch"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/parser"
	"go.uber.org/zap"
	"strings"
)

type SubLink struct {
}

// ListNodes parses a batch of inputs into SubNodes. Each input is either an
// http(s) subscription URL (fetched + base64-decoded into a URI list) or a
// single proxy URI line.
//
// Per-node parse errors are intentionally swallowed (a subscription with 200
// nodes shouldn't fail because 1 has a typo). Per-URL fetch errors used to be
// swallowed too, which made misconfigurations invisible — now we log them as
// warnings, and if the caller passed exactly one input and it failed wholesale,
// the error is surfaced so the route handler can show a real message.
func (l *SubLink) ListNodes(lines []string) ([]*node.SubNode, error) {
	nodes, _, err := l.ListNodesWithMeta(lines)
	return nodes, err
}

// FetchMeta carries subscription metadata captured during a fetch that is NOT
// part of the node list:
//
//   - Userinfo, the raw `Subscription-Userinfo` response header (e.g.
//     "upload=…; download=…; total=…; expire=…"), the cross-provider standard
//     airports use to report account traffic and plan expiry — independent of
//     the (provider-specific, localized) "info node" mechanism some feeds embed.
//   - SiteURL, the raw `profile-web-page-url` header: the provider's own page,
//     where an operator tops up or renews. Passed through unvalidated; the
//     subscription package decides what is safe to store and link.
type FetchMeta = fetch.Meta
type FetchOptions = fetch.FetchOptions

const (
	FetchModeDirect   = fetch.FetchModeDirect
	FetchModeProxy    = fetch.FetchModeProxy
	FetchModeCleanDNS = fetch.FetchModeCleanDNS
)

// ListNodesWithMeta is ListNodes plus the response metadata from the first
// subscription that supplies it. Single-line/single-URL callers (subscription
// refresh) use this to surface account traffic/expiry; ListNodes keeps the
// node-only signature for callers that don't care.
func (l *SubLink) ListNodesWithMeta(lines []string) ([]*node.SubNode, *FetchMeta, error) {
	return l.ListNodesWithMetaOpts(lines, FetchOptions{})
}

// ListNodesWithMetaOpts is ListNodesWithMeta with an explicit fetch strategy
// (direct / clean-DNS / proxy), used by the subscription refresh path to work
// around DNS poisoning or RST on censored networks.
func (l *SubLink) ListNodesWithMetaOpts(lines []string, opts FetchOptions) ([]*node.SubNode, *FetchMeta, error) {
	return l.Resolve(context.Background(), lines, opts)
}
func (l *SubLink) Resolve(ctx context.Context, lines []string, opts FetchOptions) ([]*node.SubNode, *FetchMeta, error) {

	nodes := make([]*node.SubNode, 0)
	meta := &FetchMeta{}
	var lastFetchErr error
	fetchAttempts := 0

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// 订阅链接 (http/https subscription URL): fetch + base64-decode.
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			fetchAttempts++
			body, m, err := fetch.Fetch(ctx, line, opts)
			var modeNodes []*node.SubNode
			if err == nil {
				modeNodes, err = parser.Parse(body)
				if err != nil && opts.Mode == FetchModeProxy {
					body, m, err = fetch.Fetch(ctx, line, FetchOptions{})
					if err == nil {
						modeNodes, err = parser.Parse(body)
					}
				}
			}

			if err != nil {
				lastFetchErr = err
				logger.Warn("subscription fetch failed",
					zap.String("url", line),
					zap.Error(err))
				continue
			}
			// Keep the first non-empty userinfo seen (subscription refresh
			// passes exactly one URL, so this is simply "this feed's userinfo").
			if m != nil {
				if meta.Userinfo == "" && m.Userinfo != "" {
					meta.Userinfo = m.Userinfo
				}
				if meta.SiteURL == "" && m.SiteURL != "" {
					meta.SiteURL = m.SiteURL
				}
			}
			nodes = append(nodes, modeNodes...)
			continue
		}

		// 单个节点 (direct proxy URI, e.g. vless:// vmess:// trojan://).
		if sub_node, err := parser.ParseURI(line); err == nil {
			nodes = append(nodes, sub_node)
			continue
		}

		// 粘贴的订阅内容: a base64-encoded subscription body pasted directly
		// (not a URL, not a single URI). Decode it and parse each URI line —
		// the same handling fetchNodes applies after downloading.
		if decoded, err := parser.DecodeBase64(line); err == nil &&
			strings.Contains(string(decoded), "://") {
			nodes = append(nodes, parser.ParseLines(decoded)...)
			continue
		}

		// A pasted Clash profile. Only reachable for a multi-line paste that
		// the caller passed as one string, which is how the UI sends it.
		if parsed, err := parser.ParsePlain([]byte(line)); err == nil {
			nodes = append(nodes, parsed...)
			continue
		}
		// Unrecognized line: skip silently (a 200-node paste shouldn't fail
		// because one entry is malformed).
	}

	// If every fetch failed and produced no nodes, surface the last fetch
	// error rather than returning an empty list with no diagnostic.
	if len(nodes) == 0 && fetchAttempts > 0 && lastFetchErr != nil {
		return nodes, meta, fmt.Errorf("subscription fetch failed: %w", lastFetchErr)
	}

	return nodes, meta, nil
}
