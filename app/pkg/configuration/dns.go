package configuration

import (
	"context"
	stdjson "encoding/json"
	wirejson "encoding/json"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func (h *Service) GetDNS(ctx context.Context) (any, error) {
	dns, ok, err := h.configManager.GetConfigSection("dns")
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	if !ok {
		return map[string]any{"servers": []any{}}, nil
	}
	return dns, nil
}

func (h *Service) UpdateDNS(ctx context.Context, body []byte) (any, error) {
	var err error

	if err := requireJSONObject(body); err != nil {
		return nil, fault.New(fault.Input, "invalid DNS configuration: "+err.Error())
	}
	err = h.configManager.UpdateConfigSection(ctx, "dns", func(stdjson.RawMessage) (stdjson.RawMessage, error) {
		return cloneRaw(body), nil
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{"message": "DNS configuration updated successfully"}, nil
}

func (h *Service) GetDNSServers(ctx context.Context) (any, error) {
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"servers": rawList(dns, "servers")}, nil
}

func (h *Service) GetDNSServerByTag(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag

	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	for _, server := range rawList(dns, "servers") {
		if rawStringField(server, "tag") == tag {
			return server, nil
		}
	}

	return nil, fault.New(fault.Missing, "DNS server not found")
}

func (h *Service) AddDNSServer(ctx context.Context, body []byte) (any, error) {
	server, errMsg := bindDNSServer(ctx, body)
	if errMsg != "" {
		return nil, fault.New(fault.Input, errMsg)
	}

	if err := validateDNSServer(server); err != nil {
		return nil, fault.New(fault.Invalid, err.Error())
	}

	raw := cloneRaw(body)
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "DNS server added successfully",
		"tag":     server.Tag,
	}, nil
}

func (h *Service) UpdateDNSServer(ctx context.Context, body []byte, pathTag string) (any, error) {
	tag := pathTag

	server, errMsg := bindDNSServer(ctx, body)
	if errMsg != "" {
		return nil, fault.New(fault.Input, errMsg)
	}

	server.Tag = tag

	if err := validateDNSServer(server); err != nil {
		return nil, fault.New(fault.Invalid, err.Error())
	}

	raw, err := withRawStringField(body, "tag", tag)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid DNS server: "+err.Error())
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "DNS server updated successfully",
		"tag":     tag,
	}, nil
}

func (h *Service) DeleteDNSServer(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag

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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "DNS server deleted successfully",
		"tag":     tag,
	}, nil
}

func (h *Service) GetDNSHosts(ctx context.Context) (any, error) {
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	for _, server := range rawList(dns, "servers") {
		if rawStringField(server, "tag") == "dns_lan" && rawStringField(server, "type") == "hosts" {
			var object map[string]stdjson.RawMessage
			if err := stdjson.Unmarshal(server, &object); err != nil {
				return nil, fault.New(fault.Internal, "failed to parse hosts configuration")
			}
			if predefined, ok := object["predefined"]; ok {
				return map[string]any{"hosts": predefined}, nil
			}
		}
	}

	return map[string]any{"hosts": map[string][]string{}}, nil
}

func (h *Service) UpdateDNSHosts(ctx context.Context, body []byte) (any, error) {
	var err error
	if err := requireJSONObject(body); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"message": "DNS hosts updated successfully"}, nil
}

func (h *Service) GetDNSRules(ctx context.Context) (any, error) {
	dns, _, err := readDNSDocument(h.configManager)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"rules": rawList(dns, "rules")}, nil
}

func (h *Service) AddDNSRule(ctx context.Context, body []byte) (any, error) {
	var err error
	if err := requireJSONObject(body); err != nil {
		return nil, fault.New(fault.Input, "invalid DNS rule: "+err.Error())
	}
	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		return setRawList(dns, "rules", append(rawList(dns, "rules"), cloneRaw(body)))
	})

	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{"message": "DNS rule added successfully"}, nil
}

func (h *Service) ReorderDNSRules(ctx context.Context, body []byte) (any, error) {
	var request ReorderRulesRequest
	if err := wirejson.Unmarshal(body, &request); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}

	err := h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		rules := rawList(dns, "rules")
		reordered, err := applyOrder(rules, request.Order)
		if err != nil {
			return err
		}
		return setRawList(dns, "rules", reordered)
	})

	if err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}

	return map[string]any{"message": "DNS rules reordered successfully"}, nil
}

func (h *Service) UpdateDNSRule(ctx context.Context, body []byte, pathIndex string) (any, error) {
	indexStr := pathIndex
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid index")
	}

	if err := requireJSONObject(body); err != nil {
		return nil, fault.New(fault.Input, "invalid DNS rule: "+err.Error())
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
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "DNS rule updated successfully",
		"index":   index,
	}, nil
}

func (h *Service) DeleteDNSRule(ctx context.Context, pathIndex string) (any, error) {
	indexStr := pathIndex
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid index")
	}

	err = h.updateDNSDocument(ctx, func(dns map[string]stdjson.RawMessage) error {
		rules := rawList(dns, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("DNS rule not found at index %d", index)
		}
		return setRawList(dns, "rules", append(rules[:index], rules[index+1:]...))
	})

	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}

	return map[string]any{
		"message": "DNS rule deleted successfully",
		"index":   index,
	}, nil
}

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

func (h *Service) updateDNSDocument(
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
