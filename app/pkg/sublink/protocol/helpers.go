package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// isTruthy reports whether a query-param value means "on" (1/true/yes).
func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// splitHostPort splits "host:port", supporting bracketed IPv6 ("[::1]:443").
func splitHostPort(serverInfo string) (host, port string, err error) {
	if strings.HasPrefix(serverInfo, "[") {
		closeBracket := strings.Index(serverInfo, "]")
		if closeBracket == -1 {
			return "", "", fmt.Errorf("invalid IPv6 address format")
		}
		host = serverInfo[1:closeBracket]
		remaining := serverInfo[closeBracket+1:]
		if !strings.HasPrefix(remaining, ":") {
			return "", "", fmt.Errorf("missing port after IPv6 address")
		}
		return host, remaining[1:], nil
	}

	lastColon := strings.LastIndex(serverInfo, ":")
	if lastColon == -1 {
		return "", "", fmt.Errorf("invalid server info: missing port")
	}
	return serverInfo[:lastColon], serverInfo[lastColon+1:], nil
}

// buildV2RayTransport builds a V2Ray transport block (ws/grpc/http) from URI
// query params. Shared by the VLESS/VMess/Trojan-style parsers. A "tcp"/"raw"
// type means no transport block and should be filtered by the caller.
func buildV2RayTransport(transportType string, params url.Values) (*option.V2RayTransportOptions, error) {
	transport := &option.V2RayTransportOptions{}

	switch transportType {
	case "ws", "websocket":
		transport.Type = "ws"
		ws := option.V2RayWebsocketOptions{}
		if path := params.Get("path"); path != "" {
			ws.Path = path
		}
		if host := params.Get("host"); host != "" {
			ws.Headers = map[string]badoption.Listable[string]{"Host": {host}}
		}
		if ed := params.Get("ed"); ed != "" {
			if edLen, err := strconv.Atoi(ed); err == nil && edLen > 0 {
				ws.MaxEarlyData = uint32(edLen)
				if edh := params.Get("edh"); edh != "" {
					ws.EarlyDataHeaderName = edh
				} else {
					ws.EarlyDataHeaderName = "Sec-WebSocket-Protocol"
				}
			}
		}
		transport.WebsocketOptions = ws

	case "grpc":
		transport.Type = "grpc"
		grpc := option.V2RayGRPCOptions{}
		if sn := params.Get("serviceName"); sn != "" {
			grpc.ServiceName = sn
		} else if path := params.Get("path"); path != "" {
			grpc.ServiceName = path
		}
		transport.GRPCOptions = grpc

	case "http", "h2", "http2":
		transport.Type = "http"
		httpOpts := option.V2RayHTTPOptions{}
		if host := params.Get("host"); host != "" {
			httpOpts.Host = strings.Split(host, ",")
		}
		if path := params.Get("path"); path != "" {
			httpOpts.Path = path
		}
		transport.HTTPOptions = httpOpts

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}

	return transport, nil
}
