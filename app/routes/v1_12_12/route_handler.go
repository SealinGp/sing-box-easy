package v1_13_0

import (
	"context"
	"fmt"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
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

		// Check if tag already exists
		for _, existing := range cfg.Route.RuleSet {
			if existing.Tag == ruleSet.Tag {
				logger.Warnf("rule set with tag '%s' already exists, skip it", ruleSet.Tag)
				continue
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

// DeleteRuleSet deletes a rule set
func (h *Handler) DeleteRuleSet(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		if cfg.Route == nil {
			return fmt.Errorf("route configuration not found")
		}

		newRuleSets := make([]config.RuleSet, 0)
		found := false

		for _, ruleSet := range cfg.Route.RuleSet {
			if ruleSet.Tag != tag {
				newRuleSets = append(newRuleSets, ruleSet)
			} else {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("rule set not found")
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
	if cfg.Route != nil {
		final = cfg.Route.Final
	}

	respOK(ctx, c, map[string]any{"final": final})
}

// UpdateRouteFinal updates the final route policy
func (h *Handler) UpdateRouteFinal(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Final string `json:"final"`
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

		cfg.Route.Final = req.Final
		return nil
	})

	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "final route policy updated successfully",
		"final":   req.Final,
	})
}
