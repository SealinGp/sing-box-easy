package sublink

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/protocol"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
	"golang.org/x/net/dns/dnsmessage"
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

	// subscriptionFallbackUserAgent is the retry UA, used only after the
	// neutral one is refused. Some panels gate the other way round: they serve
	// a whitelist of known proxy clients and answer HTTP 404 to everything
	// else, so a neutral UA gets no subscription at all. v2rayN is the least
	// destructive client to impersonate — panels that negotiate on it return
	// the base64 URI list, not a client-native config — and a Clash profile is
	// now importable anyway (see the clash package).
	subscriptionFallbackUserAgent = "v2rayN/6.23"
)

// httpClient is a package-level req client with a bounded timeout.
// It is safe for concurrent use.
var httpClient = newBaseClient()

// newBaseClient builds a subscription HTTP client with the bounded timeout and
// the neutral User-Agent. Each non-default fetch mode gets its own client so
// per-subscription proxy/dialer settings don't leak into the shared one.
func newBaseClient() *req.Client {
	return req.C().
		SetTimeout(subscriptionFetchTimeout).
		SetUserAgent(subscriptionUserAgent)
}

// Fetch strategy values (mirror subscription.FetchMode* — kept here to avoid an
// import cycle; the auto-updater maps the stored mode string onto FetchOptions).
const (
	FetchModeDirect   = ""
	FetchModeCleanDNS = "clean_dns"
	FetchModeProxy    = "proxy"

	// defaultDoHServer is AliDNS DoH addressed by IP, so it needs no bootstrap
	// DNS and can't be poisoned — the right default for CN-based airports.
	defaultDoHServer = "https://223.5.5.5/dns-query"
)

// FetchOptions selects how a subscription URL is retrieved on a censored network.
type FetchOptions struct {
	Mode      string // FetchMode* ("" = direct)
	ProxyURL  string // for FetchModeProxy, e.g. "socks5://127.0.0.1:7893"
	DoHServer string // for FetchModeCleanDNS; defaultDoHServer when empty
}

// buildFetchClient returns the HTTP client for the requested fetch mode.
func buildFetchClient(opts FetchOptions) (*req.Client, error) {
	switch opts.Mode {
	case FetchModeDirect:
		return httpClient, nil
	case FetchModeProxy:
		proxy := strings.TrimSpace(opts.ProxyURL)
		if proxy == "" {
			return nil, fmt.Errorf("proxy fetch mode requires a proxy URL (e.g. socks5://127.0.0.1:7893)")
		}
		return newBaseClient().SetProxyURL(proxy), nil
	case FetchModeCleanDNS:
		doh := strings.TrimSpace(opts.DoHServer)
		if doh == "" {
			doh = defaultDoHServer
		}
		return newBaseClient().SetDial(cleanDNSDialer(doh)), nil
	default:
		return nil, fmt.Errorf("unknown subscription fetch mode: %q", opts.Mode)
	}
}

// cleanDNSDialer returns a DialContext that resolves the host over DoH and dials
// the resulting IP. TLS still uses the original host for SNI (req sets it from
// the request URL), so this fixes DNS poisoning without changing the handshake.
func cleanDNSDialer(dohServer string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		// Already an IP literal → nothing to resolve.
		if net.ParseIP(host) != nil {
			return dialer.DialContext(ctx, network, addr)
		}
		ip, err := resolveDoH(ctx, dohServer, host)
		if err != nil {
			return nil, fmt.Errorf("clean-dns resolve %q via %s: %w", host, dohServer, err)
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
	}
}

// dohClient performs the DoH lookup itself. Its target is an IP-form URL, so it
// needs no recursive DNS of its own.
var dohClient = &http.Client{Timeout: 10 * time.Second}

// resolveDoH resolves host's first A record via an RFC 8484 DoH endpoint.
func resolveDoH(ctx context.Context, server, host string) (string, error) {
	fqdn := host
	if !strings.HasSuffix(fqdn, ".") {
		fqdn += "."
	}
	name, err := dnsmessage.NewName(fqdn)
	if err != nil {
		return "", fmt.Errorf("invalid host %q: %w", host, err)
	}
	query := dnsmessage.Message{
		Header: dnsmessage.Header{RecursionDesired: true},
		Questions: []dnsmessage.Question{{
			Name:  name,
			Type:  dnsmessage.TypeA,
			Class: dnsmessage.ClassINET,
		}},
	}
	packed, err := query.Pack()
	if err != nil {
		return "", fmt.Errorf("pack dns query: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server, bytes.NewReader(packed))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	resp, err := dohClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("doh server returned http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	var answer dnsmessage.Message
	if err := answer.Unpack(body); err != nil {
		return "", fmt.Errorf("unpack dns answer: %w", err)
	}
	for _, ans := range answer.Answers {
		if a, ok := ans.Body.(*dnsmessage.AResource); ok {
			return net.IP(a.A[:]).String(), nil
		}
	}
	return "", fmt.Errorf("no A record for %s", host)
}

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
// part of the node list. Currently the raw `Subscription-Userinfo` response
// header (e.g. "upload=…; download=…; total=…; expire=…"), the cross-provider
// standard airports use to report account traffic and plan expiry — independent
// of the (provider-specific, localized) "info node" mechanism some feeds embed.
type FetchMeta struct {
	Userinfo string
}

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
	client, err := buildFetchClient(opts)
	if err != nil {
		return nil, &FetchMeta{}, err
	}

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
			modeNodes, m, err := l.fetchNodesWithMeta(line, client)
			// Proxy mode: if the proxied fetch fails (proxy down / no node), fall
			// back to a direct fetch so a working network still updates.
			if err != nil && opts.Mode == FetchModeProxy {
				logger.Warn("proxied subscription fetch failed, retrying direct",
					zap.String("url", line), zap.Error(err))
				modeNodes, m, err = l.fetchNodesWithMeta(line, httpClient)
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
			if meta.Userinfo == "" && m != nil && m.Userinfo != "" {
				meta.Userinfo = m.Userinfo
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

		// A pasted Clash profile. Only reachable for a multi-line paste that
		// the caller passed as one string, which is how the UI sends it.
		if parsed, err := l.parseNonBase64Body([]byte(line), "pasted content"); err == nil {
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

func (l *SubLink) fetchNodesWithMeta(sub_url string, client *req.Client) ([]*node.SubNode, *FetchMeta, error) {
	if err := validateSubscriptionURL(sub_url); err != nil {
		return nil, nil, err
	}
	if client == nil {
		client = httpClient
	}

	resp, err := client.R().Get(sub_url)
	if err != nil {
		return nil, nil, fmt.Errorf("http get failed: %w", err)
	}
	// UA gating: a 4xx here is often the panel refusing an unknown client
	// rather than a bad URL or expired token. Retry once as a known client
	// before giving up — see subscriptionFallbackUserAgent.
	if shouldRetryWithClientUA(resp.StatusCode) {
		logger.Warn("subscription fetch rejected, retrying with a client user-agent",
			zap.String("url", sub_url),
			zap.Int("status", resp.StatusCode))
		retry, retryErr := client.R().
			SetHeader("User-Agent", subscriptionFallbackUserAgent).
			Get(sub_url)
		if retryErr == nil && retry.StatusCode < 400 {
			resp = retry
		}
	}
	if resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("subscription server returned http %d", resp.StatusCode)
	}

	// Capture the standard account-metadata header before reading the body.
	// http.Header.Get canonicalizes the key, so the wire-lowercase
	// "subscription-userinfo" is matched here.
	meta := &FetchMeta{Userinfo: resp.Header.Get("Subscription-Userinfo")}

	respStr, err := resp.ToString()
	if err != nil {
		return nil, meta, fmt.Errorf("read response body failed: %w", err)
	}

	nodes, err := decodeSubscriptionBody(respStr)
	if err != nil {
		// Not base64. Before reporting that, try the two other shapes seen in
		// the wild: a Clash/Mihomo YAML profile, and a plain (un-encoded) URI
		// list. Both are common enough that failing here would reject a
		// perfectly usable subscription.
		if parsed, convErr := l.parseNonBase64Body([]byte(respStr), sub_url); convErr == nil {
			return parsed, meta, nil
		}
		preview := respStr
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		return nil, meta, fmt.Errorf("base64 decode failed (body preview: %q): %w", preview, err)
	}

	return l.parseBody(nodes), meta, nil
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
