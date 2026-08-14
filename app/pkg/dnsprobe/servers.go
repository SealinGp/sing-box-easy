package dnsprobe

import (
	"net"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
)

// ServerDetail identifies the DNS server a query was routed to.
//
// The tag alone ("dns_router") does not say where the query went, which is the
// question a diagnostic is being asked; the address does.
type ServerDetail struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
	// Address is host:port, empty for server types with no upstream (hosts,
	// fakeip) or when the tag could not be found in the config.
	Address string `json:"address,omitempty"`
	// Detour is the outbound the server is reached through, when set. Worth
	// showing: it is why an upstream may be unreachable from this process.
	Detour string `json:"detour,omitempty"`
	// Found is false when the tag does not match any configured server, which
	// means the config references a server that does not exist.
	Found bool `json:"found"`
}

// defaultPorts per DNS transport, applied when server_port is omitted.
var defaultPorts = map[string]uint16{
	"udp":   53,
	"tcp":   53,
	"tls":   853,
	"quic":  853,
	"https": 443,
	"h3":    443,
}

// remoteOptions returns the shared remote-server fields (address, port,
// dialer) for the transports that have them, or nil for local ones such as
// hosts and fakeip.
//
// The remote types embed one another — https embeds tls embeds the base — so
// unwrapping is enough and each transport does not need its own case.
func remoteOptions(server option.DNSServerOptions) *option.RemoteDNSServerOptions {
	switch options := server.Options.(type) {
	case *option.RemoteDNSServerOptions:
		return options
	case *option.RemoteTLSDNSServerOptions:
		return &options.RemoteDNSServerOptions
	case *option.RemoteHTTPSDNSServerOptions:
		return &options.RemoteDNSServerOptions
	default:
		return nil
	}
}

// serverAddress renders host:port for a remote server, applying the transport's
// default port when none was configured.
func serverAddress(serverType string, options *option.RemoteDNSServerOptions) string {
	if options == nil || options.Server == "" {
		return ""
	}
	port := options.ServerPort
	if port == 0 {
		port = defaultPorts[serverType]
	}
	if port == 0 {
		return options.Server
	}
	return net.JoinHostPort(options.Server, strconv.Itoa(int(port)))
}

// DescribeServer resolves a server tag against the configured servers.
//
// A tag with no matching server yields Found=false rather than a zero-value
// detail, so the UI can say the config points at a server that is not defined
// instead of quietly showing a blank address.
func DescribeServer(servers []option.DNSServerOptions, tag string) *ServerDetail {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}

	for _, server := range servers {
		if server.Tag != tag {
			continue
		}
		options := remoteOptions(server)
		detail := &ServerDetail{
			Tag:     tag,
			Type:    server.Type,
			Address: serverAddress(server.Type, options),
			Found:   true,
		}
		if options != nil {
			detail.Detour = options.Detour
		}
		return detail
	}

	return &ServerDetail{Tag: tag, Found: false}
}

// routeActionServerTag extracts the server tag from a logged action such as
// "route(dns_router)".
//
// sing-box's own decision line is authoritative, so when it is available its
// server wins over the offline reconstruction — which may have predicted a
// different rule entirely.
func routeActionServerTag(action string) string {
	const prefix = "route("
	if !strings.HasPrefix(action, prefix) || !strings.HasSuffix(action, ")") {
		return ""
	}
	return strings.TrimSpace(action[len(prefix) : len(action)-1])
}

// effectiveServerTag reports which server the query actually used, preferring
// sing-box's logged decision over the reconstruction.
func effectiveServerTag(matches []LoggedMatch, attribution Attribution) string {
	if len(matches) > 0 {
		if tag := routeActionServerTag(matches[len(matches)-1].Action); tag != "" {
			return tag
		}
		// A non-route action (predefined, reject) reaches no server at all.
		if matches[len(matches)-1].Action != "" {
			return ""
		}
	}
	return attribution.Server
}
