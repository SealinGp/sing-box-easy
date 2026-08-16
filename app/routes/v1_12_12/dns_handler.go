package v1_13_0

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
)

// GetDNS returns the complete DNS configuration
func (h *Handler) GetDNS(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.DNS == nil {
		respOK(ctx, c, map[string]any{"servers": []option.DNSServerOptions{}})
		return
	}

	respOK(ctx, c, cfg.DNS)
}

// UpdateDNS updates the complete DNS configuration
func (h *Handler) UpdateDNS(ctx context.Context, c *app.RequestContext) {
	var dnsOptions option.DNSOptions

	// Use sing-box JSON deserialization to properly parse DNS config
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}

	dnsCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(dnsCtx, body, &dnsOptions); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS configuration: "+err.Error())
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		cfg.DNS = &dnsOptions
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS configuration updated successfully"})
}

// GetDNSServers returns all DNS servers
func (h *Handler) GetDNSServers(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.DNS == nil {
		respOK(ctx, c, map[string]any{"servers": []option.DNSServerOptions{}})
		return
	}

	respOK(ctx, c, map[string]any{"servers": cfg.DNS.Servers})
}

// GetDNSServerByTag returns a specific DNS server by tag
func (h *Handler) GetDNSServerByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.DNS != nil {
		for _, server := range cfg.DNS.Servers {
			if server.Tag == tag {
				respOK(ctx, c, server)
				return
			}
		}
	}

	respErr(ctx, c, CodeNotFound, "DNS server not found")
}

// AddDNSServer adds a new DNS server
func (h *Handler) AddDNSServer(ctx context.Context, c *app.RequestContext) {
	server, errMsg := bindDNSServer(ctx, c)
	if errMsg != "" {
		respErr(ctx, c, CodeBadRequest, errMsg)
		return
	}

	if err := validateDNSServer(server); err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &option.DNSOptions{}
		}

		// Check if tag already exists
		for _, existing := range cfg.DNS.Servers {
			if existing.Tag == server.Tag {
				return fmt.Errorf("DNS server with tag '%s' already exists", server.Tag)
			}
		}

		cfg.DNS.Servers = append(cfg.DNS.Servers, server)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "DNS server added successfully",
		"tag":     server.Tag,
	})
}

// UpdateDNSServer updates an existing DNS server
func (h *Handler) UpdateDNSServer(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	server, errMsg := bindDNSServer(ctx, c)
	if errMsg != "" {
		respErr(ctx, c, CodeBadRequest, errMsg)
		return
	}

	server.Tag = tag

	if err := validateDNSServer(server); err != nil {
		respErr(ctx, c, CodeValidationError, err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			return fmt.Errorf("DNS configuration not found")
		}

		found := false
		for i, existing := range cfg.DNS.Servers {
			if existing.Tag == tag {
				cfg.DNS.Servers[i] = server
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("DNS server not found")
		}

		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "DNS server updated successfully",
		"tag":     tag,
	})
}

// DeleteDNSServer deletes a DNS server
func (h *Handler) DeleteDNSServer(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			return fmt.Errorf("DNS configuration not found")
		}

		newServers := make([]option.DNSServerOptions, 0)
		found := false

		for _, server := range cfg.DNS.Servers {
			if server.Tag != tag {
				newServers = append(newServers, server)
			} else {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("DNS server not found")
		}

		cfg.DNS.Servers = newServers
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "DNS server deleted successfully",
		"tag":     tag,
	})
}

// GetDNSHosts returns the hosts configuration
func (h *Handler) GetDNSHosts(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.DNS != nil {
		for _, server := range cfg.DNS.Servers {
			if server.Tag == "dns_lan" && server.Type == "hosts" {
				hostsOpts, ok := server.Options.(option.HostsDNSServerOptions)
				if !ok {
					respErr(ctx, c, CodeInternalError, "failed to parse hosts configuration")
					return
				}

				respOK(ctx, c, map[string]any{"hosts": hostsOpts.Predefined})
				return
			}
		}
	}

	respOK(ctx, c, map[string]any{"hosts": map[string][]string{}})
}

// UpdateDNSHosts updates the hosts configuration
func (h *Handler) UpdateDNSHosts(ctx context.Context, c *app.RequestContext) {
	var hosts badjson.TypedMap[string, badoption.Listable[netip.Addr]]
	if err := c.Bind(&hosts); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &option.DNSOptions{}
		}

		found := false
		for i, server := range cfg.DNS.Servers {
			if server.Type == "hosts" {
				hostsOpts, ok := server.Options.(*option.HostsDNSServerOptions)
				if !ok {
					continue
				}
				hostsOpts.Predefined = &hosts
				server.Options = hostsOpts

				cfg.DNS.Servers[i] = server
				found = true
				break
			}
		}

		if !found {
			opts := &option.HostsDNSServerOptions{
				Predefined: &hosts,
			}
			cfg.DNS.Servers = append(cfg.DNS.Servers, option.DNSServerOptions{
				Tag:     "dns_lan",
				Type:    "hosts",
				Options: opts,
			})
		}

		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS hosts updated successfully"})
}

// GetDNSRules returns all DNS rules
func (h *Handler) GetDNSRules(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.DNS == nil {
		respOK(ctx, c, map[string]any{"rules": []option.DNSRule{}})
		return
	}

	respOK(ctx, c, map[string]any{"rules": cfg.DNS.Rules})
}

// AddDNSRule adds a new DNS rule
func (h *Handler) AddDNSRule(ctx context.Context, c *app.RequestContext) {
	// Parse with sing-box's JSON (not c.Bind): option.DNSRule is polymorphic and
	// relies on its custom UnmarshalJSON to default an omitted "type" to "default"
	// and populate the per-type options. Hertz's reflection binder skips that, so
	// the rule would marshal back with an empty type ("unknown rule type: ").
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}
	var rule option.DNSRule
	if err := json.UnmarshalContext(config.CreateContext(ctx), body, &rule); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS rule: "+err.Error())
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &option.DNSOptions{}
		}

		cfg.DNS.Rules = append(cfg.DNS.Rules, rule)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS rule added successfully"})
}

// UpdateDNSRule updates a DNS rule at specific index
func (h *Handler) UpdateDNSRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return
	}

	// Parse with sing-box's JSON (not c.Bind): option.DNSRule is polymorphic and
	// relies on its custom UnmarshalJSON to default an omitted "type" to "default"
	// and populate the per-type options. Hertz's reflection binder skips that, so
	// the rule would marshal back with an empty type ("unknown rule type: ").
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}
	var rule option.DNSRule
	if err := json.UnmarshalContext(config.CreateContext(ctx), body, &rule); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS rule: "+err.Error())
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil || index < 0 || index >= len(cfg.DNS.Rules) {
			return fmt.Errorf("DNS rule not found at index %d", index)
		}

		cfg.DNS.Rules[index] = rule
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "DNS rule updated successfully",
		"index":   index,
	})
}

// DeleteDNSRule deletes a DNS rule at specific index
func (h *Handler) DeleteDNSRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil || index < 0 || index >= len(cfg.DNS.Rules) {
			return fmt.Errorf("DNS rule not found at index %d", index)
		}

		cfg.DNS.Rules = append(cfg.DNS.Rules[:index], cfg.DNS.Rules[index+1:]...)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "DNS rule deleted successfully",
		"index":   index,
	})
}
