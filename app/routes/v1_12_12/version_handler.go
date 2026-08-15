package v1_13_0

import (
	"context"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/appupdate"
	"github.com/cloudwego/hertz/pkg/app"
)

// releaseView is the release shape returned to the frontend. It deliberately
// omits GitHub fields the UI has no use for.
type releaseView struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	URL         string `json:"url"`
	Notes       string `json:"notes"`
	IsCurrent   bool   `json:"is_current"`
	IsNewer     bool   `json:"is_newer"`
}

// GetVersionStatus returns the running version alongside the newest published
// release, so the UI can render "<current> -> <latest> [Update]".
//
// GET /version?refresh=true
func (h *Handler) GetVersionStatus(ctx context.Context, c *app.RequestContext) {
	force := parseBoolQuery(c, "refresh")
	respOK(ctx, c, h.updater.CheckStatus(force))
}

// ListVersions returns the published releases newest-first so the user can pick
// a specific version instead of only the latest.
//
// GET /version/releases?refresh=true
func (h *Handler) ListVersions(ctx context.Context, c *app.RequestContext) {
	force := parseBoolQuery(c, "refresh")

	releases, err := h.updater.Releases(force)
	if err != nil {
		respErr(ctx, c, CodeServiceError, err.Error())
		return
	}

	current := appupdate.Current()
	views := make([]releaseView, 0, len(releases))
	for _, r := range releases {
		view := releaseView{
			Tag:        r.TagName,
			Name:       r.Name,
			Prerelease: r.Prerelease,
			URL:        r.HTMLURL,
			Notes:      r.Body,
			IsCurrent:  appupdate.CompareVersions(r.TagName, current) == 0 && appupdate.IsKnown(),
			IsNewer:    appupdate.IsNewer(r.TagName, current),
		}
		if !r.PublishedAt.IsZero() {
			view.PublishedAt = r.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		views = append(views, view)
	}

	respOK(ctx, c, map[string]any{
		"current_version": current,
		"current_known":   appupdate.IsKnown(),
		"releases":        views,
		"rate_limit":      h.updater.RateLimit(),
	})
}

// StartVersionUpdate downloads and installs a release, then restarts the
// process. An empty version means "latest".
//
// POST /version/update  {"version": "v1.2.3"}
func (h *Handler) StartVersionUpdate(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Version string `json:"version"`
	}

	var req Request
	// An empty body is valid ("update to latest"), so a bind failure is only
	// fatal when the client actually sent something.
	if len(c.Request.Body()) > 0 {
		if err := c.Bind(&req); err != nil {
			respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	task, err := h.updater.StartUpdate(strings.TrimSpace(req.Version))
	if err != nil {
		respErr(ctx, c, CodeOperationFailed, err.Error())
		return
	}

	respOK(ctx, c, task.Snapshot())
}

// PrepareVersionPackage downloads and verifies the OpenWrt .ipk matching this
// install's architecture, without installing it. An empty version means
// "latest".
//
// Admin-only and deliberately stopping short of the install: opkg's prerm
// stops this very service, so an install driven from inside this process would
// kill itself mid-transaction. The response carries the exact command to run.
//
// POST /version/prepare-package  {"version": "v1.2.5"}
func (h *Handler) PrepareVersionPackage(ctx context.Context, c *app.RequestContext) {
	type Request struct {
		Version string `json:"version"`
	}

	var req Request
	// An empty body is valid ("prepare latest").
	if len(c.Request.Body()) > 0 {
		if err := c.Bind(&req); err != nil {
			respErr(ctx, c, CodeBadRequest, "invalid request body: "+err.Error())
			return
		}
	}

	task, err := h.updater.StartIpkPrepare(strings.TrimSpace(req.Version))
	if err != nil {
		respErr(ctx, c, CodeOperationFailed, err.Error())
		return
	}

	respOK(ctx, c, task.Snapshot())
}

// GetVersionUpdateTask reports progress for a running or finished update.
//
// GET /version/task/:task_id
func (h *Handler) GetVersionUpdateTask(ctx context.Context, c *app.RequestContext) {
	taskID := c.Param("task_id")
	if taskID == "" {
		respErr(ctx, c, CodeBadRequest, "task_id is required")
		return
	}

	task, err := h.updater.GetTask(taskID)
	if err != nil {
		respErr(ctx, c, CodeNotFound, err.Error())
		return
	}

	respOK(ctx, c, task.Snapshot())
}

// parseBoolQuery reads a truthy query parameter ("1", "true", "yes").
func parseBoolQuery(c *app.RequestContext, key string) bool {
	switch strings.ToLower(strings.TrimSpace(c.Query(key))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
