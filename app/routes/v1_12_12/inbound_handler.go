package v1_13_0

import (
	"context"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetInbounds returns all inbound configurations
func (h *Handler) GetInbounds(ctx context.Context, c *app.RequestContext) {
	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"inbounds": cfg.Inbounds,
	})
}

// GetInboundByTag returns a specific inbound by tag
func (h *Handler) GetInboundByTag(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	cfg, err := h.configManager.GetConfig()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	for _, inbound := range cfg.Inbounds {
		if inbound.Tag == tag {
			c.JSON(consts.StatusOK, inbound)
			return
		}
	}

	c.JSON(consts.StatusNotFound, utils.H{
		"error": "inbound not found",
	})
}

// AddInbound adds a new inbound
func (h *Handler) AddInbound(ctx context.Context, c *app.RequestContext) {
	var inbound config.Inbound
	if err := c.Bind(&inbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if inbound.Tag == "" {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "tag is required",
		})
		return
	}

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		// Check if tag already exists
		for _, existing := range cfg.Inbounds {
			if existing.Tag == inbound.Tag {
				return fmt.Errorf("inbound with tag '%s' already exists", inbound.Tag)
			}
		}

		cfg.Inbounds = append(cfg.Inbounds, inbound)
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusCreated, utils.H{
		"message": "inbound added successfully",
		"tag":     inbound.Tag,
	})
}

// UpdateInbound updates an existing inbound
func (h *Handler) UpdateInbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	var inbound config.Inbound
	if err := c.Bind(&inbound); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	// Ensure the tag matches
	inbound.Tag = tag

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		found := false
		for i, existing := range cfg.Inbounds {
			if existing.Tag == tag {
				cfg.Inbounds[i] = inbound
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("inbound not found")
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
		"message": "inbound updated successfully",
		"tag":     tag,
	})
}

// DeleteInbound deletes an inbound
func (h *Handler) DeleteInbound(ctx context.Context, c *app.RequestContext) {
	tag := c.Param("tag")

	err := h.configManager.UpdateConfig(func(cfg *config.SingBoxConfig) error {
		newInbounds := make([]config.Inbound, 0)
		found := false

		for _, inbound := range cfg.Inbounds {
			if inbound.Tag != tag {
				newInbounds = append(newInbounds, inbound)
			} else {
				found = true
			}
		}

		if !found {
			return fmt.Errorf("inbound not found")
		}

		cfg.Inbounds = newInbounds
		return nil
	})

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, utils.H{
		"message": "inbound deleted successfully",
		"tag":     tag,
	})
}
