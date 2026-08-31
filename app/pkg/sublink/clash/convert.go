package clash

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

// server fills the shared server/port pair every outbound needs.
func (p proxy) server() (option.ServerOptions, error) {
	host := p.str("server")
	if host == "" {
		return option.ServerOptions{}, fmt.Errorf("missing server")
	}
	port, err := p.port("port")
	if err != nil {
		return option.ServerOptions{}, err
	}
	return option.ServerOptions{Server: host, ServerPort: port}, nil
}

// tlsOptions builds the outbound TLS block from Clash's flat TLS keys.
//
// `force` is set for protocols that are TLS-framed by definition (trojan,
// hysteria2, anytls), where Clash omits the `tls: true` flag because there is
// nothing to toggle. For vmess/vless the flag is meaningful and honoured.
func (p proxy) tlsOptions(force bool) *option.OutboundTLSOptions {
	enabled := force || p.boolean("tls")
	if !enabled {
		return nil
	}
	tls := &option.OutboundTLSOptions{Enabled: true}
	if sni := p.str("servername", "sni", "peer"); sni != "" {
		tls.ServerName = sni
	}
	if p.boolean("skip-cert-verify", "insecure", "allowInsecure") {
		tls.Insecure = true
	}
	if alpn := p.strList("alpn"); len(alpn) > 0 {
		tls.ALPN = alpn
	}
	if fp := p.str("client-fingerprint", "fingerprint"); fp != "" {
		tls.UTLS = &option.OutboundUTLSOptions{Enabled: true, Fingerprint: fp}
	}
	if reality := p.sub("reality-opts"); len(reality) > 0 {
		tls.Reality = &option.OutboundRealityOptions{
			Enabled:   true,
			PublicKey: reality.str("public-key"),
			ShortID:   reality.str("short-id"),
		}
	}
	return tls
}

// transport maps Clash's `network` + `*-opts` onto a V2Ray transport block.
// "tcp"/"raw"/"" mean no transport block at all, which sing-box expects as a
// nil pointer rather than an empty object.
func (p proxy) transport() *option.V2RayTransportOptions {
	switch strings.ToLower(p.str("network")) {
	case "ws", "websocket":
		opts := p.sub("ws-opts")
		ws := option.V2RayWebsocketOptions{Path: opts.str("path")}
		if host := opts.headerHost(); host != "" {
			ws.Headers = map[string]badoption.Listable[string]{"Host": {host}}
		}
		if ed, ok := opts.number("max-early-data"); ok && ed > 0 {
			ws.MaxEarlyData = uint32(ed)
			ws.EarlyDataHeaderName = opts.str("early-data-header-name")
			if ws.EarlyDataHeaderName == "" {
				ws.EarlyDataHeaderName = "Sec-WebSocket-Protocol"
			}
		}
		return &option.V2RayTransportOptions{Type: "ws", WebsocketOptions: ws}

	case "grpc":
		opts := p.sub("grpc-opts")
		return &option.V2RayTransportOptions{
			Type:        "grpc",
			GRPCOptions: option.V2RayGRPCOptions{ServiceName: opts.str("grpc-service-name", "serviceName")},
		}

	case "h2", "http2", "http":
		opts := p.sub("h2-opts")
		if len(opts) == 0 {
			opts = p.sub("http-opts")
		}
		httpOpts := option.V2RayHTTPOptions{Path: opts.str("path")}
		if hosts := opts.strList("host"); len(hosts) > 0 {
			httpOpts.Host = hosts
		}
		return &option.V2RayTransportOptions{Type: "http", HTTPOptions: httpOpts}
	}
	return nil
}

// seconds reads an integer-seconds field into sing-box's duration type.
func (p proxy) seconds(keys ...string) badoption.Duration {
	if n, ok := p.number(keys...); ok && n > 0 {
		return badoption.Duration(time.Duration(n) * time.Second)
	}
	return 0
}

// mbps parses Clash's bandwidth fields, which are written either as a bare
// number (`up: 50`) or with a unit (`up: "50 Mbps"`).
func (p proxy) mbps(keys ...string) int {
	if n, ok := p.number(keys...); ok {
		return n
	}
	raw := p.str(keys...)
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return n
}

func (p proxy) toShadowsocks() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	method := p.str("cipher", "method")
	if method == "" {
		return nil, fmt.Errorf("missing cipher")
	}
	opts := option.ShadowsocksOutboundOptions{
		ServerOptions: srv,
		Method:        method,
		Password:      p.str("password"),
	}
	// simple-obfs / v2ray-plugin: Clash nests the settings, sing-box wants the
	// plugin's own "k=v;k=v" string.
	switch plugin := p.str("plugin"); plugin {
	case "obfs", "simple-obfs":
		po := p.sub("plugin-opts")
		opts.Plugin = "obfs-local"
		opts.PluginOptions = joinPluginOpts(map[string]string{
			"obfs":      po.str("mode"),
			"obfs-host": po.str("host"),
		})
	case "v2ray-plugin":
		po := p.sub("plugin-opts")
		opts.Plugin = "v2ray-plugin"
		fields := map[string]string{
			"mode": po.str("mode"),
			"host": po.str("host"),
			"path": po.str("path"),
		}
		if po.boolean("tls") {
			fields["tls"] = "true"
		}
		opts.PluginOptions = joinPluginOpts(fields)
	}
	return &node.SubNode{Type: "shadowsocks", Tag: p.str("name"), Options: opts}, nil
}

func joinPluginOpts(fields map[string]string) string {
	// Stable order: sing-box does not care, but a stable string keeps the
	// subscription diff from reporting a change on every refresh.
	var parts []string
	for _, k := range []string{"mode", "obfs", "obfs-host", "host", "path", "tls"} {
		if v := fields[k]; v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, ";")
}

func (p proxy) toVMess() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	uuid := p.str("uuid")
	if uuid == "" {
		return nil, fmt.Errorf("missing uuid")
	}
	security := p.str("cipher")
	if security == "" {
		security = "auto"
	}
	alterID, _ := p.number("alterId", "alterid")
	opts := option.VMessOutboundOptions{
		ServerOptions:  srv,
		UUID:           uuid,
		Security:       security,
		AlterId:        alterID,
		Transport:      p.transport(),
		PacketEncoding: p.str("packet-encoding"),
	}
	opts.TLS = p.tlsOptions(false)
	return &node.SubNode{Type: "vmess", Tag: p.str("name"), Options: opts}, nil
}

func (p proxy) toTrojan() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	password := p.str("password")
	if password == "" {
		return nil, fmt.Errorf("missing password")
	}
	opts := option.TrojanOutboundOptions{
		ServerOptions: srv,
		Password:      password,
		Transport:     p.transport(),
	}
	opts.TLS = p.tlsOptions(true)
	return &node.SubNode{Type: "trojan", Tag: p.str("name"), Options: opts}, nil
}

func (p proxy) toVLESS() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	uuid := p.str("uuid")
	if uuid == "" {
		return nil, fmt.Errorf("missing uuid")
	}
	opts := option.VLESSOutboundOptions{
		ServerOptions: srv,
		UUID:          uuid,
		Flow:          p.str("flow"),
		Transport:     p.transport(),
	}
	if pe := p.str("packet-encoding"); pe != "" {
		opts.PacketEncoding = &pe
	}
	// Reality implies TLS even when the feed omits `tls: true`.
	opts.TLS = p.tlsOptions(len(p.sub("reality-opts")) > 0)
	return &node.SubNode{Type: "vless", Tag: p.str("name"), Options: opts}, nil
}

func (p proxy) toHysteria2() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	opts := option.Hysteria2OutboundOptions{
		ServerOptions: srv,
		Password:      p.str("password", "auth"),
		UpMbps:        p.mbps("up"),
		DownMbps:      p.mbps("down"),
	}
	// Port hopping: Clash writes "30000-31000", sing-box wants "30000:31000".
	if ports := p.strList("ports"); len(ports) > 0 {
		converted := make([]string, 0, len(ports))
		for _, r := range ports {
			converted = append(converted, strings.ReplaceAll(r, "-", ":"))
		}
		opts.ServerPorts = converted
	}
	if obfs := p.str("obfs"); obfs != "" {
		opts.Obfs = &option.Hysteria2Obfs{
			Type:     obfs,
			Password: p.str("obfs-password"),
		}
	}
	opts.TLS = p.tlsOptions(true)
	return &node.SubNode{Type: "hysteria2", Tag: p.str("name"), Options: opts}, nil
}

func (p proxy) toAnyTLS() (*node.SubNode, error) {
	srv, err := p.server()
	if err != nil {
		return nil, err
	}
	password := p.str("password")
	if password == "" {
		return nil, fmt.Errorf("missing password")
	}
	opts := option.AnyTLSOutboundOptions{
		ServerOptions:            srv,
		Password:                 password,
		IdleSessionCheckInterval: p.seconds("idle-session-check-interval"),
		IdleSessionTimeout:       p.seconds("idle-session-timeout"),
	}
	if n, ok := p.number("min-idle-session"); ok {
		opts.MinIdleSession = n
	}
	opts.TLS = p.tlsOptions(true)
	return &node.SubNode{Type: "anytls", Tag: p.str("name"), Options: opts}, nil
}
