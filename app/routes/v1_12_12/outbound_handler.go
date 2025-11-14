package v1_13_0

import (
	"context"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
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

	c.JSON(consts.StatusOK, utils.H{
		"outbounds": cfg.Outbounds,
	})
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
			c.JSON(consts.StatusOK, outbound)
			return
		}
	}

	c.JSON(consts.StatusNotFound, utils.H{
		"error": "outbound not found",
	})
}

// AddOutbound adds a new outbound
func (h *Handler) AddOutbound(ctx context.Context, c *app.RequestContext) {
	var outbound config.Outbound
	if err := c.Bind(&outbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
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

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
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
	if err := c.Bind(&req); err != nil {
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
			return fmt.Errorf("all outbounds already exist")
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
		"message":      "outbounds batch add completed",
		"added_count":  len(addedTags),
		"added_tags":   addedTags,
	}

	if len(skippedTags) > 0 {
		response["skipped_count"] = len(skippedTags)
		response["skipped_tags"] = skippedTags
		response["message"] = fmt.Sprintf("added %d outbounds, skipped %d existing outbounds", len(addedTags), len(skippedTags))
	}

	c.JSON(consts.StatusCreated, response)
}

// UpdateOutbound updates an existing outbound
func (h *Handler) UpdateOutbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	var outbound config.Outbound
	if err := c.Bind(&outbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	// Ensure the tag matches
	outbound.Tag = tag

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
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
	tag := c.Param("tag")

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		newOutbounds := make([]config.Outbound, 0)
		found := false

		for _, outbound := range cfg.Outbounds {
			if outbound.Tag != tag {
				newOutbounds = append(newOutbounds, outbound)
			} else {
				found = true
			}
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

	groups := make([]config.Outbound, 0)
	for _, outbound := range cfg.Outbounds {
		if outbound.Type == "selector" || outbound.Type == "urltest" {
			groups = append(groups, outbound)
		}
	}

	c.JSON(consts.StatusOK, utils.H{
		"groups": groups,
	})
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
				cfg.Outbounds[i].Outbounds = req.Outbounds
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
