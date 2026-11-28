package clientprofile

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

func TestProtocolCredentialsAndCoreParsing(t *testing.T) {
	cases := []struct {
		kind, fields string
		want         map[string]any
	}{
		{"vmess", `"users":[{"name":"display","uuid":"00000000-0000-4000-8000-000000000001","alterId":0}]`, Object{"uuid": "00000000-0000-4000-8000-000000000001", "alter_id": float64(0)}},
		{"vless", `"users":[{"name":"display","uuid":"00000000-0000-4000-8000-000000000001","flow":"xtls-rprx-vision"}]`, Object{"flow": "xtls-rprx-vision"}},
		{"trojan", `"users":[{"name":"display","password":"test-secret"}]`, Object{"password": "test-secret"}},
		{"anytls", `"users":[{"name":"display","password":"test-secret"}]`, Object{"password": "test-secret"}},
		{"tuic", `"users":[{"name":"display","uuid":"00000000-0000-4000-8000-000000000001","password":"test-secret"}]`, Object{"password": "test-secret"}},
		{"hysteria", `"users":[{"name":"display","auth_str":"test-secret"}],"up_mbps":10,"down_mbps":20`, Object{"auth_str": "test-secret", "up_mbps": float64(20), "down_mbps": float64(10)}},
		{"hysteria2", `"users":[{"name":"display","password":"test-secret"}],"obfs":{"type":"salamander","password":"obfs-secret"}`, Object{"password": "test-secret"}},
		{"shadowsocks", `"method":"2022-blake3-aes-128-gcm","password":"AAAAAAAAAAAAAAAAAAAAAA==","users":[{"name":"display","password":"AQEBAQEBAQEBAQEBAQEBAQ=="}]`, Object{"password": "AAAAAAAAAAAAAAAAAAAAAA==:AQEBAQEBAQEBAQEBAQEBAQ=="}},
		{"socks", `"users":[{"username":"login","password":"test-secret"}]`, Object{"username": "login"}},
		{"http", `"users":[{"username":"login","password":"test-secret"}]`, Object{"username": "login"}},
		{"mixed", `"users":[{"username":"login","password":"test-secret"}]`, Object{"type": "socks", "username": "login"}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			var inbound Object
			if err := json.Unmarshal([]byte(`{"type":"`+tc.kind+`","tag":"test","listen_port":443,`+tc.fields+`}`), &inbound); err != nil {
				t.Fatal(err)
			}
			result, err := Generate(Request{Inbound: inbound, Server: "example.com"})
			if err != nil {
				t.Fatal(err)
			}
			for key, want := range tc.want {
				if result.Node[key] != want {
					t.Fatalf("%s mismatch", key)
				}
			}
			if tc.kind != "socks" && tc.kind != "http" && tc.kind != "mixed" {
				if _, ok := result.Node["username"]; ok {
					t.Fatal("display name leaked into authentication")
				}
			}
			raw, _ := json.Marshal(result.Node)
			var parsed option.Outbound
			if err = singjson.UnmarshalContext(config.CreateContext(context.Background()), raw, &parsed); err != nil {
				t.Fatal(err)
			}
			if binary := os.Getenv("SBE_TEST_CORE"); binary != "" {
				// TLS-dependent protocols need TLS enabled for the runtime constructor.
				if tc.kind == "tuic" || tc.kind == "hysteria" || tc.kind == "hysteria2" || tc.kind == "anytls" {
					result.Node["tls"] = Object{"enabled": true, "server_name": "example.com"}
				}
				data, _ := json.Marshal(Object{"outbounds": []any{result.Node}, "dns": Object{"servers": []any{Object{"type": "local", "tag": "dns"}}}, "route": Object{"default_domain_resolver": "dns"}})
				path := filepath.Join(t.TempDir(), "config.json")
				os.WriteFile(path, data, 0600)
				if out, err := exec.Command(binary, "check", "-c", path).CombinedOutput(); err != nil {
					t.Fatalf("core check: %s", out)
				}
			}
		})
	}
}
func TestRealityPublicKeyAndTLSIsolation(t *testing.T) {
	key, _ := ecdh.X25519().GenerateKey(rand.Reader)
	in := Object{"enabled": true, "key": "SERVER SECRET", "key_path": "/server/key", "certificate_path": "/server/cert", "certificate_provider": Object{"type": "acme"}, "reality": Object{"enabled": true, "private_key": base64.RawURLEncoding.EncodeToString(key.Bytes()), "handshake": Object{"server": "sni.example.com"}, "short_id": []any{"1234"}}}
	tls, _, err := clientTLS(in, "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if tls["server_name"] != "sni.example.com" {
		t.Fatal("SNI not preserved")
	}
	if object(tls["reality"])["public_key"] != base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()) {
		t.Fatal("wrong public key")
	}
	raw, _ := json.Marshal(tls)
	for _, forbidden := range []string{"private_key", "key_path", "certificate_path", "certificate_provider", "SERVER SECRET"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("leaked %s", forbidden)
		}
	}
	if in["key"] != "SERVER SECRET" {
		t.Fatal("mutated server config")
	}
}
func TestUnsupportedAndUserSelection(t *testing.T) {
	for _, kind := range []string{"tun", "direct", "redirect", "tproxy", "shadowtls", "naive"} {
		if _, err := Generate(Request{Inbound: Object{"type": kind, "listen_port": float64(443)}, Server: "example.com"}); err == nil {
			t.Fatalf("exported %s as complete node", kind)
		}
	}
	in := Object{"type": "trojan", "listen_port": float64(443), "users": []any{Object{"password": "first"}, Object{"password": "second"}}}
	result, err := Generate(Request{Inbound: in, Server: "example.com", User: 1})
	if err != nil || result.Node["password"] != "second" {
		t.Fatal("wrong user")
	}
	if _, err = Generate(Request{Inbound: in, Server: "example.com", User: 2}); err == nil {
		t.Fatal("invalid user accepted")
	}
}

func TestMissingCredentialsRejected(t *testing.T) {
	for _, kind := range []string{"vmess", "vless", "trojan", "anytls", "tuic", "hysteria", "hysteria2", "shadowsocks"} {
		if _, err := Generate(Request{Inbound: Object{"type": kind, "listen_port": float64(443)}, Server: "example.com"}); err == nil {
			t.Fatalf("accepted missing credentials for %s", kind)
		}
	}
}
