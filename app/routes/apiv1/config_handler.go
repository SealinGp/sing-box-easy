package apiv1

import (
	"context"
	"errors"
	"strconv"

	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetConfig returns the current configuration
func (h *Handler) GetConfig(ctx context.Context, c *app.RequestContext) {
	document, err := h.configDocument().Current()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, document)
}

// GetRawConfig returns the active document directly for exact export and raw
// editor use. Unlike GetConfig, this endpoint has no response envelope.
func (h *Handler) GetRawConfig(ctx context.Context, c *app.RequestContext) {
	document, err := h.configDocument().Current()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	c.Data(consts.StatusOK, "application/json; charset=utf-8", document.Raw)
}

// UpdateConfig saves the provided configuration
func (h *Handler) UpdateConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configDocument().Save(ctx, c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "configuration saved successfully",
	})
}

// ValidateConfig validates the provided configuration
func (h *Handler) ValidateConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configDocument().Validate(ctx, c.Request.Body()); err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, map[string]any{
		"valid":   true,
		"message": "configuration is valid",
	})
}

// GetBackupConfig returns the backup configuration
func (h *Handler) GetBackupConfig(ctx context.Context, c *app.RequestContext) {
	document, err := h.configDocument().Backup()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, document)
}

// RollbackConfig restores the most recent historical configuration.
func (h *Handler) RollbackConfig(ctx context.Context, c *app.RequestContext) {
	if err := h.configDocument().RestoreLatest(); err != nil {
		respondOperationError(ctx, c, err)
		return
	}

	respOK(ctx, c, map[string]any{
		"message": "configuration rolled back successfully",
	})
}

// ListConfigVersions returns historical config metadata (newest first).
func (h *Handler) ListConfigVersions(ctx context.Context, c *app.RequestContext) {
	versions, err := h.configDocument().ListHistory()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{"versions": versions})
}

// GetConfigVersion returns the full config of a single historical version.
func (h *Handler) GetConfigVersion(ctx context.Context, c *app.RequestContext) {
	document, err := h.configDocument().HistoryVersion(versionIDParam(c))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, document)
}

// GetCoreInfo reports the installed core version and structured-editor feature
// gates. Raw config validation remains available for newer, untested versions.
func (h *Handler) GetCoreInfo(ctx context.Context, c *app.RequestContext) {
	capabilities, err := h.configDocument().CoreInfo(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{
		"version":        capabilities.Version,
		"supported":      capabilities.Supported,
		"minimum":        capabilities.Minimum,
		"maximum_tested": capabilities.MaximumTested,
		"capabilities": map[string]bool{
			"dns_evaluate":       capabilities.DNSEvaluate,
			"dns_respond":        capabilities.DNSRespond,
			"dns_race":           capabilities.DNSRace,
			"dns_match_response": capabilities.DNSMatchResponse,
			"dns_optimistic":     capabilities.DNSOptimistic,
		},
	})
}

func respondConfigError(ctx context.Context, c *app.RequestContext, err error) {
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		resp(ctx, c, CodeValidationError, map[string]any{
			"stage":        validationErr.Stage,
			"core_version": validationErr.CoreVersion,
		}, validationErr.Error())
		return
	}
	respErr(ctx, c, CodeInternalError, err.Error())
}

// DeleteConfigVersion removes a single historical version by id. This only
// affects history; the live config is untouched.
func (h *Handler) DeleteConfigVersion(ctx context.Context, c *app.RequestContext) {
	if err := h.configDocument().DeleteHistory(versionIDParam(c)); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{
		"message": "configuration version deleted",
	})
}

// DeleteConfigVersionsBatch removes multiple historical versions in one call.
// Used by the dashboard's multi-select delete. Only history is affected; the
// live config.json is never touched.
func (h *Handler) DeleteConfigVersionsBatch(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		IDs []int64 `json:"ids"`
	}
	var req Request
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
		return
	}
	deleted, err := h.configDocument().DeleteHistoryBatch(req.IDs)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{
		"message":       "configuration versions deleted",
		"deleted_count": deleted,
	})
}

// RollbackToConfigVersion restores a specific historical version to live config.
func (h *Handler) RollbackToConfigVersion(ctx context.Context, c *app.RequestContext) {
	if err := h.configDocument().RestoreVersion(versionIDParam(c)); err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, map[string]any{
		"message": "configuration rolled back to selected version",
	})
}

// versionIDParam reads the :id path segment. An unparsable id becomes 0, which
// the service rejects with the same "invalid version id" a negative one gets —
// the range rule lives there, not in the transport.
func versionIDParam(c *app.RequestContext) int64 {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
