package v1_13_0

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
)

type ReorderRulesRequest struct {
	Order []int `json:"order"`
}

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

func (h *Handler) GetRouteRules(ctx context.Context, c *app.RequestContext) {
	route, err := h.readRouteDocument()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"rules": rawList(route, "rules")})
}

func (h *Handler) AddRouteRule(ctx context.Context, c *app.RequestContext) {
	body, err := objectBody(c)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		return setRawList(route, "rules", append(rawList(route, "rules"), cloneRaw(body)))
	})
	h.respondRouteMutation(ctx, c, err, "route rule added successfully", nil)
}

func (h *Handler) ReorderRouteRules(ctx context.Context, c *app.RequestContext) {
	var request ReorderRulesRequest
	if err := c.Bind(&request); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	err := h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules, err := applyOrder(rawList(route, "rules"), request.Order)
		if err != nil {
			return err
		}
		return setRawList(route, "rules", rules)
	})
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"message": "route rules reordered successfully"})
}

func (h *Handler) UpdateRouteRule(ctx context.Context, c *app.RequestContext) {
	index, ok := routeIndex(ctx, c)
	if !ok {
		return
	}
	body, err := objectBody(c)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	err = h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules := rawList(route, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}
		rules[index] = cloneRaw(body)
		return setRawList(route, "rules", rules)
	})
	h.respondRouteMutation(ctx, c, err, "route rule updated successfully", map[string]any{"index": index})
}

func (h *Handler) DeleteRouteRule(ctx context.Context, c *app.RequestContext) {
	index, ok := routeIndex(ctx, c)
	if !ok {
		return
	}
	err := h.updateRouteDocument(ctx, func(route map[string]stdjson.RawMessage) error {
		rules := rawList(route, "rules")
		if index < 0 || index >= len(rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}
		return setRawList(route, "rules", append(rules[:index], rules[index+1:]...))
	})
	h.respondRouteMutation(ctx, c, err, "route rule deleted successfully", map[string]any{"index": index})
}

func (h *Handler) GetRuleSets(ctx context.Context, c *app.RequestContext) {
	route, err := h.readRouteDocument()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	respOK(ctx, c, map[string]any{"rule_sets": rawList(route, "rule_set")})
}

func (h *Handler) GetRuleSetByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	route, err := h.readRouteDocument()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	for _, ruleSet := range rawList(route, "rule_set") {
		if rawStringField(ruleSet, "tag") == tag {
			respOK(ctx, c, ruleSet)
			return
		}
	}
	respErr(ctx, c, CodeNotFound, "rule set not found")
}

func (h *Handler) AddRuleSet(ctx context.Context, c *app.RequestContext) {
	body, err := objectBody(c)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	tag := rawStringField(body, "tag")
	if tag == "" {
		respErr(ctx, c, CodeBadRequest, "tag is required")
		return
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
	h.respondRouteMutation(ctx, c, err, "rule set added successfully", map[string]any{"tag": tag})
}

func (h *Handler) UpdateRuleSet(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	body, err := objectBody(c)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	body, err = withRawStringField(body, "tag", tag)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
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
	h.respondRouteMutation(ctx, c, err, "rule set updated successfully", map[string]any{"tag": tag})
}

func (h *Handler) GetRuleSetReferences(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	refs, exists, err := h.rawRuleSetReferences(tag)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	if !exists {
		respErr(ctx, c, CodeNotFound, "rule set not found")
		return
	}
	routeCount, dnsCount := 0, 0
	for _, ref := range refs {
		if ref.Scope == config.RefScopeRoute {
			routeCount++
		} else {
			dnsCount++
		}
	}
	respOK(ctx, c, map[string]any{"tag": tag, "references": refs, "route_count": routeCount, "dns_count": dnsCount})
}

func (h *Handler) DeleteRuleSet(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	cascade, _ := strconv.ParseBool(string(c.Query("cascade")))
	refs, exists, err := h.rawRuleSetReferences(tag)
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	if !exists {
		respErr(ctx, c, CodeNotFound, "rule set not found")
		return
	}
	if len(refs) > 0 && !cascade {
		respErr(ctx, c, CodeConflict, fmt.Sprintf("rule set %q is referenced by %d rule(s); retry with ?cascade=true to remove them", tag, len(refs)))
		return
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
		respondConfigError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"message": "rule set deleted successfully", "tag": tag, "cascade": cascade})
}

func (h *Handler) GetRouteFinal(ctx context.Context, c *app.RequestContext) {
	route, err := h.readRouteDocument()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
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
	respOK(ctx, c, map[string]any{"final": final, "auto_detect_interface": auto, "default_domain_resolver": resolver})
}

func (h *Handler) UpdateRouteFinal(ctx context.Context, c *app.RequestContext) {
	var request map[string]stdjson.RawMessage
	body, err := objectBody(c)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := stdjson.Unmarshal(body, &request); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
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
	h.respondRouteMutation(ctx, c, err, "route policy updated successfully", nil)
}

func (h *Handler) readRouteDocument() (map[string]stdjson.RawMessage, error) {
	raw, _, err := h.configManager.GetConfigSection("route")
	if err != nil {
		return nil, err
	}
	return readRawObjectSection(raw, "route")
}

func (h *Handler) updateRouteDocument(ctx context.Context, update func(map[string]stdjson.RawMessage) error) error {
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

func objectBody(c *app.RequestContext) ([]byte, error) {
	body, err := c.Body()
	if err != nil {
		return nil, err
	}
	if err := requireJSONObject(body); err != nil {
		return nil, err
	}
	return body, nil
}

func routeIndex(ctx context.Context, c *app.RequestContext) (int, bool) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return 0, false
	}
	return index, true
}

func (h *Handler) respondRouteMutation(ctx context.Context, c *app.RequestContext, err error, message string, extra map[string]any) {
	if err != nil {
		respondConfigError(ctx, c, err)
		return
	}
	data := map[string]any{"message": message}
	for key, value := range extra {
		data[key] = value
	}
	respOK(ctx, c, data)
}
