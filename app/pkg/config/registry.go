package config

import (
	"context"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

// Registry provides all necessary registries for sing-box config parsing
type Registry struct{}

// 编译期断言: 五个 wrapper 必须分别满足对应的 registry 接口。
// 上游若变更接口签名, 这里会直接编译失败, 而不是等到运行时才报
// "missing ... fields registry in context"。
var (
	_ option.DNSTransportOptionsRegistry = (*dnsRegistryWrapper)(nil)
	_ option.InboundOptionsRegistry      = (*inboundRegistryWrapper)(nil)
	_ option.OutboundOptionsRegistry     = (*outboundRegistryWrapper)(nil)
	_ option.EndpointOptionsRegistry     = (*endpointRegistryWrapper)(nil)
	_ option.ServiceOptionsRegistry      = (*serviceRegistryWrapper)(nil)
)

// CreateOptions implements option.DNSTransportOptionsRegistry
// 类型字符串一律用 C.DNSType* 常量, 不要写字面量。
//
// 这里原本是 `case "http3"`, 但 sing-box 的常量是 C.DNSTypeHTTP3 = "h3"
// (constant/dns.go), 传输层也注册在 "h3" 下 (dns/transport/quic/http3.go)。
// 两边不一致造成双向损坏:
//
//	type: "h3"    -> 这里 default 分支返回 false, 整个 config.json 解析失败,
//	                 报 "unknown transport type: h3" —— 面板连配置都打不开;
//	type: "http3" -> 这里能解析, 但 sing-box 运行时没有 "http3" 传输,
//	                 写出来的配置根本起不来。
//
// 同一个后端内部也互相矛盾: dnsprobe/servers.go 的默认端口表用的就是 "h3"。
func (r *Registry) CreateDNSOptions(transportType string) (any, bool) {
	switch transportType {
	case C.DNSTypeUDP:
		return new(option.RemoteDNSServerOptions), true
	case C.DNSTypeTCP:
		return new(option.RemoteDNSServerOptions), true
	case C.DNSTypeTLS:
		return new(option.RemoteTLSDNSServerOptions), true
	case C.DNSTypeHTTPS:
		return new(option.RemoteHTTPSDNSServerOptions), true
	case C.DNSTypeHTTP3:
		return new(option.RemoteHTTPSDNSServerOptions), true
	case C.DNSTypeQUIC:
		return new(option.RemoteDNSServerOptions), true
	case C.DNSTypeDHCP:
		return new(option.DHCPDNSServerOptions), true
	case C.DNSTypeFakeIP:
		return new(option.FakeIPDNSServerOptions), true
	case C.DNSTypeLocal:
		return new(option.LocalDNSServerOptions), true
	case C.DNSTypeHosts:
		return new(option.HostsDNSServerOptions), true
	case C.DNSTypeTailscale:
		// 选项结构体在 option 包里始终存在, 传输实现则由 with_tailscale
		// 构建标签决定。面板只负责解析 —— 能不能跑是 sing-box 二进制的事,
		// 不该因此连配置都读不出来。
		return new(option.TailscaleDNSServerOptions), true
	default:
		return nil, false
	}
}

// CreateEndpointOptions implements option.EndpointOptionsRegistry
//
// Endpoint 是 sing-box 1.11 引入的类型: 同时具备 inbound 与 outbound 行为。
// 注意 "wireguard" 同时存在于 outbound 表和 endpoint 表, 两者互不相干 ——
// outbound 侧是已废弃的 LegacyWireGuardOutboundOptions, endpoint 侧才是新式写法。
func (r *Registry) CreateEndpointOptions(endpointType string) (any, bool) {
	switch endpointType {
	case C.TypeWireGuard:
		return new(option.WireGuardEndpointOptions), true
	case C.TypeTailscale:
		return new(option.TailscaleEndpointOptions), true
	default:
		return nil, false
	}
}

// CreateDNSRuleActionOptions 按 `action` 返回 DNS 规则动作的选项结构。
//
// 与其他 Create* 不同, 这个方法不实现任何 sing-box registry 接口 ——
// DNSRuleAction 自己在 UnmarshalJSONContext 里硬编码了这四个分支, 不走 context
// 里的 registry。这里存在的唯一理由是给 schema 生成器一个可枚举的入口:
// Go 的 type switch 无法在运行时遍历, 所以“有哪些 action”必须另有一份列表,
// 而那份列表必须能被测试对着真实结构体校验。见 DNSRuleActionTypes。
//
// 四个分支与 option/rule_action.go 的 _DNSRuleAction 一一对应。route-rule 专用的
// direct / hijack-dns / sniff / resolve 不在此处, 见 DNSRuleActionTypes 的注释。
func (r *Registry) CreateDNSRuleActionOptions(action string) (any, bool) {
	switch action {
	case C.RuleActionTypeRoute:
		return new(option.DNSRouteActionOptions), true
	case C.RuleActionTypeRouteOptions:
		return new(option.DNSRouteOptionsActionOptions), true
	case C.RuleActionTypeReject:
		return new(option.RejectActionOptions), true
	case C.RuleActionTypePredefined:
		return new(option.DNSRouteActionPredefined), true
	default:
		return nil, false
	}
}

// CreateRouteRuleActionOptions 按 `action` 返回路由规则动作的选项结构。
//
// 与 CreateDNSRuleActionOptions 同理: 不实现任何 sing-box registry 接口,
// 存在的唯一理由是给 schema 生成器一个可枚举的入口。
//
// 七个分支与 option/rule_action.go 的 _RuleAction 一一对应。注意这与 DNS 规则的
// 动作集合并不相同, 两个方向都有差异, 见 RouteRuleActionTypes 的注释。
//
// hijack-dns 没有任何选项结构 —— RuleAction.MarshalJSON 对它取 v = nil ——
// 所以这里返回 nil 且 ok=true, 并由 domain.FieldlessTypes 允许零字段。
func (r *Registry) CreateRouteRuleActionOptions(action string) (any, bool) {
	switch action {
	case C.RuleActionTypeRoute:
		return new(option.RouteActionOptions), true
	case C.RuleActionTypeRouteOptions:
		return new(option.RouteOptionsActionOptions), true
	case C.RuleActionTypeDirect:
		return new(option.DirectActionOptions), true
	case C.RuleActionTypeReject:
		return new(option.RejectActionOptions), true
	case C.RuleActionTypeHijackDNS:
		// 行为而非配置: 没有可编辑字段。
		return new(struct{}), true
	case C.RuleActionTypeSniff:
		return new(option.RouteActionSniff), true
	case C.RuleActionTypeResolve:
		return new(option.RouteActionResolve), true
	default:
		return nil, false
	}
}

// CreateRouteRuleMatcherOptions 返回路由规则的匹配条件结构。
//
// 路由规则的匹配条件不是多态的: option.RawDefaultRule 是一个约 37 个字段的扁平
// 结构, 没有任何判别字段。这里保持 Create(type) 的签名只是为了复用生成器的形状。
func (r *Registry) CreateRouteRuleMatcherOptions(matcherType string) (any, bool) {
	switch matcherType {
	case C.RuleTypeDefault:
		return new(option.RawDefaultRule), true
	default:
		return nil, false
	}
}

// CreateServiceOptions implements option.ServiceOptionsRegistry
func (r *Registry) CreateServiceOptions(serviceType string) (any, bool) {
	switch serviceType {
	case C.TypeResolved:
		return new(option.ResolvedServiceOptions), true
	case C.TypeSSMAPI:
		return new(option.SSMAPIServiceOptions), true
	case C.TypeDERP:
		return new(option.DERPServiceOptions), true
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
	case C.TypeAnyTLS:
		return new(option.AnyTLSInboundOptions), true
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
	case C.TypeAnyTLS:
		return new(option.AnyTLSOutboundOptions), true
	default:
		return nil, false
	}
}

// CreateContext creates a context with all required registries.
//
// 必须覆盖 option.Options 里所有需要 registry 的字段, 否则该字段一出现就会报
// "missing <x> fields registry in context"。对照 sing-box include.Context():
// inbound / outbound / endpoint / dns transport / service 共 5 个。
func CreateContext(ctx context.Context) context.Context {
	registry := &Registry{}
	// Add DNS registry
	ctx = service.ContextWith[option.DNSTransportOptionsRegistry](ctx, &dnsRegistryWrapper{registry})
	// Add Inbound registry
	ctx = service.ContextWith[option.InboundOptionsRegistry](ctx, &inboundRegistryWrapper{registry})
	// Add Outbound registry
	ctx = service.ContextWith[option.OutboundOptionsRegistry](ctx, &outboundRegistryWrapper{registry})
	// Add Endpoint registry (wireguard / tailscale)
	ctx = service.ContextWith[option.EndpointOptionsRegistry](ctx, &endpointRegistryWrapper{registry})
	// Add Service registry (resolved / ssm-api / derp)
	ctx = service.ContextWith[option.ServiceOptionsRegistry](ctx, &serviceRegistryWrapper{registry})
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

type endpointRegistryWrapper struct {
	*Registry
}

func (w *endpointRegistryWrapper) CreateOptions(endpointType string) (any, bool) {
	return w.Registry.CreateEndpointOptions(endpointType)
}

type serviceRegistryWrapper struct {
	*Registry
}

func (w *serviceRegistryWrapper) CreateOptions(serviceType string) (any, bool) {
	return w.Registry.CreateServiceOptions(serviceType)
}
