package configuration

import (
	"context"
	stdjson "encoding/json"
	wirejson "encoding/json"
	"fmt"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
)

func applyOrder[T any](items []T, order []int) ([]T, error) {
	if len(order) != len(items) {
		return nil, fmt.Errorf("order must list all %d rules, got %d", len(items), len(order))
	}
	seen := make([]bool, len(items))
	reordered := make([]T, 0, len(items))
	for _, idx := range order {
		if idx < 0 || idx >= len(items) {
			return nil, fmt.Errorf("index %d out of range", idx)
		}
		if seen[idx] {
			return nil, fmt.Errorf("index %d listed more than once", idx)
		}
		seen[idx] = true
		reordered = append(reordered, items[idx])
	}
	return reordered, nil
}

func (h *Service) GetRouteRules(ctx context.Context) (any, error) {
	route, err := h.readRouteDocument()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	return map[string]any{"rules": rawList(route, "rules")}, nil
}

func (h *Service) AddRouteRule(ctx context.Context, body []byte) (any, error) {
	body, err := objectBody(body)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		return setRawList(route, "rules", append(rawList(route, "rules"), cloneRaw(body)))
	})
	return mutationResult(err, "route rule added successfully", nil)
}

func (h *Service) ReorderRouteRules(ctx context.Context, body []byte) (any, error) {
	var request ReorderRulesRequest
	if err := wirejson.Unmarshal(body, &request); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	err := h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules, err := applyOrder(rawList(route, "rules"), request.Order)
		if err != nil {
			return err
		}
		return setRawList(route, "rules", rules)
	})
	if err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	return map[string]any{"message": "route rules reordered successfully"}, nil
}

func (h *Service) UpdateRouteRule(ctx context.Context, body []byte, pathIndex string) (any, error) {
	index, err := strconv.Atoi(pathIndex)
	if err != nil {
		return nil, fault.New(fault.Invalid, "invalid index")
	}
	body, err = objectBody(body)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules := rawList(route, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}
		rules[index] = cloneRaw(body)
		return setRawList(route, "rules", rules)
	})
	return mutationResult(err, "route rule updated successfully", map[string]any{"index": index})
}

func (h *Service) DeleteRouteRule(ctx context.Context, pathIndex string) (any, error) {
	index, err := strconv.Atoi(pathIndex)
	if err != nil {
		return nil, fault.New(fault.Invalid, "invalid index")
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules := rawList(route, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}
		return setRawList(route, "rules", append(rules[:index], rules[index+1:]...))
	})
	return mutationResult(err, "route rule deleted successfully", map[string]any{"index": index})
}

func (h *Service) GetRuleSets(ctx context.Context) (any, error) {
	route, err := h.readRouteDocument()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	return map[string]any{"rule_sets": rawList(route, "rule_set")}, nil
}

func (h *Service) GetRuleSetByTag(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag
	route, err := h.readRouteDocument()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	for _, ruleSet := range rawList(route, "rule_set") {
		if rawStringField(ruleSet, "tag") == tag {
			return ruleSet, nil
		}
	}
	return nil, fault.New(fault.Missing, "rule set not found")
}

func (h *Service) AddRuleSet(ctx context.Context, body []byte) (any, error) {
	body, err := objectBody(body)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	tag := rawStringField(body, "tag")
	if tag == "" {
		return nil, fault.New(fault.Input, "tag is required")
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		ruleSets := rawList(route, "rule_set")
		for _, existing := range ruleSets {
			if rawStringField(existing, "tag") == tag {
				return fmt.Errorf("rule set with tag %q already exists", tag)
			}
		}
		return setRawList(route, "rule_set", append(ruleSets, cloneRaw(body)))
	})
	return mutationResult(err, "rule set added successfully", map[string]any{"tag": tag})
}

func (h *Service) UpdateRuleSet(ctx context.Context, body []byte, pathTag string) (any, error) {
	tag := pathTag
	body, err := objectBody(body)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	body, err = withRawStringField(body, "tag", tag)
	if err != nil {
		return nil, fault.New(fault.Input, err.Error())
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		ruleSets := rawList(route, "rule_set")
		for i, existing := range ruleSets {
			if rawStringField(existing, "tag") == tag {
				ruleSets[i] = body
				return setRawList(route, "rule_set", ruleSets)
			}
		}
		return fmt.Errorf("rule set not found")
	})
	return mutationResult(err, "rule set updated successfully", map[string]any{"tag": tag})
}

func (h *Service) GetRuleSetReferences(ctx context.Context, pathTag string) (any, error) {
	tag := pathTag
	refs, exists, err := h.rawRuleSetReferences(tag)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	if !exists {
		return nil, fault.New(fault.Missing, "rule set not found")
	}
	routeCount, dnsCount := 0, 0
	for _, ref := range refs {
		if ref.Scope == config.RefScopeRoute {
			routeCount++
		} else {
			dnsCount++
		}
	}
	return map[string]any{"tag": tag, "references": refs, "route_count": routeCount, "dns_count": dnsCount}, nil
}

func (h *Service) DeleteRuleSet(ctx context.Context, pathTag string, cascadeQuery string) (any, error) {
	tag := pathTag
	cascade, _ := strconv.ParseBool(cascadeQuery)
	refs, exists, err := h.rawRuleSetReferences(tag)
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	if !exists {
		return nil, fault.New(fault.Missing, "rule set not found")
	}
	if len(refs) > 0 && !cascade {
		return nil, fault.New(fault.Conflict, fmt.Sprintf("rule set %q is referenced by %d rule(s); retry with ?cascade=true to remove them", tag, len(refs)))
	}
	err = h.configManager.UpdateConfigSections(ctx, func(sections map[string]stdjson.RawMessage) error {
		route, err := readRawObjectSection(sections["route"], "route")
		if err != nil {
			return err
		}
		ruleSets := rawList(route, "rule_set")
		kept := make([]stdjson.RawMessage, 0, len(ruleSets))
		for _, ruleSet := range ruleSets {
			if rawStringField(ruleSet, "tag") != tag {
				kept = append(kept, ruleSet)
			}
		}
		if err := setRawList(route, "rule_set", kept); err != nil {
			return err
		}
		if cascade {
			rules, err := scrubRawRules(rawList(route, "rules"), tag, false)
			if err != nil {
				return err
			}
			if err := setRawList(route, "rules", rules); err != nil {
				return err
			}
			dns, err := readRawObjectSection(sections["dns"], "dns")
			if err != nil {
				return err
			}
			dnsRules, err := scrubRawRules(rawList(dns, "rules"), tag, true)
			if err != nil {
				return err
			}
			if err := setRawList(dns, "rules", dnsRules); err != nil {
				return err
			}
			sections["dns"], err = stdjson.Marshal(dns)
			if err != nil {
				return err
			}
		}
		sections["route"], err = stdjson.Marshal(route)
		return err
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"message": "rule set deleted successfully", "tag": tag, "cascade": cascade}, nil
}

func (h *Service) GetRouteFinal(ctx context.Context) (any, error) {
	route, err := h.readRouteDocument()
	if err != nil {
		return nil, fault.New(fault.Internal, err.Error())
	}
	var final, resolver string
	var auto bool
	_ = stdjson.Unmarshal(route["final"], &final)
	_ = stdjson.Unmarshal(route["auto_detect_interface"], &auto)
	if stdjson.Unmarshal(route["default_domain_resolver"], &resolver) != nil {
		var object map[string]stdjson.RawMessage
		if stdjson.Unmarshal(route["default_domain_resolver"], &object) == nil {
			_ = stdjson.Unmarshal(object["server"], &resolver)
		}
	}
	return map[string]any{"final": final, "auto_detect_interface": auto, "default_domain_resolver": resolver}, nil
}

func (h *Service) UpdateRouteFinal(ctx context.Context, body []byte) (any, error) {
	var request map[string]stdjson.RawMessage
	body, err := objectBody(body)
	if err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	if err := stdjson.Unmarshal(body, &request); err != nil {
		return nil, fault.New(fault.Input, "invalid request body: "+err.Error())
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		for _, key := range []string{"final", "auto_detect_interface"} {
			if value, ok := request[key]; ok {
				route[key] = value
			}
		}
		if value, ok := request["default_domain_resolver"]; ok {
			var server string
			if err := stdjson.Unmarshal(value, &server); err != nil {
				return fmt.Errorf("default_domain_resolver must be a string")
			}
			server = strings.TrimSpace(server)
			if server == "" {
				delete(route, "default_domain_resolver")
			} else {
				raw, _ := stdjson.Marshal(server)
				route["default_domain_resolver"] = raw
			}
		}
		return nil
	})
	return mutationResult(err, "route policy updated successfully", nil)
}

func (h *Service) readRouteDocument() (map[string]stdjson.RawMessage, error) {
	raw, _, err := h.configManager.GetConfigSection("route")
	if err != nil {
		return nil, err
	}
	return readRawObjectSection(raw, "route")
}

func (h *Service) updateRouteDocument(ctx context.Context, update func(map[string]stdjson.RawMessage) error) error {
	return h.configManager.UpdateConfigSection(ctx, "route", func(raw stdjson.RawMessage) (stdjson.RawMessage, error) {
		route, err := readRawObjectSection(raw, "route")
		if err != nil {
			return nil, err
		}
		if err := update(route); err != nil {
			return nil, err
		}
		return stdjson.Marshal(route)
	})
}

type ReorderRulesRequest struct {
	Order []int `json:"order"`
}
