// Package nodetag mints and recognises the tags of subscription nodes.
//
// A node is tagged `<name> <fingerprint> | <subID>`; the fingerprint is 8 hex
// characters of md5 over `server:port`. See CLAUDE.md "Subscription outbound
// tags" for why, and for the older shapes OutboundTagCandidates still matches.
package nodetag

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing/common/json"
)

// outboundOptionsAsMap normalizes outbound.Options into a generic map so the
// server/port extractors can work uniformly. The Options field can be either:
//
//   - a typed sing-box option struct (e.g. *option.VMessOutboundOptions),
//     produced by the protocol parsers and by sing-box's typed JSON registry
//     when loading config.json — this is the common case in this app;
//   - a generic map[string]interface{}, produced when callers unmarshal an
//     outbound with a plain json.Unmarshal (rare, mostly tests).
//
// Previously the helpers only handled the map case, so typed structs silently
// returned an empty server/port — which collapsed the subscription diff into a
// no-op (no adds, no updates, no deletes). JSON-round-tripping is the cheapest
// way to support both shapes without enumerating every sing-box outbound type.
func outboundOptionsAsMap(opts any) map[string]interface{} {
	if opts == nil {
		return nil
	}
	if m, ok := opts.(map[string]interface{}); ok {
		return m
	}
	// Use the sing-box JSON context so option types with context-sensitive
	// marshalers (e.g. version-gated fields) serialize the same way they do
	// when written to config.json.
	ctx := config.CreateContext(context.Background())
	data, err := json.MarshalContext(ctx, opts)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.UnmarshalContext(ctx, data, &m); err != nil {
		return nil
	}
	return m
}

// readServerPort extracts the server address (and optionally port) from a
// normalized options map. Handles the wireguard `peers[0]` special case.
func readServerPort(opts map[string]interface{}, outboundType string) (server, port string) {
	if opts == nil {
		return "", ""
	}

	if s, ok := opts["server"].(string); ok {
		server = s
	}

	// JSON numbers decode as float64 by default; tolerate both float64 and int
	// in case the map came from a different source.
	if p, ok := opts["server_port"].(float64); ok {
		port = fmt.Sprintf("%d", int(p))
	} else if p, ok := opts["server_port"].(int); ok {
		port = fmt.Sprintf("%d", p)
	}

	if outboundType == "wireguard" && server == "" {
		if peers, ok := opts["peers"].([]interface{}); ok && len(peers) > 0 {
			if peer, ok := peers[0].(map[string]interface{}); ok {
				if addr, ok := peer["address"].(string); ok {
					server = addr
				}
				if p, ok := peer["port"].(float64); ok {
					port = fmt.Sprintf("%d", int(p))
				} else if p, ok := peer["port"].(int); ok {
					port = fmt.Sprintf("%d", p)
				}
			}
		}
	}
	return server, port
}

// GetOutboundServerKey returns the server endpoint key (server:port) for the outbound
// This identifies which server the node connects to.
func GetOutboundServerKey(outbound config.Outbound) string {
	server, port := readServerPort(outboundOptionsAsMap(outbound.Options), outbound.Type)
	if server != "" && port != "" {
		return fmt.Sprintf("%s:%s", server, port)
	}
	if server != "" {
		return server
	}
	return ""
}

// GetOutboundServer returns just the server field from the outbound
// This retrieves only the server address without the port.
func GetOutboundServer(outbound config.Outbound) string {
	server, _ := readServerPort(outboundOptionsAsMap(outbound.Options), outbound.Type)
	return server
}

// endpointFingerprintLen is how much of the endpoint hash goes into a tag.
//
// 8 hex characters is 32 bits. The fingerprint only has to separate nodes that
// share a display name inside ONE subscription — a few hundred at the very
// most — so a collision is on the order of one in ten million there, while a
// full 32-character md5 would make the tag LONGER than the "host:port" it
// replaces (26 characters for a typical provider) and defeat the point.
const endpointFingerprintLen = 8

// FingerprintEndpointKey hashes a "server:port" endpoint key into a short,
// stable token for use inside an outbound tag.
//
// md5 is used as a fast identity function, not as a security primitive: the
// input is a public hostname and the output is a display discriminator. Nothing
// here relies on collision resistance against an adversary.
func FingerprintEndpointKey(key string) string {
	if key == "" {
		return ""
	}
	sum := md5.Sum([]byte(key)) // #nosec G401 -- identity key, not a credential
	return hex.EncodeToString(sum[:])[:endpointFingerprintLen]
}

// GenerateFingerprintedTag is GenerateUniqueTag with the endpoint hashed rather
// than spelled out: "<name> <fingerprint>" instead of "<name> <server:port>".
//
// Subscription tags use this because they are numerous and long, and because
// the hostname in the tag was actively harmful — a `code` matcher reading the
// whole tag treated the server's name as evidence of the node's country.
// Manually added outbounds keep the readable form: there are few of them and
// the endpoint is the useful thing to see.
func GenerateFingerprintedTag(originalTag string, outbound config.Outbound) string {
	if fp := FingerprintEndpointKey(GetOutboundServerKey(outbound)); fp != "" {
		return fmt.Sprintf("%s %s", originalTag, fp)
	}
	return originalTag
}

// OutboundTagCandidates returns the tag a newly added outbound should carry
// (first) followed by the forms the same node may ALREADY be stored under.
//
// The second form exists because the endpoint used to be spelled out. Adding
// nodes is idempotent — pasting the same links twice skips them — and that
// check compares tags, so without the legacy form every previously added node
// would come back as a second copy under a new tag the first time links were
// re-pasted after the format change.
//
// Only these two forms are candidates, on purpose. A bare display name is NOT
// one: two different servers legitimately share a name ("香港 01" from two
// providers), so treating the name alone as identity would silently drop the
// second as a duplicate.
func OutboundTagCandidates(originalTag string, outbound config.Outbound) []string {
	fingerprinted := GenerateFingerprintedTag(originalTag, outbound)
	readable := GenerateUniqueTag(originalTag, outbound)
	if readable == fingerprinted {
		return []string{fingerprinted}
	}
	return []string{fingerprinted, readable}
}

// GenerateUniqueTag returns the readable "<name> <server:port>" form.
//
// This is the shape the panel minted before endpoints were hashed. New
// outbounds get GenerateFingerprintedTag instead; this remains the way to
// recognize a tag minted by an older build (see OutboundTagCandidates), and
// the readable form for anything that wants to show an endpoint.
func GenerateUniqueTag(originalTag string, outbound config.Outbound) string {
	serverKey := GetOutboundServerKey(outbound)
	if serverKey != "" {
		return fmt.Sprintf("%s %s", originalTag, serverKey)
	}
	return originalTag
}
