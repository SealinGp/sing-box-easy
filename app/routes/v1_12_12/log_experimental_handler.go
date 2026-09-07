package v1_13_0

import (
	"context"
	stdjson "encoding/json"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
)

// GetLog returns the log configuration without decoding unrelated sections.
func (h *Handler) GetLog(ctx context.Context, c *app.RequestContext) {
	raw, ok, err := h.configManager.GetConfigSection("log")
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	if !ok {
		respOK(ctx, c, map[string]any{})
		return
	}
	respOK(ctx, c, raw)
}

func (h *Handler) UpdateLog(ctx context.Context, c *app.RequestContext) {
	if err := h.replaceSectionFromBody(ctx, c, "log"); err != nil {
		return
	}
	respOK(ctx, c, map[string]any{"message": "log configuration updated successfully"})
}

func (h *Handler) GetClashAPI(ctx context.Context, c *app.RequestContext) {
	h.getExperimentalChild(ctx, c, "clash_api")
}

func (h *Handler) UpdateClashAPI(ctx context.Context, c *app.RequestContext) {
	if err := h.replaceExperimentalChildFromBody(ctx, c, "clash_api"); err != nil {
		return
	}
	respOK(ctx, c, map[string]any{"message": "Clash API configuration updated successfully"})
}

func (h *Handler) GetCacheFile(ctx context.Context, c *app.RequestContext) {
	h.getExperimentalChild(ctx, c, "cache_file")
}

func (h *Handler) UpdateCacheFile(ctx context.Context, c *app.RequestContext) {
	// rdrc_timeout is a duration and rejects "". An untouched optional UI field
	// means absent, not an explicit empty duration.
	c.Request.SetBody(dropEmptyJSONFields(c.Request.Body(), "rdrc_timeout"))
	if err := h.replaceExperimentalChildFromBody(ctx, c, "cache_file"); err != nil {
		return
	}
	respOK(ctx, c, map[string]any{"message": "cache file configuration updated successfully"})
}

func (h *Handler) GetV2RayAPI(ctx context.Context, c *app.RequestContext) {
	h.getExperimentalChild(ctx, c, "v2ray_api")
}

func (h *Handler) UpdateV2RayAPI(ctx context.Context, c *app.RequestContext) {
	if err := h.replaceExperimentalChildFromBody(ctx, c, "v2ray_api"); err != nil {
		return
	}
	respOK(ctx, c, map[string]any{"message": "V2Ray API configuration updated successfully"})
}

func (h *Handler) getExperimentalChild(ctx context.Context, c *app.RequestContext, name string) {
	experimental, ok, err := h.configManager.GetConfigSection("experimental")
	if err != nil {
		respErr(ctx, c, CodeInternalError, err.Error())
		return
	}
	if !ok {
		respOK(ctx, c, map[string]any{})
		return
	}
	var object map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(experimental, &object); err != nil {
		respErr(ctx, c, CodeInternalError, "failed to parse experimental configuration: "+err.Error())
		return
	}
	child, ok := object[name]
	if !ok || string(child) == "null" {
		respOK(ctx, c, map[string]any{})
		return
	}
	respOK(ctx, c, child)
}

func (h *Handler) replaceSectionFromBody(
	ctx context.Context,
	c *app.RequestContext,
	section string,
) error {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return err
	}
	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return err
	}
	err = h.configManager.UpdateConfigSection(ctx, section, func(stdjson.RawMessage) (stdjson.RawMessage, error) {
		return cloneRaw(body), nil
	})
	if err != nil {
		respondConfigError(ctx, c, err)
	}
	return err
}

func (h *Handler) replaceExperimentalChildFromBody(
	ctx context.Context,
	c *app.RequestContext,
	name string,
) error {
	body, err := c.Body()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, "failed to read request body: "+err.Error())
		return err
	}
	if err := requireJSONObject(body); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return err
	}
	err = h.configManager.UpdateConfigSection(ctx, "experimental", func(raw stdjson.RawMessage) (stdjson.RawMessage, error) {
		object := map[string]stdjson.RawMessage{}
		if len(raw) > 0 && string(raw) != "null" {
			if err := stdjson.Unmarshal(raw, &object); err != nil {
				return nil, fmt.Errorf("failed to parse experimental configuration: %w", err)
			}
		}
		object[name] = cloneRaw(body)
		return stdjson.Marshal(object)
	})
	if err != nil {
		respondConfigError(ctx, c, err)
	}
	return err
}
