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

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/protocol"
	"github.com/imroc/req/v3"
)

const (
	// subscriptionFetchTimeout caps how long a single subscription fetch may
	// take. Subscriptions are typically a few KB of base64 text, so 30s is
	// generous; it also bounds the SSRF blast radius if validation is bypassed.
	subscriptionFetchTimeout = 30 * time.Second

	subscriptionUserAgent = "sing-box-easy/1.0"
)

// httpClient is a package-level req client with a bounded timeout.
// It is safe for concurrent use.
var httpClient = req.C().
	SetTimeout(subscriptionFetchTimeout).
	SetUserAgent(subscriptionUserAgent)

type SubLink struct {
}

func (l *SubLink) ListNodes(lines []string) ([]*node.SubNode, error) {
	nodes := make([]*node.SubNode, 0)
	for _, line := range lines {
		// 订阅链接
		if strings.Contains(line, "http") {
			modeNodes, err := l.fetchNodes(line)
			if err != nil {
				continue
			}
			nodes = append(nodes, modeNodes...)
			continue
		}

		//单个节点
		sub_node, err := l.parseNode(line)
		if err != nil {
			continue
		}
		nodes = append(nodes, sub_node)
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
		return nil, err
	}

	var sub_nodes []*node.SubNode
	respStr, err := resp.ToString()
	if err != nil {
		return nil, err
	}

	nodes, err := base64.StdEncoding.DecodeString(respStr)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(nodes))
	for scanner.Scan() {
		line := scanner.Text()

		sub_node, err := l.parseNode(line)
		if err != nil {
			continue
		}

		sub_nodes = append(sub_nodes, sub_node)
	}
	return sub_nodes, nil
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
