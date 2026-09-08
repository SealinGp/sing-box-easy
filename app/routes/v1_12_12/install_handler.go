package v1_13_0

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/installation"
	"github.com/cloudwego/hertz/pkg/app"
)

func (h *Handler) InstallSingBox(ctx context.Context, c *app.RequestContext) {
	var req installer.InstallSingBoxCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.installation().InstallSingBox(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetInstallTask(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().GetInstallTask(ctx, c.Param("task_id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetInstallStatus(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().GetInstallStatus(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UpdateSingBox(ctx context.Context, c *app.RequestContext) {
	var req installer.UpdateSingBoxCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.installation().UpdateSingBox(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) DownloadDashboard(ctx context.Context, c *app.RequestContext) {
	var req installer.DownloadDashboardCommand
	if err := c.Bind(&req); err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	result, err := h.installation().DownloadDashboard(ctx, req)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDashboardTask(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().GetDashboardTask(ctx, c.Param("task_id"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) GetDashboardStatus(ctx context.Context, c *app.RequestContext) {
	result, err := h.installation().GetDashboardStatus(ctx)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
func (h *Handler) UploadDashboard(ctx context.Context, c *app.RequestContext) {
	form, err := c.MultipartForm()
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		respErr(ctx, c, CodeBadRequest, "no file uploaded")
		return
	}
	input, err := files[0].Open()
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	defer input.Close()
	result, err := h.installation().UploadDashboard(input, c.PostForm("target_dir"), c.PostForm("folder_name"))
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}
