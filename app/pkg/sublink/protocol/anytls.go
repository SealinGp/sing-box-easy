package protocol

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
)

// AnyTLS represents an AnyTLS outbound configuration.
//
// URL format: anytls://password@server:port/?params#tag
// Common params: sni, insecure / allowInsecure / skip-cert-verify, alpn, fp.
//
// AnyTLS is TLS-framed by definition, so the TLS block is enabled
// unconditionally — mirroring the Hysteria2 parser.
//
// Example:
//
//	anytls://pass@host:37231/?sni=cdn.example.com&insecure=1#TW-01
type AnyTLS struct {
	Tag string `json:"tag,omitempty"`
	option.AnyTLSOutboundOptions
}

func (a *AnyTLS) TypeName() string {
	return "anytls"
}

func (a *AnyTLS) Schema() string {
	return "anytls://"
}

func (a *AnyTLS) Parse(uri string) (*node.SubNode, error) {
	parsed, err := parseUserinfoURI(strings.TrimPrefix(uri, a.Schema()))
	if err != nil {
		return nil, fmt.Errorf("invalid anytls URI: %w", err)
	}

	a.Tag = parsed.Tag
	a.Password = parsed.Password
	a.Server = parsed.Server
	a.ServerPort = parsed.Port
	a.parseQueryParams(parsed.Params)

	return &node.SubNode{
		Type:    a.TypeName(),
		Tag:     a.Tag,
		Options: a.AnyTLSOutboundOptions,
	}, nil
}

func (a *AnyTLS) Validate() error {
	if a.Password == "" {
		return fmt.Errorf("anytls password is required")
	}
	if a.Server == "" {
		return fmt.Errorf("anytls server is required")
	}
	if a.ServerPort == 0 {
		return fmt.Errorf("anytls server port is required")
	}
	return nil
}

func (a *AnyTLS) parseQueryParams(params url.Values) {
	// AnyTLS is TLS-framed: the block is always present, even with no params.
	tls := &option.OutboundTLSOptions{Enabled: true}
	if params != nil {
		if sni := params.Get("sni"); sni != "" {
			tls.ServerName = sni
		} else if peer := params.Get("peer"); peer != "" {
			tls.ServerName = peer
		}
		if isTruthy(params.Get("insecure")) ||
			isTruthy(params.Get("allowInsecure")) ||
			isTruthy(params.Get("skip-cert-verify")) {
			tls.Insecure = true
		}
		if alpn := params.Get("alpn"); alpn != "" {
			tls.ALPN = strings.Split(alpn, ",")
		}
		if fp := params.Get("fp"); fp != "" {
			tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fp}
		}
	}
	a.OutboundTLSOptionsContainer.TLS = tls
}
