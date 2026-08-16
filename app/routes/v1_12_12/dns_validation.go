package v1_13_0

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

// bindDNSServer parses a DNS server from the request body.
//
// It uses sing-box's context-aware JSON rather than c.Bind, and that is the
// whole point of this helper. option.DNSServerOptions is polymorphic:
//
//	type _DNSServerOptions struct {
//	    Type    string `json:"type,omitempty"`
//	    Tag     string `json:"tag,omitempty"`
//	    Options any    `json:"-"`
//	}
//
// Every real field — server, server_port, detour, path, interface,
// inet4_range, predefined — lives behind `Options`, which is populated only by
// UnmarshalJSONContext looking the type up in the transport registry. There is
// no plain UnmarshalJSON on the type at all, so Hertz's reflection binder
// filled in Type and Tag and left Options nil. The config then failed to
// marshal with "expected json object start, but starts with nil", which meant
// adding or editing a DNS server through the panel could not succeed — the
// request always came back as an internal error and nothing was saved.
//
// AddDNSRule/UpdateDNSRule already carry the same fix and the same reasoning;
// the server handlers were missed.
//
// Returns a non-empty message on failure rather than an error, so callers map
// it straight onto the response envelope.
func bindDNSServer(ctx context.Context, c *app.RequestContext) (option.DNSServerOptions, string) {
	var server option.DNSServerOptions

	body, err := c.Body()
	if err != nil {
		return server, "failed to read request body: " + err.Error()
	}

	if err := json.UnmarshalContext(config.CreateContext(ctx), body, &server); err != nil {
		return server, "invalid DNS server: " + err.Error()
	}

	return server, ""
}

// validateDNSServer rejects the shapes sing-box would reject on start, plus the
// ones that produce a server which resolves nothing.
//
// There was no validation here at all before — only a bare "tag is required"
// inline in the handler — so a server with an unknown type or a remote server
// with no address was written to config.json and only surfaced when
// `sing-box check` failed later in the save, with a message pointing at the
// file rather than the field.
func validateDNSServer(server option.DNSServerOptions) error {
	if strings.TrimSpace(server.Tag) == "" {
		return fmt.Errorf("tag is required")
	}

	if strings.TrimSpace(server.Type) == "" {
		// An absent type means the deprecated legacy transport, which
		// sing-box auto-upgrades on read. Nothing should be creating one now.
		return fmt.Errorf("type is required")
	}

	if !config.IsKnownDNSType(server.Type) {
		return fmt.Errorf("unknown DNS server type %q", server.Type)
	}

	// Remote transports need somewhere to send the query. The address lives on
	// the embedded DNSServerAddressOptions, so this covers udp/tcp/quic and,
	// through embedding, tls/https/h3 as well.
	switch options := server.Options.(type) {
	case *option.RemoteDNSServerOptions:
		if strings.TrimSpace(options.Server) == "" {
			return fmt.Errorf("server is required for type %q", server.Type)
		}
	case *option.RemoteTLSDNSServerOptions:
		if strings.TrimSpace(options.Server) == "" {
			return fmt.Errorf("server is required for type %q", server.Type)
		}
	case *option.RemoteHTTPSDNSServerOptions:
		if strings.TrimSpace(options.Server) == "" {
			return fmt.Errorf("server is required for type %q", server.Type)
		}
	case *option.DHCPDNSServerOptions:
		// `interface` is genuinely optional — sing-box auto-detects when empty.
	case *option.HostsDNSServerOptions:
		if len(options.Path) == 0 && options.Predefined == nil {
			return fmt.Errorf("hosts server needs a path or predefined entries")
		}
	}

	return nil
}
