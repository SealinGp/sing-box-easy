package protocol

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
)

type Shadowsocks struct {
	Tag string `json:"tag,omitempty"`
	option.ShadowsocksROutboundOptions
}

func (s *Shadowsocks) TypeName() string {
	return "shadowsocks"
}

func (s *Shadowsocks) Schema() string {
	return "ss://"
}

func (s *Shadowsocks) Parse(uri string) (*node.SubNode, error) {
	// Remove the refix
	ssData := strings.TrimPrefix(uri, s.Schema())

	// Split by # to get the node name (tag)
	parts := strings.SplitN(ssData, "#", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid ss URI format: missing tag")
	}

	// Split off the ?query
	ssInfoParts := strings.SplitN(parts[0], "?", 2)
	ssInfo := ssInfoParts[0]
	encodedTag := parts[1]

	// URL decode the tag
	tag, err := url.QueryUnescape(encodedTag)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tag: %w", err)
	}
	s.Tag = tag

	// Split the ss info by @ to separate credentials and server info
	infoParts := strings.SplitN(ssInfo, "@", 2)
	if len(infoParts) != 2 {
		return nil, fmt.Errorf("invalid ss URI format: missing @ separator")
	}

	encodedCredentials := infoParts[0]
	serverInfo := infoParts[1]

	err = s.decodeCredentials(encodedCredentials)
	if err != nil {
		return nil, err
	}

	// Parse server and port
	err = s.decodeServerInfo(serverInfo)
	if err != nil {
		return nil, err
	}

	sn := &node.SubNode{
		Type: s.TypeName(),
		Tag:  s.Tag,
	}
	sn.Options = s.ShadowsocksROutboundOptions
	return sn, nil
}

func (s *Shadowsocks) decodeCredentials(credentials string) error {
	// Decode the credentials (method:password)
	cri, err := s.decodeBase64(credentials)
	if err != nil {
		return fmt.Errorf("failed to decode credentials: %w", err)
	}

	// Parse method and password
	credStr := string(cri)
	credParts := strings.SplitN(credStr, ":", 2)
	if len(credParts) != 2 {
		return fmt.Errorf("invalid credentials format, expected method:password")
	}
	s.Method = credParts[0]
	s.Password = credParts[1]
	return nil
}

// parseServerInfo parses server:port from the server info string
func (s *Shadowsocks) decodeServerInfo(serverInfo string) error {
	// Handle IPv6 addresses (enclosed in brackets)
	if strings.HasPrefix(serverInfo, "[") {
		closeBracket := strings.Index(serverInfo, "]")
		if closeBracket == -1 {
			return fmt.Errorf("invalid IPv6 address format")

		}

		server := serverInfo[1:closeBracket]
		remaining := serverInfo[closeBracket+1:]

		if !strings.HasPrefix(remaining, ":") {
			return fmt.Errorf("missing port after IPv6 address")
		}

		portStr := remaining[1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port number: %w", err)
		}
		s.Server = server
		s.ServerPort = uint16(port)
		return nil
	}

	// Handle IPv4 addresses and hostnames
	lastColon := strings.LastIndex(serverInfo, ":")
	if lastColon == -1 {
		return fmt.Errorf("invalid server info: missing port")
	}

	server := serverInfo[:lastColon]
	portStr := serverInfo[lastColon+1:]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port number out of range: %d", port)
	}

	s.Server = server
	s.ServerPort = uint16(port)
	return nil
}

// decodeBase64 tries different base64 encoding variations
func (s *Shadowsocks) decodeBase64(encoded string) ([]byte, error) {
	// Try standard base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}

	// Try URL-safe base64 encoding
	decoded, err = base64.URLEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}

	// Try raw standard base64 (no padding)
	decoded, err = base64.RawStdEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}

	// Try raw URL-safe base64 (no padding)
	decoded, err = base64.RawURLEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("unable to decode base64 with any encoding")
}
