package sublink

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/protocol"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

const (
	// subscriptionFetchTimeout caps how long a single subscription fetch may
	// take. Subscriptions are typically a few KB of base64 text, so 30s is
	// generous; it also bounds the SSRF blast radius if validation is bypassed.
	subscriptionFetchTimeout = 30 * time.Second

	// subscriptionUserAgent is the UA we send when fetching subscriptions.
	// IMPORTANT: must NOT contain "sing-box", "clash", "v2ray", "shadowrocket"
	// or other proxy-client substrings. Many subscription panels (V2Board,
	// SSPanel, etc.) do User-Agent content negotiation — if they detect a
	// known client name they return that client's native config format
	// (JSON for sing-box, YAML for clash, …) instead of the canonical
	// base64-encoded URI list this parser expects.
	subscriptionUserAgent = "sbe-fetcher/1.0"
)

// httpClient is a package-level req client with a bounded timeout.
// It is safe for concurrent use.
var httpClient = req.C().
	SetTimeout(subscriptionFetchTimeout).
	SetUserAgent(subscriptionUserAgent)

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
	nodes := make([]*node.SubNode, 0)
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
			modeNodes, err := l.fetchNodes(line)
			if err != nil {
				lastFetchErr = err
				logger.Warn("subscription fetch failed",
					zap.String("url", line),
					zap.Error(err))
				continue
			}
			nodes = append(nodes, modeNodes...)
			continue
		}

		// 单个节点 (direct proxy URI, e.g. vless:// vmess:// trojan://).
		if sub_node, err := l.parseNode(line); err == nil {
			nodes = append(nodes, sub_node)
			continue
		}

		// 粘贴的订阅内容: a base64-encoded subscription body pasted directly
		// (not a URL, not a single URI). Decode it and parse each URI line —
		// the same handling fetchNodes applies after downloading.
		if decoded, err := decodeSubscriptionBody(line); err == nil &&
			strings.Contains(string(decoded), "://") {
			nodes = append(nodes, l.parseBody(decoded)...)
			continue
		}
		// Unrecognized line: skip silently (a 200-node paste shouldn't fail
		// because one entry is malformed).
	}

	// If every fetch failed and produced no nodes, surface the last fetch
	// error rather than returning an empty list with no diagnostic.
	if len(nodes) == 0 && fetchAttempts > 0 && lastFetchErr != nil {
		return nodes, fmt.Errorf("subscription fetch failed: %w", lastFetchErr)
	}

	return nodes, nil
}

// validateSubscriptionURL rejects URLs that point at non-public addresses.
// Subscription URLs are user-supplied and travel through an HTTP client on
// the server, so an unrestricted URL is a server-side request forgery
// primitive (e.g. http://169.254.169.254/, http://127.0.0.1:5100/...).
//
// This is a best-effort pre-check: it blocks literal IP addresses in
// private/loopback/link-local ranges and a handful of well-known hostnames.
// It does NOT defend against DNS rebinding — that would require a custom
// dialer that re-checks the resolved IP after connect.
func validateSubscriptionURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme: %q (only http/https allowed)", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}

	// Block well-known loopback/metadata hostnames regardless of resolution.
	lowered := strings.ToLower(host)
	switch lowered {
	case "localhost", "ip6-localhost", "ip6-loopback":
		return fmt.Errorf("blocked host: %q", host)
	}

	// If host is a literal IP, reject private/loopback/link-local/multicast.
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("blocked ip address: %s", ip)
		}
	}

	return nil
}

func (l *SubLink) fetchNodes(sub_url string) ([]*node.SubNode, error) {
	if err := validateSubscriptionURL(sub_url); err != nil {
		return nil, err
	}

	resp, err := httpClient.R().Get(sub_url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("subscription server returned http %d", resp.StatusCode)
	}

	respStr, err := resp.ToString()
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	nodes, err := decodeSubscriptionBody(respStr)
	if err != nil {
		// Most common cause: the subscription server returned the wrong
		// format because it sniffed the User-Agent. See the comment on
		// subscriptionUserAgent for the mitigation.
		preview := respStr
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		return nil, fmt.Errorf("base64 decode failed (body preview: %q): %w", preview, err)
	}

	return l.parseBody(nodes), nil
}

// parseBody scans a decoded subscription body (newline-separated proxy URIs)
// and parses each into a SubNode. Per-line parse errors are swallowed so one
// malformed entry doesn't drop the whole subscription.
func (l *SubLink) parseBody(body []byte) []*node.SubNode {
	var sub_nodes []*node.SubNode
	scanner := bufio.NewScanner(bytes.NewReader(body))
	// Subscription bodies can be larger than bufio's default 64KB line buffer
	// once decoded — give the scanner a generous ceiling.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		sub_node, err := l.parseNode(line)
		if err != nil {
			continue
		}
		sub_nodes = append(sub_nodes, sub_node)
	}
	return sub_nodes
}

// decodeSubscriptionBody decodes a subscription server's response body.
// Standard subscriptions are StdEncoding base64, but in the wild we see:
//   - URL-safe base64 (- and _ instead of + and /)
//   - Missing padding
//   - Trailing whitespace/newlines
//
// Try the strict variant first (fast path), fall back to the permissive one.
func decodeSubscriptionBody(body string) ([]byte, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil, fmt.Errorf("empty response body")
	}

	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	return nil, fmt.Errorf("body is not valid base64 in any common variant")
}

func (l *SubLink) parseNode(line string) (*node.SubNode, error) {
	parser, err := protocol.NewParser(line)
	if err != nil {
		return nil, err
	}

	sub_node, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	return sub_node, nil
}
