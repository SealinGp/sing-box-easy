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

// userinfoURI is the decomposed form of a "password@host:port" style link
// (hysteria2, anytls): the shape where the userinfo half is a single secret
// rather than the method:password or uuid pairs the other schemes use.
type userinfoURI struct {
	Password string
	Server   string
	Port     uint16
	Params   url.Values
	Tag      string
}

// parseUserinfoURI splits "password@host:port[/path][?query][#tag]".
//
// Order matters and is the reason this is shared rather than copied: the path
// must be stripped from the AUTHORITY, not from the whole string. Scanning the
// raw link for the first "/" hits a "/" inside the password — legal, and common
// since these passwords are frequently raw base64 — and truncates away the
// "@host:port" that follows, so the link is then rejected as having no "@".
func parseUserinfoURI(data string) (userinfoURI, error) {
	var out userinfoURI

	// 1. Fragment (#tag) — last component, so it comes off first.
	if hash := strings.Index(data, "#"); hash != -1 {
		tag, err := url.QueryUnescape(data[hash+1:])
		if err != nil {
			return out, fmt.Errorf("failed to decode tag: %w", err)
		}
		out.Tag = tag
		data = data[:hash]
	}

	// 2. Query (?k=v). A literal "?" in a password is not valid URI syntax —
	// it has to be percent-encoded — so the first one always starts the query.
	if q := strings.Index(data, "?"); q != -1 {
		params, err := url.ParseQuery(data[q+1:])
		if err != nil {
			return out, fmt.Errorf("failed to parse query parameters: %w", err)
		}
		out.Params = params
		data = data[:q]
	}

	// 3. Userinfo. LastIndex, so an "@" inside the password loses to the real
	// separator.
	at := strings.LastIndex(data, "@")
	if at == -1 {
		return out, fmt.Errorf("missing @ separator")
	}
	password, err := url.QueryUnescape(data[:at])
	if err != nil {
		return out, fmt.Errorf("failed to decode password: %w", err)
	}
	out.Password = password
	authority := data[at+1:]

	// 4. Path — only now, once the password can no longer be mistaken for one.
	if slash := strings.Index(authority, "/"); slash != -1 {
		authority = authority[:slash]
	}

	server, portStr, err := splitHostPort(authority)
	if err != nil {
		return out, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return out, fmt.Errorf("invalid port number: %w", err)
	}
	if port < 1 || port > 65535 {
		return out, fmt.Errorf("port number out of range: %d", port)
	}
	out.Server = server
	out.Port = uint16(port)

	return out, nil
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
