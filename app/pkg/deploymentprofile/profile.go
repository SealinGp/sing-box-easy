// Package deploymentprofile builds complete, reviewable starter configurations.
package deploymentprofile

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
	singjson "github.com/sagernet/sing/common/json"
)

type Request struct {
	Role         string `json:"role"`
	Port         uint16 `json:"port"`
	Listen       string `json:"listen"`
	Subscription string `json:"subscription"`
	LANAddress   string `json:"lan_address"`
}
type Result struct {
	Config    map[string]any `json:"config"`
	NodeCount int            `json:"node_count"`
	Port      uint16         `json:"port,omitempty"`
}

// Generate does not write configuration or change any running service.
func Generate(ctx context.Context, request Request) (Result, error) {
	result := Result{Config: map[string]any{"log": map[string]any{"level": "info"}}}
	direct := map[string]any{"type": "direct", "tag": "direct"}
	route := map[string]any{"auto_detect_interface": true, "default_domain_resolver": "google", "final": "direct"}
	google := map[string]any{"type": "tls", "tag": "google", "server": "8.8.8.8"}
	dns := map[string]any{"servers": []any{google}, "final": "google"}
	result.Config["dns"] = dns
	result.Config["route"] = route
	result.Config["outbounds"] = []any{direct}
	switch request.Role {
	case "exit":
		listen := request.Listen
		if listen == "" {
			listen = "::"
		}
		if _, err := netip.ParseAddr(listen); err != nil {
			return Result{}, fmt.Errorf("listen must be an IP address")
		}
		port := request.Port
		if port == 0 {
			var err error
			port, err = AvailablePort()
			if err != nil {
				return Result{}, err
			}
		}
		key := make([]byte, 16)
		if _, err := rand.Read(key); err != nil {
			return Result{}, err
		}
		result.Port = port
		result.Config["inbounds"] = []any{map[string]any{"type": "shadowsocks", "tag": "ss-in", "listen": listen, "listen_port": port, "method": "2022-blake3-aes-128-gcm", "password": base64.StdEncoding.EncodeToString(key)}}
	case "client":
		if strings.TrimSpace(request.Subscription) == "" {
			return Result{}, fmt.Errorf("subscription URL or node link is required")
		}
		nodes, err := subscription.PreviewImport(ctx, request.Subscription)
		if err != nil {
			return Result{}, fmt.Errorf("read subscription: %w", err)
		}
		if len(nodes) == 0 {
			return Result{}, fmt.Errorf("subscription contains no usable proxy nodes")
		}
		outbounds := []any{direct}
		tags := []string{}
		for index, node := range nodes {
			// Names are display labels, not trusted unique identifiers.
			node.Tag = fmt.Sprintf("node-%d-%s", index+1, node.Tag)
			raw, err := singjson.MarshalContext(ctx, &node)
			if err != nil {
				return Result{}, err
			}
			var object map[string]any
			if err = json.Unmarshal(raw, &object); err != nil {
				return Result{}, err
			}
			switch node.Type {
			case "direct", "block", "dns", "selector", "urltest":
				return Result{}, fmt.Errorf("starter profiles require proxy nodes, not %s", node.Type)
			}
			if object["detour"] != nil || object["domain_resolver"] != nil {
				return Result{}, fmt.Errorf("starter profiles require standalone proxy nodes without detour or resolver references")
			}
			outbounds = append(outbounds, object)
			tags = append(tags, node.Tag)
		}
		outbounds = append(outbounds, map[string]any{"type": "urltest", "tag": "proxy", "outbounds": tags, "url": "https://www.gstatic.com/generate_204", "interval": "3m"})
		result.NodeCount = len(nodes)
		result.Config["outbounds"] = outbounds
		route["final"] = "proxy"
		route["default_domain_resolver"] = "bootstrap"
		route["rules"] = []any{map[string]any{"port": 53, "action": "hijack-dns"}, map[string]any{"ip_is_private": true, "action": "route", "outbound": "direct"}}
		google["detour"] = "proxy"
		dns["servers"] = []any{map[string]any{"type": "local", "tag": "bootstrap"}, google}
		inbounds := []any{map[string]any{"type": "tun", "tag": "tun-in", "interface_name": "tun0", "address": []string{"172.16.250.1/30", "fdfe:dcba:9876::1/126"}, "auto_route": true, "strict_route": true, "stack": "system"}}
		if request.LANAddress != "" {
			address, err := netip.ParseAddr(request.LANAddress)
			if err != nil || !address.IsPrivate() {
				return Result{}, fmt.Errorf("LAN DNS must bind to a private LAN IP address")
			}
			inbounds = append(inbounds, map[string]any{"type": "direct", "tag": "lan-dns", "listen": address.String(), "listen_port": 53})
		}
		result.Config["inbounds"] = inbounds
	default:
		return Result{}, fmt.Errorf("role must be exit or client")
	}
	return result, nil
}

// AvailablePort checks TCP and UDP because the default proxy accepts both.
func AvailablePort() (uint16, error) {
	for attempt := 0; attempt < 100; attempt++ {
		tcp, err := net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
		port := tcp.Addr().(*net.TCPAddr).Port
		udp, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
		tcp.Close()
		if err == nil {
			udp.Close()
			return uint16(port), nil
		}
	}
	return 0, fmt.Errorf("could not find a free TCP/UDP port")
}
