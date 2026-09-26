package config

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"
)

// TestCreateContext_AllRegistriesPresent 守住 CreateContext 的完整性。
//
// option.Options 里每一个多态字段 (inbounds/outbounds/endpoints/dns.servers/
// services) 反序列化时都会去 context 取自己的 registry, 取不到就直接报
// "missing <x> fields registry in context"。少注册任何一个, 对应字段一出现
// 配置就解析不了 —— 这正是 endpoints 之前踩的坑。
func TestCreateContext_AllRegistriesPresent(t *testing.T) {
	ctx := CreateContext(context.Background())

	if service.FromContext[option.DNSTransportOptionsRegistry](ctx) == nil {
		t.Error("DNSTransportOptionsRegistry 未注册")
	}
	if service.FromContext[option.InboundOptionsRegistry](ctx) == nil {
		t.Error("InboundOptionsRegistry 未注册")
	}
	if service.FromContext[option.OutboundOptionsRegistry](ctx) == nil {
		t.Error("OutboundOptionsRegistry 未注册")
	}
	if service.FromContext[option.EndpointOptionsRegistry](ctx) == nil {
		t.Error("EndpointOptionsRegistry 未注册")
	}
	if service.FromContext[option.ServiceOptionsRegistry](ctx) == nil {
		t.Error("ServiceOptionsRegistry 未注册")
	}
}

// TestUnmarshal_WireGuardEndpoint 是这次 bug 的回归用例。
// 配置取自真实落地机 (WireGuard endpoint 主动拨入中转机)。
// 修复前这里会失败: "missing endpoint fields registry in context"。
func TestUnmarshal_WireGuardEndpoint(t *testing.T) {
	const raw = `{
  "log": { "level": "info", "timestamp": true },
  "dns": {
    "servers": [{ "type": "udp", "tag": "dns-local", "server": "1.1.1.1" }],
    "final": "dns-local",
    "strategy": "prefer_ipv4"
  },
  "endpoints": [
    {
      "type": "wireguard",
      "tag": "wg-us-transit",
      "system": false,
      "mtu": 1408,
      "address": ["10.20.0.2/32"],
      "private_key": "aF4KOgrNBt93zcP5nyGLu3jXMgTcC3hnQPZVAy5ln2o=",
      "peers": [
        {
          "address": "199.180.117.87",
          "port": 51820,
          "public_key": "5D501kIjcRks6L8exmVB3+KX0kb/3fPjnfp3ZNpWMFE=",
          "allowed_ips": ["0.0.0.0/0", "::/0"],
          "persistent_keepalive_interval": 25
        }
      ]
    }
  ],
  "outbounds": [{ "type": "direct", "tag": "direct" }],
  "route": { "final": "direct" }
}`

	ctx := CreateContext(context.Background())
	var cfg option.Options
	if err := json.UnmarshalContext(ctx, []byte(raw), &cfg); err != nil {
		t.Fatalf("解析含 endpoints 的配置失败: %v", err)
	}

	if len(cfg.Endpoints) != 1 {
		t.Fatalf("Endpoints 数量 = %d, want 1", len(cfg.Endpoints))
	}
	ep := cfg.Endpoints[0]
	if ep.Type != "wireguard" {
		t.Errorf("Endpoint.Type = %q, want %q", ep.Type, "wireguard")
	}
	if ep.Tag != "wg-us-transit" {
		t.Errorf("Endpoint.Tag = %q, want %q", ep.Tag, "wg-us-transit")
	}

	// Options 必须是具体类型, 不是 map —— 说明 registry 真的被用上了。
	wg, ok := ep.Options.(*option.WireGuardEndpointOptions)
	if !ok {
		t.Fatalf("Endpoint.Options 类型 = %T, want *option.WireGuardEndpointOptions", ep.Options)
	}
	if len(wg.Peers) != 1 {
		t.Fatalf("Peers 数量 = %d, want 1", len(wg.Peers))
	}
	if got := wg.Peers[0].PersistentKeepaliveInterval; got != 25 {
		t.Errorf("PersistentKeepaliveInterval = %d, want 25", got)
	}
}

// TestCreateEndpointOptions 覆盖 endpoint 类型表本身。
func TestCreateEndpointOptions(t *testing.T) {
	r := &Registry{}

	if _, ok := r.CreateEndpointOptions("wireguard"); !ok {
		t.Error("wireguard endpoint 应被支持")
	}
	if _, ok := r.CreateEndpointOptions("tailscale"); !ok {
		t.Error("tailscale endpoint 应被支持")
	}
	if _, ok := r.CreateEndpointOptions("nonexistent"); ok {
		t.Error("未知 endpoint 类型不应返回 ok")
	}

	// "wireguard" 在 outbound 表和 endpoint 表里是两个不同的结构体,
	// 前者是已废弃的 legacy 写法, 别串了。
	obOpts, _ := r.CreateOutboundOptions("wireguard")
	if _, isLegacy := obOpts.(*option.LegacyWireGuardOutboundOptions); !isLegacy {
		t.Errorf("outbound wireguard 类型 = %T, want *option.LegacyWireGuardOutboundOptions", obOpts)
	}
	epOpts, _ := r.CreateEndpointOptions("wireguard")
	if _, isEndpoint := epOpts.(*option.WireGuardEndpointOptions); !isEndpoint {
		t.Errorf("endpoint wireguard 类型 = %T, want *option.WireGuardEndpointOptions", epOpts)
	}
}

// TestCreateServiceOptions 覆盖 services 字段 (同一类 bug 的第二处)。
func TestCreateServiceOptions(t *testing.T) {
	r := &Registry{}
	for _, typ := range []string{"resolved", "ssm-api", "derp"} {
		if _, ok := r.CreateServiceOptions(typ); !ok {
			t.Errorf("service 类型 %q 应被支持", typ)
		}
	}
	if _, ok := r.CreateServiceOptions("nonexistent"); ok {
		t.Error("未知 service 类型不应返回 ok")
	}
}

// TestCreateOutboundOptions_AnyTLS 补上之前漏掉的 anytls。
func TestCreateOutboundOptions_AnyTLS(t *testing.T) {
	r := &Registry{}
	if _, ok := r.CreateOutboundOptions("anytls"); !ok {
		t.Error("anytls outbound 应被支持")
	}
	if _, ok := r.CreateInboundOptions("anytls"); !ok {
		t.Error("anytls inbound 应被支持")
	}
}
