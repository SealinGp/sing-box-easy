package diagnostics

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"net/netip"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dns"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/route"
)

// RouteProbeRequest asks where a destination would be routed.
type RouteProbeRequest struct {
	// Destination is a domain or an IP address.
	Destination string `json:"destination"`
	Port        uint16 `json:"port"`
	Network     string `json:"network"`
	// Inbound is the tag traffic arrives on. Often decisive — the first rule
	// of a typical config keys on it — so the UI offers the configured tags.
	Inbound string `json:"inbound"`
	// SourceIP is the client address, for source_ip_cidr rules.
	SourceIP string `json:"source_ip"`
	// Protocol is the application protocol sing-box would sniff ("tls",
	// "http", "dns", …). Supplying it decides `protocol` rules, which are
	// otherwise undecidable because sniffing needs bytes on the wire.
	Protocol string `json:"protocol"`
}

// ProbeRoute predicts the outbound a destination would leave through.
//
// POST /route/probe
func (h *Service) ProbeRoute(ctx context.Context, req RouteProbeRequest) (any, error) {

	cfg, err := h.configManager.GetConfigSubset("route", "outbounds", "endpoints")
	if err != nil {
		return nil, fault.New(fault.Configuration, err.Error())
	}

	options := routeprobe.Options{
		Destination: req.Destination,
		Port:        req.Port,
		Network:     req.Network,
		Inbound:     req.Inbound,
		SourceIP:    req.SourceIP,
		Protocol:    req.Protocol,
	}

	// Both extras come from the running instance and both are optional: a
	// stopped sing-box degrades the prediction rather than failing it, which
	// is the case the offline walk exists for in the first place.
	clash, clashErr := h.readClashAPISettings()
	if client, clientErr := dnsprobe.NewClashClientFromValues(clash.ExternalController, clash.Secret); clashErr == nil && clientErr == nil {
		options.Resolve = clashResolver(client)
		if mode, modeErr := client.Mode(); modeErr == nil {
			options.ClashMode = mode
		}
	}

	result, runErr := routeprobe.Run(&cfg.Options, options)
	if runErr != nil {
		// Only invalid input reaches here; everything else degrades into the
		// result, where the user can see it.
		return nil, fault.New(fault.Invalid, runErr.Error())
	}

	return result, nil
}

// clashResolver resolves through sing-box itself rather than through this
// process's resolver.
//
// The distinction matters: address-based rules are matched against whatever
// sing-box resolved, and a panel that asked its own resolver could predict a
// different rule purely because it got a different CDN address back. Asking
// sing-box means a wrong prediction is at least wrong for the same reason the
// real connection would be.
func clashResolver(client *dnsprobe.ClashClient) routeprobe.Resolver {
	return func(domain string) (netip.Addr, error) {
		live, err := client.Query(domain, "A")
		if err != nil {
			return netip.Addr{}, err
		}
		for _, answer := range live.Answers {
			if address, parseErr := netip.ParseAddr(answer.Data); parseErr == nil {
				return address, nil
			}
		}
		return netip.Addr{}, errors.New("no address records returned")
	}
}
