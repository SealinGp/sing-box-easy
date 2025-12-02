package v1_13_0

import (
	"context"
	"fmt"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// GetOutbounds returns all outbound configurations
func (h *Handler) GetOutbounds(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	// Use sing-box JSON serialization to preserve all outbound fields
	outboundCtx := config.CreateContext(ctx)
	response := map[string]any{"outbounds": cfg.Outbounds}
	responseJSON, err := json.MarshalContext(outboundCtx, response)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to serialize outbounds: " + err.Error(),
		})
		return
	}

	c.Data(consts.StatusOK, "application/json; charset=utf-8", responseJSON)
}

// GetOutboundByTag returns a specific outbound by tag
func (h *Handler) GetOutboundByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	for _, outbound := range cfg.Outbounds {
		if outbound.Tag == tag {
			// Use sing-box JSON serialization to preserve all outbound fields
			outboundCtx := config.CreateContext(ctx)
			outboundJSON, err := json.MarshalContext(outboundCtx, outbound)
			if err != nil {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"error": "failed to serialize outbound: " + err.Error(),
				})
				return
			}
			c.Data(consts.StatusOK, "application/json; charset=utf-8", outboundJSON)
			return
		}
	}

	c.JSON(consts.StatusNotFound, utils.H{
		"error": "outbound not found",
	})
}

// AddOutbound adds a new outbound
func (h *Handler) AddOutbound(ctx context.Context, c *app.RequestContext) {
	// Use sing-box JSON deserialization to properly parse outbound config
	body, err := c.Body()
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "failed to read request body: " + err.Error(),
		})
		return
	}

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid outbound configuration: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if outbound.Tag == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "tag is required",
		})
		return
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		// Check if tag already exists
		for _, existing := range cfg.Outbounds {
			if existing.Tag == outbound.Tag {
				return fmt.Errorf("outbound with tag '%s' already exists", outbound.Tag)
			}
		}

		cfg.Outbounds = append(cfg.Outbounds, outbound)
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"message": "outbound added successfully",
		"tag":     outbound.Tag,
	})
}

// AddOutboundsBatch adds multiple outbounds at once
func (h *Handler) AddOutboundsBatch(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Outbounds []config.Outbound `json:"outbounds"`
	}

	var req Request
	data := c.Request.Body()
	jsonCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(jsonCtx, data, &req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if len(req.Outbounds) == 0 {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "outbounds array is required and cannot be empty",
		})
		return
	}

	// Validate all outbounds first
	tagMap := make(map[string]bool)
	for i, outbound := range req.Outbounds {
		if outbound.Tag == "" {
			c.JSON(consts.StatusBadRequest, utils.H{
				"error": fmt.Sprintf("outbound at index %d: tag is required", i),
			})
			return
		}

		// Check for duplicate tags in request
		if tagMap[outbound.Tag] {
			c.JSON(consts.StatusBadRequest, utils.H{
				"error": fmt.Sprintf("duplicate tag '%s' in request", outbound.Tag),
			})
			return
		}
		tagMap[outbound.Tag] = true
	}

	var addedTags []string
	var skippedTags []string

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		// Build existing tags map
		existingTags := make(map[string]bool)
		for _, existing := range cfg.Outbounds {
			existingTags[existing.Tag] = true
		}

		// Add outbounds that don't exist
		for _, outbound := range req.Outbounds {
			if existingTags[outbound.Tag] {
				skippedTags = append(skippedTags, outbound.Tag)
				continue
			}

			cfg.Outbounds = append(cfg.Outbounds, outbound)
			addedTags = append(addedTags, outbound.Tag)
		}

		if len(addedTags) == 0 {
			logger.Warn("all outbounds already exist")
		}

		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	response := utils.H{
		"message":     "outbounds batch add completed",
		"added_count": len(addedTags),
		"added_tags":  addedTags,
	}

	if len(skippedTags) > 0 {
		response["skipped_count"] = len(skippedTags)
		response["skipped_tags"] = skippedTags
		response["message"] = fmt.Sprintf("added %d outbounds, skipped %d existing outbounds", len(addedTags), len(skippedTags))
	}

	respData, err := json.Marshal(response)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.Data(consts.StatusOK, "application/json; charset=utf-8", respData)
}

// UpdateOutbound updates an existing outbound
func (h *Handler) UpdateOutbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	// Use sing-box JSON deserialization to properly parse outbound config
	body, err := c.Body()
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "failed to read request body: " + err.Error(),
		})
		return
	}

	var outbound option.Outbound
	outboundCtx := config.CreateContext(ctx)
	if err := json.UnmarshalContext(outboundCtx, body, &outbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid outbound configuration: " + err.Error(),
		})
		return
	}

	// Ensure the tag matches
	outbound.Tag = tag

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		found := false
		for i, existing := range cfg.Outbounds {
			if existing.Tag == tag {
				cfg.Outbounds[i] = outbound
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("outbound not found")
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
		"message": "outbound updated successfully",
		"tag":     tag,
	})
}

// DeleteOutbound deletes an outbound
func (h *Handler) DeleteOutbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag") //tag or index
	idx, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		idx = -1
	}

	err = h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		newOutbounds := make([]config.Outbound, 0)
		found := false

		for i, outbound := range cfg.Outbounds {
			if idx > -1 && int(idx) == i {
				found = true
				continue
			}
			if outbound.Tag == tag {
				found = true
				continue
			}

			newOutbounds = append(newOutbounds, outbound)
		}

		if !found {
			return fmt.Errorf("outbound not found")
		}

		cfg.Outbounds = newOutbounds
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "outbound deleted successfully",
		"tag":     tag,
	})
}

// GetOutboundGroups returns all group type outbounds (selector/urltest)
func (h *Handler) GetOutboundGroups(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	groups := make([]option.Outbound, 0)
	for _, outbound := range cfg.Outbounds {
		if outbound.Type == "selector" || outbound.Type == "urltest" {
			groups = append(groups, outbound)
		}
	}

	// Use sing-box JSON serialization to preserve all outbound fields
	outboundCtx := config.CreateContext(ctx)
	response := map[string]any{"groups": groups}
	responseJSON, err := json.MarshalContext(outboundCtx, response)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": "failed to serialize outbound groups: " + err.Error(),
		})
		return
	}

	c.Data(consts.StatusOK, "application/json; charset=utf-8", responseJSON)
}

// UpdateOutboundMembers updates the members of a group outbound
func (h *Handler) UpdateOutboundMembers(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	type Request struct {
		Outbounds []string `json:"outbounds"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		found := false
		for i, outbound := range cfg.Outbounds {
			if outbound.Tag == tag {
				if outbound.Type != "selector" && outbound.Type != "urltest" {
					return fmt.Errorf("outbound '%s' is not a group type (selector/urltest)", tag)
				}

				// Update the Outbounds field in the Options
				switch outbound.Type {
				case "selector":
					if opts, ok := outbound.Options.(*option.SelectorOutboundOptions); ok {
						opts.Outbounds = req.Outbounds
						cfg.Outbounds[i].Options = opts
					} else {
						return fmt.Errorf("invalid selector outbound options")
					}
				case "urltest":
					if opts, ok := outbound.Options.(*option.URLTestOutboundOptions); ok {
						opts.Outbounds = req.Outbounds
						cfg.Outbounds[i].Options = opts
					} else {
						return fmt.Errorf("invalid urltest outbound options")
					}
				}

				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("outbound not found")
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
		"message": "outbound members updated successfully",
		"tag":     tag,
	})
}
