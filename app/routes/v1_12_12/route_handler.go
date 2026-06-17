package v1_13_0

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sagernet/sing-box/option"
)

// GetRouteRules returns all route rules
func (h *Handler) GetRouteRules(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Route == nil {
		respOK(ctx, c, map[string]any{"rules": []config.RouteRule{}})
		return
	}

	respOK(ctx, c, map[string]any{"rules": cfg.Route.Rules})
}

// AddRouteRule adds a new route rule
func (h *Handler) AddRouteRule(ctx context.Context, c *app.RequestContext) {
	var rule config.RouteRule
	if err := c.Bind(&rule); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			cfg.Route = &config.RouteConfig{}
		}

		cfg.Route.Rules = append(cfg.Route.Rules, rule)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{"message": "route rule added successfully"})
}

// UpdateRouteRule updates a route rule at specific index
func (h *Handler) UpdateRouteRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return
	}

	var rule config.RouteRule
	if err := c.Bind(&rule); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil || index < 0 || index >= len(cfg.Route.Rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}

		cfg.Route.Rules[index] = rule
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "route rule updated successfully",
		"index":   index,
	})
}

// DeleteRouteRule deletes a route rule at specific index
func (h *Handler) DeleteRouteRule(ctx context.Context, c *app.RequestContext) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid index")
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil || index < 0 || index >= len(cfg.Route.Rules) {
			return fmt.Errorf("route rule not found at index %d", index)
		}

		cfg.Route.Rules = append(cfg.Route.Rules[:index], cfg.Route.Rules[index+1:]...)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "route rule deleted successfully",
		"index":   index,
	})
}

// GetRuleSets returns all rule sets
func (h *Handler) GetRuleSets(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Route == nil {
		respOK(ctx, c, map[string]any{"rule_sets": []config.RuleSet{}})
		return
	}

	respOK(ctx, c, map[string]any{"rule_sets": cfg.Route.RuleSet})
}

// GetRuleSetByTag returns a specific rule set by tag
func (h *Handler) GetRuleSetByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if cfg.Route != nil {
		for _, ruleSet := range cfg.Route.RuleSet {
			if ruleSet.Tag == tag {
				respOK(ctx, c, ruleSet)
				return
			}
		}
	}

	respErr(ctx, c, CodeNotFound, "rule set not found")
}

// AddRuleSet adds a new rule set
func (h *Handler) AddRuleSet(ctx context.Context, c *app.RequestContext) {
	var ruleSet config.RuleSet
	if err := c.Bind(&ruleSet); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	if ruleSet.Tag == "" {
		respErr(ctx, c, CodeBadRequest, "tag is required")
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			cfg.Route = &config.RouteConfig{}
		}

		// Reject duplicate tag. The previous form used `continue` inside the
		// loop, which only skipped the check itself and still appended the
		// duplicate after the loop completed.
		for _, existing := range cfg.Route.RuleSet {
			if existing.Tag == ruleSet.Tag {
				return fmt.Errorf("rule set with tag %q already exists", ruleSet.Tag)
			}
		}

		cfg.Route.RuleSet = append(cfg.Route.RuleSet, ruleSet)
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "rule set added successfully",
		"tag":     ruleSet.Tag,
	})
}

// UpdateRuleSet updates an existing rule set
func (h *Handler) UpdateRuleSet(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	var ruleSet config.RuleSet
	if err := c.Bind(&ruleSet); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	ruleSet.Tag = tag

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			return fmt.Errorf("route configuration not found")
		}

		found := false
		for i, existing := range cfg.Route.RuleSet {
			if existing.Tag == tag {
				cfg.Route.RuleSet[i] = ruleSet
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("rule set not found")
		}

		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "rule set updated successfully",
		"tag":     tag,
	})
}

// GetRuleSetReferences returns a dry-run of how deleting :tag would affect
// route.rules and dns.rules, so the frontend can show the user exactly what
// will be stripped or removed before they confirm a cascade delete.
func (h *Handler) GetRuleSetReferences(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	if !config.RuleSetExists(cfg, tag) {
		respErr(ctx, c, CodeNotFound, "rule set not found")
		return
	}

	refs := config.FindRuleSetReferences(cfg, tag)
	routeCount, dnsCount := 0, 0
	for _, ref := range refs {
		if ref.Scope == config.RefScopeRoute {
			routeCount++
		} else {
			dnsCount++
		}
	}

	respOK(ctx, c, map[string]any{
		"tag":         tag,
		"references":  refs,
		"route_count": routeCount,
		"dns_count":   dnsCount,
	})
}

// DeleteRuleSet deletes a rule set. With ?cascade=true it also scrubs the tag
// from every route.rules / dns.rules matcher (deleting rules whose only matcher
// was this tag) in the same validated transaction — otherwise sing-box would
// reject the config for referencing an undefined rule-set.
func (h *Handler) DeleteRuleSet(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")
	cascade, _ := strconv.ParseBool(string(c.Query("cascade")))

	// Pre-flight: surface "not found" and "referenced" as actionable codes
	// instead of an opaque validation error from the write-validate-rollback.
	if cfg, err := h.configManager.GetConfig(); err == nil {
		if !config.RuleSetExists(cfg, tag) {
			respErr(ctx, c, CodeNotFound, "rule set not found")
			return
		}
		if refs := config.FindRuleSetReferences(cfg, tag); len(refs) > 0 && !cascade {
			respErr(ctx, c, CodeConflict, fmt.Sprintf(
				"rule set %q is referenced by %d rule(s); retry with ?cascade=true to remove them", tag, len(refs)))
			return
		}
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			return fmt.Errorf("route configuration not found")
		}

		found := false
		for _, ruleSet := range cfg.Route.RuleSet {
			if ruleSet.Tag == tag {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("rule set not found")
		}

		// Scrub references first so removing the definition leaves a valid config.
		if cascade {
			config.ApplyRuleSetCascade(cfg, tag)
		}

		newRuleSets := make([]config.RuleSet, 0, len(cfg.Route.RuleSet))
		for _, ruleSet := range cfg.Route.RuleSet {
			if ruleSet.Tag != tag {
				newRuleSets = append(newRuleSets, ruleSet)
			}
		}
		cfg.Route.RuleSet = newRuleSets
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "rule set deleted successfully",
		"tag":     tag,
		"cascade": cascade,
	})
}

// GetRouteFinal returns the final route policy
func (h *Handler) GetRouteFinal(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	final := ""
	autoDetectInterface := false
	defaultDomainResolver := ""
	if cfg.Route != nil {
		final = cfg.Route.Final
		autoDetectInterface = cfg.Route.AutoDetectInterface
		if cfg.Route.DefaultDomainResolver != nil {
			defaultDomainResolver = cfg.Route.DefaultDomainResolver.Server
		}
	}

	respOK(ctx, c, map[string]any{
		"final":                   final,
		"auto_detect_interface":   autoDetectInterface,
		"default_domain_resolver": defaultDomainResolver,
	})
}

// UpdateRouteFinal updates route-level policy: the final outbound, interface
// auto-detection, and the default domain resolver. All fields are optional
// pointers so callers can patch a single field (the init wizard sends only
// `final`, the route page may send any subset).
func (h *Handler) UpdateRouteFinal(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Final                 *string `json:"final"`
		AutoDetectInterface   *bool   `json:"auto_detect_interface"`
		DefaultDomainResolver *string `json:"default_domain_resolver"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			cfg.Route = &config.RouteConfig{}
		}

		if req.Final != nil {
			cfg.Route.Final = *req.Final
		}
		if req.AutoDetectInterface != nil {
			cfg.Route.AutoDetectInterface = *req.AutoDetectInterface
		}
		if req.DefaultDomainResolver != nil {
			// Empty string clears the resolver; otherwise it is the DNS server tag.
			server := strings.TrimSpace(*req.DefaultDomainResolver)
			if server == "" {
				cfg.Route.DefaultDomainResolver = nil
			} else {
				cfg.Route.DefaultDomainResolver = &option.DomainResolveOptions{Server: server}
			}
		}
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "route policy updated successfully",
	})
}
