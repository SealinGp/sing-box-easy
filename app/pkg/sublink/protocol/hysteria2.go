package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
)

// Hysteria2 represents a Hysteria2 outbound configuration.
//
// URL format: hysteria2://password@server:port/?params#tag
// Common params: sni, insecure, obfs, obfs-password, mport (port hopping).
//
// Example:
//
//	hysteria2://pass@host:60000/?insecure=1&sni=s.example&mport=60000-65530#JP-01
type Hysteria2 struct {
	Tag string `json:"tag,omitempty"`
	option.Hysteria2OutboundOptions
}

func (h *Hysteria2) TypeName() string {
	return "hysteria2"
}

func (h *Hysteria2) Schema() string {
	return "hysteria2://"
}

func (h *Hysteria2) Parse(uri string) (*node.SubNode, error) {
	data := strings.TrimPrefix(uri, h.Schema())

	// Split off the #tag fragment.
	parts := strings.SplitN(data, "#", 2)
	info := parts[0]
	if len(parts) == 2 {
		tag, err := url.QueryUnescape(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode tag: %w", err)
		}
		h.Tag = tag
	}

	// A "/" may separate authority from the query (".../?k=v"); strip the path.
	if slash := strings.Index(info, "/"); slash != -1 {
		// Keep the query that follows "/?": move it back onto info sans path.
		q := ""
		if qIdx := strings.Index(info, "?"); qIdx != -1 {
			q = info[qIdx:]
			info = info[:slash] + q
		} else {
			info = info[:slash]
		}
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

	// password@server:port
	atIndex := strings.LastIndex(baseInfo, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid hysteria2 URI format: missing @ separator")
	}
	password, err := url.QueryUnescape(baseInfo[:atIndex])
	if err != nil {
		return nil, fmt.Errorf("failed to decode password: %w", err)
	}
	h.Password = password

	server, portStr, err := splitHostPort(baseInfo[atIndex+1:])
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port number: %w", err)
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port number out of range: %d", port)
	}
	h.Server = server
	h.ServerPort = uint16(port)

	if params != nil {
		h.parseQueryParams(params)
	}

	return &node.SubNode{
		Type:    h.TypeName(),
		Tag:     h.Tag,
		Options: h.Hysteria2OutboundOptions,
	}, nil
}

func (h *Hysteria2) Validate() error {
	if h.Password == "" {
		return fmt.Errorf("hysteria2 password is required")
	}
	if h.Server == "" {
		return fmt.Errorf("hysteria2 server is required")
	}
	if h.ServerPort == 0 {
		return fmt.Errorf("hysteria2 server port is required")
	}
	return nil
}

func (h *Hysteria2) parseQueryParams(params url.Values) {
	// Hysteria2 always runs over TLS.
	tls := &option.OutboundTLSOptions{Enabled: true}
	if sni := params.Get("sni"); sni != "" {
		tls.ServerName = sni
	}
	if isTruthy(params.Get("insecure")) || isTruthy(params.Get("allowInsecure")) {
		tls.Insecure = true
	}
	if alpn := params.Get("alpn"); alpn != "" {
		tls.ALPN = strings.Split(alpn, ",")
	}
	h.OutboundTLSOptionsContainer.TLS = tls

	// Salamander obfuscation.
	if obfs := params.Get("obfs"); obfs != "" {
		h.Obfs = &option.Hysteria2Obfs{
			Type:     obfs,
			Password: params.Get("obfs-password"),
		}
	}

	// Port hopping: "mport=60000-65530" (or comma-separated ranges) maps to
	// sing-box server_ports, which uses "start:end".
	if mport := params.Get("mport"); mport != "" {
		var ports []string
		for _, r := range strings.Split(mport, ",") {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			ports = append(ports, strings.ReplaceAll(r, "-", ":"))
		}
		if len(ports) > 0 {
			h.ServerPorts = ports
		}
	}
}
