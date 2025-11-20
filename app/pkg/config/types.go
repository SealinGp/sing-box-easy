package config

import (
	"context"

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
