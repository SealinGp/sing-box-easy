package protocol

import (
	"encoding/base64"
	"testing"
)

func TestParserParseValidatesVMessRequiredFields(t *testing.T) {
	raw := `{"v":"2","ps":"missing-id","add":"example.com","port":443,"aid":0,"net":"tcp","type":"none","host":"","path":"","tls":"none"}`
	uri := "vmess://" + base64.StdEncoding.EncodeToString([]byte(raw))

	parser, err := NewParser(uri)
	if err != nil {
		t.Fatalf("NewParser returned error: %v", err)
	}

	_, err = parser.Parse()
	if err == nil {
		t.Fatal("Parse returned nil error, want vmess uuid validation error")
	}
	if err.Error() != "vmess uuid is required" {
		t.Fatalf("Parse error = %q, want %q", err, "vmess uuid is required")
	}
}

func TestParserParseValidatesShadowsocksRequiredFields(t *testing.T) {
	credentials := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:"))
	uri := "ss://" + credentials + "@example.com:8388#missing-password"

	parser, err := NewParser(uri)
	if err != nil {
		t.Fatalf("NewParser returned error: %v", err)
	}

	_, err = parser.Parse()
	if err == nil {
		t.Fatal("Parse returned nil error, want shadowsocks password validation error")
	}
	if err.Error() != "shadowsocks password is required" {
		t.Fatalf("Parse error = %q, want %q", err, "shadowsocks password is required")
	}
}
