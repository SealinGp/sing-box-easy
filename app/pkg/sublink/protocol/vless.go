package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
)

// VLESS represents a VLESS outbound configuration.
//
// URL format: vless://uuid@server:port?params#tag
// Common params: type (tcp/ws/grpc/http), security (none/tls/reality),
// sni, host, path, flow, pbk, sid, fp, alpn, insecure.
//
// Examples:
//   vless://uuid@host:443?type=ws&security=tls&host=h.example&path=%2Fp&sni=s.example&fp=chrome#JP-01
//   vless://uuid@host:443?type=tcp&security=reality&flow=xtls-rprx-vision&pbk=KEY&sid=ID&sni=s.example&fp=ios#US-01
type VLESS struct {
	Tag string `json:"tag,omitempty"`
	option.VLESSOutboundOptions
}

func (v *VLESS) TypeName() string {
	return "vless"
}

func (v *VLESS) Schema() string {
	return "vless://"
}

func (v *VLESS) Parse(uri string) (*node.SubNode, error) {
	data := strings.TrimPrefix(uri, v.Schema())

	// Split off the #tag fragment.
	parts := strings.SplitN(data, "#", 2)
	info := parts[0]
	if len(parts) == 2 {
		tag, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode tag: %w", err)
		}
		v.Tag = tag
	}

	// Split off the ?query.
	infoParts := strings.SplitN(info, "?", 2)
	baseInfo := infoParts[0]
	var params url.Values
	if len(infoParts) == 2 {
		var err error
		params, err = url.ParseQuery(infoParts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse query parameters: %w", err)
		}
	}

	// uuid@server:port
	atIndex := strings.LastIndex(baseInfo, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid vless URI format: missing @ separator")
	}
	uuid, err := url.QueryUnescape(baseInfo[:atIndex])
	if err != nil {
		return nil, fmt.Errorf("failed to decode uuid: %w", err)
	}
	v.UUID = uuid

	if err := v.parseServerInfo(baseInfo[atIndex+1:]); err != nil {
		return nil, err
	}

	if params != nil {
		if err := v.parseQueryParams(params); err != nil {
			return nil, err
		}
	}

	return &node.SubNode{
		Type:    v.TypeName(),
		Tag:     v.Tag,
		Options: v.VLESSOutboundOptions,
	}, nil
}

// parseServerInfo parses server:port, supporting bracketed IPv6.
func (v *VLESS) parseServerInfo(serverInfo string) error {
	server, portStr, err := splitHostPort(serverInfo)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number out of range: %d", port)
	}
	v.Server = server
	v.ServerPort = uint16(port)
	return nil
}

func (v *VLESS) parseQueryParams(params url.Values) error {
	// flow (e.g. xtls-rprx-vision)
	if flow := params.Get("flow"); flow != "" {
		v.Flow = flow
	}

	// network (tcp/udp restriction)
	if network := params.Get("network"); network != "" {
		v.Network = option.NetworkList(strings.ReplaceAll(network, ",", "\n"))
	}

	security := params.Get("security")
	switch security {
	case "tls", "reality", "xtls":
		tls := &option.OutboundTLSOptions{Enabled: true}

		if sni := params.Get("sni"); sni != "" {
			tls.ServerName = sni
		} else if host := params.Get("host"); host != "" {
			tls.ServerName = host
		}

		if isTruthy(params.Get("insecure")) || isTruthy(params.Get("allowInsecure")) {
			tls.Insecure = true
		}

		if alpn := params.Get("alpn"); alpn != "" {
			tls.ALPN = strings.Split(alpn, ",")
		}

		if fp := params.Get("fp"); fp != "" {
			tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fp}
		}

		// Reality: public key (pbk) + short id (sid). uTLS is required for
		// reality, so default a fingerprint if the link omitted fp.
		if pbk := params.Get("pbk"); pbk != "" {
			tls.Reality = &option.OutboundRealityOptions{
				Enabled:   true,
				PublicKey: pbk,
				ShortID:   params.Get("sid"),
			}
			if tls.UTLS == nil {
				tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: "chrome"}
			}
		}

		v.OutboundTLSOptionsContainer.TLS = tls
	}

	// transport (ws/grpc/http); "tcp" / "" => no transport block
	transportType := params.Get("type")
	if transportType == "" {
		transportType = params.Get("net")
	}
	if transportType != "" && transportType != "tcp" && transportType != "raw" {
		transport, err := buildV2RayTransport(transportType, params)
		if err != nil {
			return err
		}
		v.Transport = transport
	}

	return nil
}
