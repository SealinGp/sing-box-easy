package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// Trojan represents a Trojan outbound configuration
// URL format: trojan://password@server:port?params#tag
// Example: trojan://603dc34d-1070-4e59-93ff-59f98250efe5@lm-allnodes.lma1b2.com:53438?allowInsecure=1#新加坡 08
type Trojan struct {
	Tag string `json:"tag,omitempty"`
	option.TrojanOutboundOptions
}

func (t *Trojan) TypeName() string {
	return "trojan"
}

func (t *Trojan) Schema() string {
	return "trojan://"
}

func (t *Trojan) Parse(uri string) (*node.SubNode, error) {
	// Remove the prefix
	trojanData := strings.TrimPrefix(uri, t.Schema())

	// Split by # to get the node name (tag)
	parts := strings.SplitN(trojanData, "#", 2)
	trojanInfo := parts[0]

	// Decode tag if present
	if len(parts) == 2 {
		tag, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode tag: %w", err)
		}
		t.Tag = tag
	}

	// Split by ? to get query parameters
	infoParts := strings.SplitN(trojanInfo, "?", 2)
	baseInfo := infoParts[0]
	var params url.Values
	if len(infoParts) == 2 {
		var err error
		params, err = url.ParseQuery(infoParts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse query parameters: %w", err)
		}
	}

	// Parse password@server:port
	atIndex := strings.LastIndex(baseInfo, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid trojan URI format: missing @ separator")
	}

	password := baseInfo[:atIndex]
	serverInfo := baseInfo[atIndex+1:]

	// URL decode the password
	password, err := url.QueryUnescape(password)
	if err != nil {
		return nil, fmt.Errorf("failed to decode password: %w", err)
	}
	t.Password = password

	// Parse server and port
	err = t.parseServerInfo(serverInfo)
	if err != nil {
		return nil, err
	}

	// Parse query parameters
	if params != nil {
		err = t.parseQueryParams(params)
		if err != nil {
			return nil, err
		}
	}

	sn := &node.SubNode{
		Type:    t.TypeName(),
		Tag:     t.Tag,
		Options: t.TrojanOutboundOptions,
	}

	return sn, nil
}

func (t *Trojan) Validate() error {
	if t.Password == "" {
		return fmt.Errorf("trojan password is required")
	}
	if t.Server == "" {
		return fmt.Errorf("trojan server is required")
	}
	if t.ServerPort == 0 {
		return fmt.Errorf("trojan server port is required")
	}
	return nil
}

// parseServerInfo parses server:port from the server info string
func (t *Trojan) parseServerInfo(serverInfo string) error {
	// Handle IPv6 addresses (enclosed in brackets)
	if strings.HasPrefix(serverInfo, "[") {
		closeBracket := strings.Index(serverInfo, "]")
		if closeBracket == -1 {
			return fmt.Errorf("invalid IPv6 address format")
		}

		server := serverInfo[1:closeBracket]
		remaining := serverInfo[closeBracket+1:]

		if !strings.HasPrefix(remaining, ":") {
			return fmt.Errorf("missing port after IPv6 address")
		}

		portStr := remaining[1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port number: %w", err)
		}
		t.Server = server
		t.ServerPort = uint16(port)
		return nil
	}

	// Handle IPv4 addresses and hostnames
	lastColon := strings.LastIndex(serverInfo, ":")
	if lastColon == -1 {
		return fmt.Errorf("invalid server info: missing port")
	}

	server := serverInfo[:lastColon]
	portStr := serverInfo[lastColon+1:]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port number out of range: %d", port)
	}

	t.Server = server
	t.ServerPort = uint16(port)
	return nil
}

// parseQueryParams parses query parameters and sets appropriate options
func (t *Trojan) parseQueryParams(params url.Values) error {
	// TLS options - Trojan typically uses TLS by default
	// Initialize TLS with default enabled state
	t.OutboundTLSOptionsContainer.TLS = &option.OutboundTLSOptions{
		Enabled: true,
	}

	// Handle SNI
	if sni := params.Get("sni"); sni != "" {
		t.OutboundTLSOptionsContainer.TLS.ServerName = sni
	}

	// Handle allowInsecure / insecure / skipCertVerify
	if allowInsecure := params.Get("allowInsecure"); allowInsecure == "1" || allowInsecure == "true" {
		t.OutboundTLSOptionsContainer.TLS.Insecure = true
	} else if insecure := params.Get("insecure"); insecure == "1" || insecure == "true" {
		t.OutboundTLSOptionsContainer.TLS.Insecure = true
	} else if skipVerify := params.Get("skipCertVerify"); skipVerify == "1" || skipVerify == "true" {
		t.OutboundTLSOptionsContainer.TLS.Insecure = true
	}

	// Handle ALPN
	if alpn := params.Get("alpn"); alpn != "" {
		alpnList := strings.Split(alpn, ",")
		t.OutboundTLSOptionsContainer.TLS.ALPN = alpnList
	}

	// Handle fingerprint
	if fp := params.Get("fp"); fp != "" {
		t.OutboundTLSOptionsContainer.TLS.UTLS = &option.OutboundUTLSOptions{
			Enabled:     true,
			Fingerprint: fp,
		}
	}

	// Handle transport type
	transportType := params.Get("type")
	if transportType == "" {
		transportType = params.Get("net") // Alternative parameter name
	}

	if transportType != "" && transportType != "tcp" {
		transport, err := t.buildTransport(transportType, params)
		if err != nil {
			return err
		}
		t.Transport = transport
	}

	// Handle network (tcp/udp)
	if network := params.Get("network"); network != "" {
		networks := strings.Split(network, ",")
		t.Network = option.NetworkList(strings.Join(networks, "\n"))
	}

	return nil
}

// buildTransport creates transport options based on network type
func (t *Trojan) buildTransport(transportType string, params url.Values) (*option.V2RayTransportOptions, error) {
	transport := &option.V2RayTransportOptions{}

	switch transportType {
	case "ws", "websocket":
		transport.Type = "ws"
		wsOptions := option.V2RayWebsocketOptions{}

		if path := params.Get("path"); path != "" {
			wsOptions.Path = path
		}

		if host := params.Get("host"); host != "" {
			wsOptions.Headers = map[string]badoption.Listable[string]{
				"Host": {host},
			}
		}

		// Handle early data
		if earlyData := params.Get("ed"); earlyData != "" {
			if edLen, err := strconv.Atoi(earlyData); err == nil && edLen > 0 {
				wsOptions.MaxEarlyData = uint32(edLen)
				if edHeader := params.Get("edh"); edHeader != "" {
					wsOptions.EarlyDataHeaderName = edHeader
				} else {
					wsOptions.EarlyDataHeaderName = "Sec-WebSocket-Protocol"
				}
			}
		}

		transport.WebsocketOptions = wsOptions

	case "grpc":
		transport.Type = "grpc"
		grpcOptions := option.V2RayGRPCOptions{}

		if serviceName := params.Get("serviceName"); serviceName != "" {
			grpcOptions.ServiceName = serviceName
		} else if path := params.Get("path"); path != "" {
			grpcOptions.ServiceName = path
		}

		// Handle gRPC mode
		if mode := params.Get("mode"); mode == "multi" {
			// multi mode is not directly configurable in sing-box
			// but can be handled through other options if needed
		}

		transport.GRPCOptions = grpcOptions

	case "http":
		transport.Type = "http"
		httpOptions := option.V2RayHTTPOptions{}

		if host := params.Get("host"); host != "" {
			httpOptions.Host = []string{host}
		}

		if path := params.Get("path"); path != "" {
			httpOptions.Path = path
		}

		transport.HTTPOptions = httpOptions

	case "h2", "http2":
		transport.Type = "http"
		httpOptions := option.V2RayHTTPOptions{}

		if host := params.Get("host"); host != "" {
			httpOptions.Host = strings.Split(host, ",")
		}

		if path := params.Get("path"); path != "" {
			httpOptions.Path = path
		}

		transport.HTTPOptions = httpOptions

	case "quic":
		transport.Type = "quic"
		quicOptions := option.V2RayQUICOptions{}

		if security := params.Get("security"); security != "" {
			// QUIC security options
		}

		transport.QUICOptions = quicOptions

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}

	return transport, nil
}
