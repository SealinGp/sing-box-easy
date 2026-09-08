package parser

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/clash"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/feed/protocol"
	"strings"
)

// Parse decodes a feed without networking or persistence.
func Parse(body []byte) ([]*node.SubNode, error) {
	if decoded, err := DecodeBase64(string(body)); err == nil {
		return ParseLines(decoded), nil
	}
	return ParsePlain(body)
}
func ParsePlain(body []byte) ([]*node.SubNode, error) {
	if clash.Detect(body) {
		nodes, _, err := clash.Parse(body)
		return nodes, err
	}
	if nodes := ParseLines(body); len(nodes) > 0 {
		return nodes, nil
	}
	return nil, fmt.Errorf("body is neither base64, a clash profile, nor a uri list")
}
func ParseLines(body []byte) []*node.SubNode {
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
		sub_node, err := ParseURI(line)
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
func DecodeBase64(body string) ([]byte, error) {
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

func ParseURI(line string) (*node.SubNode, error) {
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
