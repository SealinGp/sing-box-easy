// Package clientprofile converts server listeners to client nodes, never copying
// server-only fields or treating a user's display name as authentication.
package clientprofile

import (
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
	"strings"
)

type Object = map[string]any
type Request struct {
	Inbound Object `json:"inbound"`
	Server  string `json:"server"`
	User    int    `json:"user"`
}
type Result struct {
	Node     Object   `json:"node"`
	Warnings []string `json:"warnings"`
}

func str(o Object, k string) string { v, _ := o[k].(string); return v }
func object(v any) Object           { o, _ := v.(map[string]any); return o }
func copyFields(dst, src Object, keys ...string) {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			dst[k] = v
		}
	}
}

func Generate(r Request) (Result, error) {
	in := r.Inbound
	kind := str(in, "type")
	server := strings.Trim(strings.TrimSpace(r.Server), "[]")
	if server == "" || strings.ContainsAny(server, "/ \t\r\n") || server == "0.0.0.0" || server == "::" {
		return Result{}, fmt.Errorf("clientExport.serverRequired")
	}
	port, ok := in["listen_port"].(float64)
	if !ok || port < 1 || port > 65535 || port != float64(int(port)) {
		return Result{}, fmt.Errorf("clientExport.portRequired")
	}
	user := Object{}
	if users, ok := in["users"].([]any); ok && len(users) > 0 {
		if r.User < 0 || r.User >= len(users) {
			return Result{}, fmt.Errorf("clientExport.userRequired")
		}
		user = object(users[r.User])
	}
	node := Object{"type": kind, "tag": str(in, "tag") + "-out", "server": server, "server_port": port}
	result := Result{Node: node, Warnings: []string{}}
	switch kind {
	case "mixed", "socks", "http":
		if kind == "mixed" {
			node["type"] = "socks"
			if object(in["tls"])["enabled"] == true {
				node["type"] = "http"
			}
		}
		copyFields(node, user, "username", "password")
	case "vmess":
		copyFields(node, user, "uuid")
		node["security"] = "auto"
		if v, ok := user["alterId"]; ok {
			node["alter_id"] = v
		}
	case "vless":
		copyFields(node, user, "uuid", "flow")
	case "trojan", "anytls":
		copyFields(node, user, "password")
	case "tuic":
		copyFields(node, user, "uuid", "password")
		copyFields(node, in, "congestion_control", "zero_rtt_handshake", "heartbeat")
	case "hysteria":
		copyFields(node, user, "auth", "auth_str")
		copyFields(node, in, "obfs")
		// Bandwidth limits are viewed from opposite ends of the connection.
		for _, pair := range [][2]string{{"up", "down"}, {"up_mbps", "down_mbps"}, {"down", "up"}, {"down_mbps", "up_mbps"}} {
			if v, ok := in[pair[1]]; ok {
				node[pair[0]] = v
			}
		}
	case "hysteria2":
		copyFields(node, user, "password")
		copyFields(node, in, "obfs")
	case "shadowsocks":
		if in["managed"] == true || in["destinations"] != nil {
			return Result{}, fmt.Errorf("clientExport.specialShadowsocks")
		}
		copyFields(node, in, "method", "password")
		if password := str(user, "password"); password != "" {
			node["password"] = password
			if strings.HasPrefix(str(in, "method"), "2022-") && str(in, "password") != "" {
				node["password"] = str(in, "password") + ":" + password
			}
		}
	case "naive":
		return Result{}, fmt.Errorf("clientExport.naive")
	case "shadowtls":
		return Result{}, fmt.Errorf("clientExport.shadowtls")
	default:
		return Result{}, fmt.Errorf("clientExport.unsupported")
	}
	required := map[string][]string{"vmess": {"uuid"}, "vless": {"uuid"}, "trojan": {"password"}, "anytls": {"password"}, "tuic": {"uuid", "password"}, "hysteria2": {"password"}, "shadowsocks": {"method", "password"}}
	for _, key := range required[kind] {
		if str(node, key) == "" && !(kind == "shadowsocks" && str(node, "method") == "none" && key == "password") {
			return Result{}, fmt.Errorf("clientExport.credentialsRequired")
		}
	}
	if kind == "hysteria" && str(node, "auth") == "" && str(node, "auth_str") == "" {
		return Result{}, fmt.Errorf("clientExport.credentialsRequired")
	}
	if node["type"] != "socks" {
		tls, warnings, err := clientTLS(object(in["tls"]), server)
		if err != nil {
			return Result{}, err
		}
		if tls != nil {
			node["tls"] = tls
		}
		result.Warnings = append(result.Warnings, warnings...)
	}
	// Only these protocols have V2Ray transports. Keep their full nested options.
	switch kind {
	case "vmess", "vless", "trojan":
		copyFields(node, in, "transport")
	}
	return result, nil
}

func clientTLS(in Object, server string) (Object, []string, error) {
	if in["enabled"] != true {
		return nil, nil, nil
	}
	warnings := []string{}
	if ech := object(in["ech"]); ech["enabled"] == true {
		return nil, nil, fmt.Errorf("clientExport.ech")
	}
	if mode := str(in, "client_authentication"); mode != "" && mode != "no" {
		return nil, nil, fmt.Errorf("clientExport.clientCertificate")
	}
	tls := Object{"enabled": true, "server_name": server}
	copyFields(tls, in, "server_name", "alpn", "min_version", "max_version", "cipher_suites", "certificate")
	if str(tls, "server_name") == "" {
		tls["server_name"] = server
	}
	if reality := object(in["reality"]); reality["enabled"] == true {
		key, err := base64.RawURLEncoding.DecodeString(str(reality, "private_key"))
		if err != nil {
			return nil, nil, fmt.Errorf("clientExport.realityKey")
		}
		private, err := ecdh.X25519().NewPrivateKey(key)
		if err != nil {
			return nil, nil, fmt.Errorf("clientExport.realityKey")
		}
		sid := ""
		switch ids := reality["short_id"].(type) {
		case string:
			sid = ids
		case []any:
			if len(ids) > 0 {
				sid, _ = ids[0].(string)
			}
		}
		tls["reality"] = Object{"enabled": true, "public_key": base64.RawURLEncoding.EncodeToString(private.PublicKey().Bytes()), "short_id": sid}
		tls["utls"] = Object{"enabled": true, "fingerprint": "chrome"}
		if str(in, "server_name") == "" {
			if host := str(object(reality["handshake"]), "server"); host != "" {
				tls["server_name"] = host
			}
		}
	} else if str(in, "server_name") == "" {
		warnings = append(warnings, "clientExport.verifySNI")
	}
	return tls, warnings, nil
}
