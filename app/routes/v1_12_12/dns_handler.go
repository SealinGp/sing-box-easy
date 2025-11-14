package v1_13_0

import (
	"context"
	"fmt"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetDNSServers returns all DNS servers
func (h *Handler) GetDNSServers(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.DNS == nil {
		c.JSON(consts.StatusOK, utils.H{
			"servers": []config.DNSServer{},
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"servers": cfg.DNS.Servers,
	})
}

// GetDNSServerByTag returns a specific DNS server by tag
func (h *Handler) GetDNSServerByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.DNS != nil {
		for _, server := range cfg.DNS.Servers {
			if server.Tag == tag {
				c.JSON(consts.StatusOK, server)
				return
			}
		}
	}

	c.JSON(consts.StatusNotFound, utils.H{
		"error": "DNS server not found",
	})
}

// AddDNSServer adds a new DNS server
func (h *Handler) AddDNSServer(ctx context.Context, c *app.RequestContext) {
	var server config.DNSServer
	if err := c.Bind(&server); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if server.Tag == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "tag is required",
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &config.DNSConfig{}
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"message": "DNS server added successfully",
		"tag":     server.Tag,
	})
}

// UpdateDNSServer updates an existing DNS server
func (h *Handler) UpdateDNSServer(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	var server config.DNSServer
	if err := c.Bind(&server); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	server.Tag = tag

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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
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

		newServers := make([]config.DNSServer, 0)
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "DNS server deleted successfully",
		"tag":     tag,
	})
}

// GetDNSHosts returns the hosts configuration
func (h *Handler) GetDNSHosts(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.DNS != nil {
		for _, server := range cfg.DNS.Servers {
			if server.Tag == "dns_lan" && server.Type == "hosts" {
				c.JSON(consts.StatusOK, utils.H{
					"hosts": server.Predefined,
				})
				return
			}
		}
	}

	c.JSON(consts.StatusOK, utils.H{
		"hosts": map[string][]string{},
	})
}

// UpdateDNSHosts updates the hosts configuration
func (h *Handler) UpdateDNSHosts(ctx context.Context, c *app.RequestContext) {
	var hosts map[string][]string
	if err := c.Bind(&hosts); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &config.DNSConfig{}
		}

		found := false
		for i, server := range cfg.DNS.Servers {
			if server.Tag == "dns_lan" {
				cfg.DNS.Servers[i].Predefined = hosts
				found = true
				break
			}
		}

		if !found {
			// Create new hosts server
			cfg.DNS.Servers = append(cfg.DNS.Servers, config.DNSServer{
				Tag:        "dns_lan",
				Type:       "hosts",
				Predefined: hosts,
			})
		}

		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "DNS hosts updated successfully",
	})
}

// GetDNSRules returns all DNS rules
func (h *Handler) GetDNSRules(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	if cfg.DNS == nil {
		c.JSON(consts.StatusOK, utils.H{
			"rules": []config.DNSRule{},
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"rules": cfg.DNS.Rules,
	})
}

// AddDNSRule adds a new DNS rule
func (h *Handler) AddDNSRule(ctx context.Context, c *app.RequestContext) {
	var rule config.DNSRule
	if err := c.Bind(&rule); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.DNS == nil {
			cfg.DNS = &config.DNSConfig{}
		}

		cfg.DNS.Rules = append(cfg.DNS.Rules, rule)
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"message": "DNS rule added successfully",
	})
}

// UpdateDNSRule updates a DNS rule at specific index
func (h *Handler) UpdateDNSRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid index",
		})
		return
	}

	var rule config.DNSRule
	if err := c.Bind(&rule); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "DNS rule updated successfully",
		"index":   index,
	})
}

// DeleteDNSRule deletes a DNS rule at specific index
func (h *Handler) DeleteDNSRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid index",
		})
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
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "DNS rule deleted successfully",
		"index":   index,
	})
}
