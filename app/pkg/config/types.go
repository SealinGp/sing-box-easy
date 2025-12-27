package config

import (
	"context"
	"fmt"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// SingBoxConfig wraps option.Options with custom JSON marshal/unmarshal
// It embeds option.Options so all fields are directly accessible
type SingBoxConfig struct {
	option.Options
}

// UnmarshalJSON implements custom JSON unmarshaling with all required registries
func (c *SingBoxConfig) UnmarshalJSON(data []byte) error {
	ctx := CreateContext(context.Background())
	return json.UnmarshalContext(ctx, data, &c.Options)
}

// MarshalJSON implements custom JSON marshaling with all required registries
func (c *SingBoxConfig) MarshalJSON() ([]byte, error) {
	ctx := CreateContext(context.Background())
	return json.MarshalContext(ctx, &c.Options)
}

// Type aliases for backward compatibility
type (
	LogConfig          = option.LogOptions
	ExperimentalConfig = option.ExperimentalOptions
	ClashAPIConfig     = option.ClashAPIOptions
	CacheFileConfig    = option.CacheFileOptions
	Inbound            = option.Inbound
	Outbound           = option.Outbound
	RouteConfig        = option.RouteOptions
	RouteRule          = option.Rule
	DNSRule            = option.DNSRule
	RuleSet            = option.RuleSet
)

// GetOutboundServerKey returns the server endpoint key (server:port) for the outbound
// This identifies which server the node connects to
func GetOutboundServerKey(outbound Outbound) string {
	if outbound.Options == nil {
		return ""
	}

	// Extract server and port to create unique key
	var server, port string

	if opts, ok := outbound.Options.(map[string]interface{}); ok {
		// Get server
		if s, ok := opts["server"].(string); ok {
			server = s
		}

		// Get port - handle both float64 and int
		if p, ok := opts["server_port"].(float64); ok {
			port = fmt.Sprintf("%d", int(p))
		} else if p, ok := opts["server_port"].(int); ok {
			port = fmt.Sprintf("%d", p)
		}

		// Special handling for wireguard which may use peers
		if outbound.Type == "wireguard" && server == "" {
			if peers, ok := opts["peers"].([]interface{}); ok && len(peers) > 0 {
				if peer, ok := peers[0].(map[string]interface{}); ok {
					if addr, ok := peer["address"].(string); ok {
						server = addr
					}
					if p, ok := peer["port"].(float64); ok {
						port = fmt.Sprintf("%d", int(p))
					} else if p, ok := peer["port"].(int); ok {
						port = fmt.Sprintf("%d", p)
					}
				}
			}
		}
	}

	if server != "" && port != "" {
		return fmt.Sprintf("%s:%s", server, port)
	} else if server != "" {
		return server
	}

	return ""
}

// GetOutboundServer returns just the server field from the outbound
// This retrieves only the server address without the port
func GetOutboundServer(outbound Outbound) string {
	if outbound.Options == nil {
		return ""
	}

	var server string

	if opts, ok := outbound.Options.(map[string]interface{}); ok {
		// Get server
		if s, ok := opts["server"].(string); ok {
			server = s
		}

		// Special handling for wireguard which may use peers
		if outbound.Type == "wireguard" && server == "" {
			if peers, ok := opts["peers"].([]interface{}); ok && len(peers) > 0 {
				if peer, ok := peers[0].(map[string]interface{}); ok {
					if addr, ok := peer["address"].(string); ok {
						server = addr
					}
				}
			}
		}
	}

	return server
}

// GenerateUniqueTag generates a unique tag for an outbound node
// Format: "original_tag server:port"
func GenerateUniqueTag(originalTag string, outbound Outbound) string {
	serverKey := GetOutboundServerKey(outbound)
	if serverKey != "" {
		return fmt.Sprintf("%s %s", originalTag, serverKey)
	}
	return originalTag
}
