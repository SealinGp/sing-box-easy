package fetch

import (
	"bytes"
	"context"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
	"golang.org/x/net/dns/dnsmessage"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Meta struct {
	Userinfo string
	SiteURL  string
}

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

func ValidateURL(raw string) error {
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

func FetchWithClient(ctx context.Context, sub_url string, client *req.Client) ([]byte, *Meta, error) {
	if err := ValidateURL(sub_url); err != nil {
		return nil, nil, err
	}
	if client == nil {
		client = httpClient
	}

	resp, err := client.R().SetContext(ctx).Get(sub_url)
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
		retry, retryErr := client.R().SetContext(ctx).
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
	meta := &Meta{
		Userinfo: resp.Header.Get("Subscription-Userinfo"),
		SiteURL:  resp.Header.Get("Profile-Web-Page-Url"),
	}

	respStr, err := resp.ToString()
	if err != nil {
		return nil, meta, fmt.Errorf("read response body failed: %w", err)
	}

	return []byte(respStr), meta, nil
}
func shouldRetryWithClientUA(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return false
	}
	return statusCode >= 400 && statusCode < 500
}

func Fetch(ctx context.Context, url string, opts FetchOptions) ([]byte, *Meta, error) {
	client, err := buildFetchClient(opts)
	if err != nil {
		return nil, nil, err
	}
	body, meta, err := FetchWithClient(ctx, url, client)
	if err != nil && opts.Mode == FetchModeProxy {
		return FetchWithClient(ctx, url, httpClient)
	}
	return body, meta, err
}
