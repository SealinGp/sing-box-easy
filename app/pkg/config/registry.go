package config

import (
	"context"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

// Registry provides all necessary registries for sing-box config parsing
type Registry struct{}

// CreateOptions implements option.DNSTransportOptionsRegistry
func (r *Registry) CreateDNSOptions(transportType string) (any, bool) {
	switch transportType {
	case "udp":
		return new(option.RemoteDNSServerOptions), true
	case "tcp":
		return new(option.RemoteDNSServerOptions), true
	case "tls":
		return new(option.RemoteTLSDNSServerOptions), true
	case "https":
		return new(option.RemoteHTTPSDNSServerOptions), true
	case "http3":
		return new(option.RemoteHTTPSDNSServerOptions), true
	case "quic":
		return new(option.RemoteDNSServerOptions), true
	case "dhcp":
		return new(option.DHCPDNSServerOptions), true
	case "fakeip":
		return new(option.FakeIPDNSServerOptions), true
	case "local":
		return new(option.LocalDNSServerOptions), true
	case "hosts":
		return new(option.HostsDNSServerOptions), true
	default:
		return nil, false
	}
}

// CreateInboundOptions implements option.InboundOptionsRegistry
func (r *Registry) CreateInboundOptions(inboundType string) (any, bool) {
	switch inboundType {
	case C.TypeTun:
		return new(option.TunInboundOptions), true
	case C.TypeRedirect:
		return new(option.RedirectInboundOptions), true
	case C.TypeTProxy:
		return new(option.TProxyInboundOptions), true
	case C.TypeDirect:
		return new(option.DirectInboundOptions), true
	case C.TypeSOCKS:
		return new(option.SocksInboundOptions), true
	case C.TypeHTTP:
		return new(option.HTTPMixedInboundOptions), true
	case C.TypeMixed:
		return new(option.HTTPMixedInboundOptions), true
	case C.TypeShadowsocks:
		return new(option.ShadowsocksInboundOptions), true
	case C.TypeVMess:
		return new(option.VMessInboundOptions), true
	case C.TypeTrojan:
		return new(option.TrojanInboundOptions), true
	case C.TypeNaive:
		return new(option.NaiveInboundOptions), true
	case C.TypeHysteria:
		return new(option.HysteriaInboundOptions), true
	case C.TypeHysteria2:
		return new(option.Hysteria2InboundOptions), true
	case C.TypeVLESS:
		return new(option.VLESSInboundOptions), true
	case C.TypeTUIC:
		return new(option.TUICInboundOptions), true
	case C.TypeShadowTLS:
		return new(option.ShadowTLSInboundOptions), true
	default:
		return nil, false
	}
}

// CreateOutboundOptions implements option.OutboundOptionsRegistry
func (r *Registry) CreateOutboundOptions(outboundType string) (any, bool) {
	switch outboundType {
	case C.TypeDirect:
		return new(option.DirectOutboundOptions), true
	case C.TypeBlock, C.TypeDNS:
		return new(option.StubOptions), true
	case C.TypeSOCKS:
		return new(option.SOCKSOutboundOptions), true
	case C.TypeHTTP:
		return new(option.HTTPOutboundOptions), true
	case C.TypeShadowsocks:
		return new(option.ShadowsocksOutboundOptions), true
	case C.TypeVMess:
		return new(option.VMessOutboundOptions), true
	case C.TypeTrojan:
		return new(option.TrojanOutboundOptions), true
	case C.TypeWireGuard:
		return new(option.LegacyWireGuardOutboundOptions), true
	case C.TypeHysteria:
		return new(option.HysteriaOutboundOptions), true
	case C.TypeHysteria2:
		return new(option.Hysteria2OutboundOptions), true
	case C.TypeTor:
		return new(option.TorOutboundOptions), true
	case C.TypeSSH:
		return new(option.SSHOutboundOptions), true
	case C.TypeShadowTLS:
		return new(option.ShadowTLSOutboundOptions), true
	case C.TypeShadowsocksR:
		return new(option.ShadowsocksROutboundOptions), true
	case C.TypeVLESS:
		return new(option.VLESSOutboundOptions), true
	case C.TypeTUIC:
		return new(option.TUICOutboundOptions), true
	case C.TypeSelector:
		return new(option.SelectorOutboundOptions), true
	case C.TypeURLTest:
		return new(option.URLTestOutboundOptions), true
	default:
		return nil, false
	}
}

// CreateContext creates a context with all required registries
func CreateContext(ctx context.Context) context.Context {
	registry := &Registry{}
	// Add DNS registry
	ctx = service.ContextWith[option.DNSTransportOptionsRegistry](ctx, &dnsRegistryWrapper{registry})
	// Add Inbound registry
	ctx = service.ContextWith[option.InboundOptionsRegistry](ctx, &inboundRegistryWrapper{registry})
	// Add Outbound registry
	ctx = service.ContextWith[option.OutboundOptionsRegistry](ctx, &outboundRegistryWrapper{registry})
	return ctx
}

// Wrapper types to satisfy different registry interfaces
type dnsRegistryWrapper struct {
	*Registry
}

func (w *dnsRegistryWrapper) CreateOptions(transportType string) (any, bool) {
	return w.Registry.CreateDNSOptions(transportType)
}

type inboundRegistryWrapper struct {
	*Registry
}

func (w *inboundRegistryWrapper) CreateOptions(inboundType string) (any, bool) {
	return w.Registry.CreateInboundOptions(inboundType)
}

type outboundRegistryWrapper struct {
	*Registry
}

func (w *outboundRegistryWrapper) CreateOptions(outboundType string) (any, bool) {
	return w.Registry.CreateOutboundOptions(outboundType)
}
