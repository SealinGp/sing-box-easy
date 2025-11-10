package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json/badoption"
)

type Vmess struct {
	Tag string `json:"tag,omitempty"`
	option.VMessOutboundOptions
}

// vmessJSON represents the JSON structure of a VMess URI
type vmessJSON struct {
	Version  string `json:"v"`
	PS       string `json:"ps"`
	Add      string `json:"add"`
	Port     int    `json:"port"`
	ID       string `json:"id"`
	AlterId  int    `json:"aid"`
	Security string `json:"scy"`
	Net      string `json:"net"`
	Type     string `json:"type"` //?
	Host     string `json:"host"`
	Path     string `json:"path"`
	TLS      string `json:"tls"`
	SNI      string `json:"sni"`
}

func (vj *vmessJSON) toVemss(v *Vmess) error {
	// Populate Vmess struct
	v.Tag = vj.PS
	v.Server = vj.Add
	v.ServerPort = uint16(vj.Port)
	v.UUID = vj.ID
	v.AlterId = vj.AlterId
	v.Security = vj.Security

	if vj.Net != "" && vj.Net != "tcp" {
		v.Transport = vj.buildTransport()
	}

	if vj.TLS == "tls" {
		v.OutboundTLSOptionsContainer.TLS = &option.OutboundTLSOptions{
			Enabled: true,
		}

		if vj.Host != "" {
			v.OutboundTLSOptionsContainer.TLS.ServerName = vj.Host
		}
	}

	// Set security (default to auto if not specified)
	if vj.Security != "" {
		v.Security = vj.Security
	} else {
		v.Security = "auto"
	}

	return nil
}

// buildTransport creates transport options based on network type
func (vj *vmessJSON) buildTransport() *option.V2RayTransportOptions {
	transport := &option.V2RayTransportOptions{}

	switch vj.Net {
	case "ws":
		transport.Type = "ws"
		wsOptions := option.V2RayWebsocketOptions{}
		if vj.Path != "" {
			wsOptions.Path = vj.Path
		}
		if vj.Host != "" {
			wsOptions.Headers = map[string]badoption.Listable[string]{
				"Host": badoption.Listable[string]{vj.Host},
			}
		}
		transport.WebsocketOptions = wsOptions

	case "grpc":
		transport.Type = "grpc"
		grpcOptions := option.V2RayGRPCOptions{}
		if vj.Path != "" {
			grpcOptions.ServiceName = vj.Path
		}
		transport.GRPCOptions = grpcOptions

	case "http":
		transport.Type = "http"
		httpOptions := option.V2RayHTTPOptions{}
		if vj.Host != "" {
			httpOptions.Host = badoption.Listable[string]{vj.Host}
		}
		if vj.Path != "" {
			httpOptions.Path = vj.Path
		}
		transport.HTTPOptions = httpOptions
	}

	return transport
}

func (v *Vmess) TypeName() string {
	return "vmess"
}

func (v *Vmess) Schema() string {
	return "vmess://"
}

// uri=vmess://ewogICJ2IjogIjIiLAogICJwcyI6ICJiYWNrdXAxIiwKICAiYWRkIjogIjE5OS4xODAuMTE1LjEyNiIsCiAgInBvcnQiOiA1NDUwOSwKICAiaWQiOiAiMjM0Yjc1ODUtMjZjMy00Y2QwLWZiODctNzI2ZmVlMzgzMDk4IiwKICAiYWlkIjogMCwKICAibmV0IjogInRjcCIsCiAgInR5cGUiOiAibm9uZSIsCiAgImhvc3QiOiAiIiwKICAicGF0aCI6ICIiLAogICJ0bHMiOiAibm9uZSIKfQ==
//
//	{
//	  "v": "2",
//	  "ps": "backup1",
//	  "add": "199.180.115.126",
//	  "port": 54509,
//	  "id": "234b7585-26c3-4cd0-fb87-726fee383098",
//	  "aid": 0,
//	  "net": "tcp",
//	  "type": "none",
//	  "host": "",
//	  "path": "",
//	  "tls": "none"
//	}
func (v *Vmess) Parse(uri string) (*node.SubNode, error) {
	// Remove the prefix
	vmessStr := strings.TrimPrefix(uri, v.Schema())

	// Decode base64
	vbs, err := v.decodeBase64(vmessStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vmess URI: %w", err)
	}

	// Unmarshal JSON
	vj := new(vmessJSON)
	if err := json.Unmarshal(vbs, vj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vmess JSON: %w", err)
	}

	err = vj.toVemss(v)
	if err != nil {
		return nil, err
	}

	sn := &node.SubNode{
		Type:    v.TypeName(),
		Tag:     v.Tag,
		Options: v.VMessOutboundOptions,
	}

	return sn, nil
}

// decodeBase64 tries different base64 encoding variations
func (v *Vmess) decodeBase64(encoded string) ([]byte, error) {
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
