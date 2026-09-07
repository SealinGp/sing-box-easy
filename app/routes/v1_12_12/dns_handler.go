package v1_13_0

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetDNS returns the complete DNS configuration
func (h *Handler) GetDNS(ctx context.Context, c *app.RequestContext) {
	dns, ok, err := h.configManager.GetConfigSection("dns")
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	if !ok {
		respOK(ctx, c, map[string]any{"servers": []any{}})
		return
	}
	respOK(ctx, c, dns)
}

// UpdateDNS updates the complete DNS configuration
func (h *Handler) UpdateDNS(ctx context.Context, c *app.RequestContext) {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}

	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS configuration: "+err.Error())
		return
	}
	err = h.configManager.UpdateConfigSection(ctx, "dns", func(stdjson.RawMessage) (stdjson.RawMessage, error) {
		return cloneRaw(body), nil
	})
	if err != nil {
		respondConfigError(ctx, c, err)
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS configuration updated successfully"})
}

// GetDNSServers returns all DNS servers
func (h *Handler) GetDNSServers(ctx context.Context, c *app.RequestContext) {
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"servers": rawList(dns, "servers")})
}

// GetDNSServerByTag returns a specific DNS server by tag
func (h *Handler) GetDNSServerByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	for _, server := range rawList(dns, "servers") {
		if rawStringField(server, "tag") == tag {
			respOK(ctx, c, server)
			return
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

	raw := cloneRaw(c.Request.Body())
	err := h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		servers := rawList(dns, "servers")
		for _, existing := range servers {
			if rawStringField(existing, "tag") == server.Tag {
				return fmt.Errorf("DNS server with tag '%s' already exists", server.Tag)
			}
		}
		return setRawList(dns, "servers", append(servers, raw))
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

	raw, err := withRawStringField(c.Request.Body(), "tag", tag)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS server: "+err.Error())
		return
	}
	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		servers := rawList(dns, "servers")
		for i, existing := range servers {
			if rawStringField(existing, "tag") == tag {
				servers[i] = raw
				return setRawList(dns, "servers", servers)
			}
		}
		return fmt.Errorf("DNS server not found")
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

	err := h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		servers := rawList(dns, "servers")
		newServers := make([]stdjson.RawMessage, 0, len(servers))
		for _, server := range servers {
			if rawStringField(server, "tag") != tag {
				newServers = append(newServers, server)
			}
		}
		if len(newServers) == len(servers) {
			return fmt.Errorf("DNS server not found")
		}
		return setRawList(dns, "servers", newServers)
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
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	for _, server := range rawList(dns, "servers") {
		if rawStringField(server, "tag") == "dns_lan" && rawStringField(server, "type") == "hosts" {
			var object map[string]stdjson.RawMessage
			if err := stdjson.Unmarshal(server, &object); err != nil {
				respErr(ctx, c, CodeInternalError, "failed to parse hosts configuration")
				return
			}
			if predefined, ok := object["predefined"]; ok {
				respOK(ctx, c, map[string]any{"hosts": predefined})
				return
			}
		}
	}

	respOK(ctx, c, map[string]any{"hosts": map[string][]string{}})
}

// UpdateDNSHosts updates the hosts configuration
func (h *Handler) UpdateDNSHosts(ctx context.Context, c *app.RequestContext) {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}
	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		servers := rawList(dns, "servers")
		for i, server := range servers {
			if rawStringField(server, "type") == "hosts" && rawStringField(server, "tag") == "dns_lan" {
				updated, err := withRawField(server, "predefined", cloneRaw(body))
				if err != nil {
					return err
				}
				servers[i] = updated
				return setRawList(dns, "servers", servers)
			}
		}
		server, err := stdjson.Marshal(map[string]stdjson.RawMessage{
			"type":       stdjson.RawMessage(`"hosts"`),
			"tag":        stdjson.RawMessage(`"dns_lan"`),
			"predefined": cloneRaw(body),
		})
		if err != nil {
			return err
		}
		return setRawList(dns, "servers", append(servers, server))
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS hosts updated successfully"})
}

// GetDNSRules returns all DNS rules
func (h *Handler) GetDNSRules(ctx context.Context, c *app.RequestContext) {
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"rules": rawList(dns, "rules")})
}

// AddDNSRule adds a new DNS rule
func (h *Handler) AddDNSRule(ctx context.Context, c *app.RequestContext) {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}
	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS rule: "+err.Error())
		return
	}
	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		return setRawList(dns, "rules", append(rawList(dns, "rules"), cloneRaw(body)))
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS rule added successfully"})
}

// ReorderDNSRules reorders DNS rules according to a permutation of indices.
//
// Registered on the collection path (PUT /dns/rules) for the same reason as
// the route one: `/dns/rules/:index` already owns the next path segment, and a
// static sibling there collides in the Hertz router.
func (h *Handler) ReorderDNSRules(ctx context.Context, c *app.RequestContext) {
	var body ReorderRulesRequest
	if err := c.Bind(&body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		rules := rawList(dns, "rules")
		reordered, err := applyOrder(rules, body.Order)
		if err != nil {
			return err
		}
		return setRawList(dns, "rules", reordered)
	})

	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "DNS rules reordered successfully"})
}

// UpdateDNSRule updates a DNS rule at specific index
func (h *Handler) UpdateDNSRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return
	}

	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return
	}
	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid DNS rule: "+err.Error())
		return
	}
	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		rules := rawList(dns, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("DNS rule not found at index %d", index)
		}
		rules[index] = cloneRaw(body)
		return setRawList(dns, "rules", rules)
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

	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		rules := rawList(dns, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("DNS rule not found at index %d", index)
		}
		return setRawList(dns, "rules", append(rules[:index], rules[index+1:]...))
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

// readDNSDocument decodes only the DNS object. It deliberately does not decode
// option.DNSOptions: the panel may be built against 1.12 while the installed
// core accepts newer actions such as evaluate/respond.
func readDNSDocument(manager *config.Manager) (map[string]stdjson.RawMessage, bool, error) {
	raw, ok, err := manager.GetConfigSection("dns")
	if err != nil || !ok {
		return map[string]stdjson.RawMessage{}, ok, err
	}
	var dns map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(raw, &dns); err != nil {
		return nil, true, fmt.Errorf("decode DNS configuration: %w", err)
	}
	if dns == nil {
		dns = map[string]stdjson.RawMessage{}
	}
	return dns, true, nil
}

func (h *Handler) updateDNSDocument(
	ctx context.Context,
	update func(map[string]stdjson.RawMessage) error,
) error {
	return h.configManager.UpdateConfigSection(ctx, "dns", func(raw stdjson.RawMessage) (stdjson.RawMessage, error) {
		dns := map[string]stdjson.RawMessage{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := stdjson.Unmarshal(raw, &dns); err != nil {
				return nil, fmt.Errorf("decode DNS configuration: %w", err)
			}
		}
		if err := update(dns); err != nil {
			return nil, err
		}
		return stdjson.Marshal(dns)
	})
}

func rawList(object map[string]stdjson.RawMessage, key string) []stdjson.RawMessage {
	var values []stdjson.RawMessage
	_ = stdjson.Unmarshal(object[key], &values)
	if values == nil {
		return []stdjson.RawMessage{}
	}
	return values
}

func setRawList(object map[string]stdjson.RawMessage, key string, values []stdjson.RawMessage) error {
	raw, err := stdjson.Marshal(values)
	if err != nil {
		return err
	}
	object[key] = raw
	return nil
}

func rawStringField(raw stdjson.RawMessage, key string) string {
	var object map[string]stdjson.RawMessage
	if stdjson.Unmarshal(raw, &object) != nil {
		return ""
	}
	var value string
	_ = stdjson.Unmarshal(object[key], &value)
	return value
}

func withRawStringField(raw stdjson.RawMessage, key, value string) (stdjson.RawMessage, error) {
	field, err := stdjson.Marshal(value)
	if err != nil {
		return nil, err
	}
	return withRawField(raw, key, field)
}

func withRawField(raw stdjson.RawMessage, key string, value stdjson.RawMessage) (stdjson.RawMessage, error) {
	var object map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("expected JSON object")
	}
	object[key] = value
	return stdjson.Marshal(object)
}

func requireJSONObject(raw []byte) error {
	if !stdjson.Valid(raw) {
		return fmt.Errorf("invalid JSON")
	}
	var object map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(raw, &object); err != nil || object == nil {
		return fmt.Errorf("expected JSON object")
	}
	return nil
}

func cloneRaw(raw []byte) stdjson.RawMessage {
	return append(stdjson.RawMessage(nil), raw...)
}
