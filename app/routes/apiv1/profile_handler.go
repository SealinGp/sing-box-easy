package apiv1

import (
	"context"
	"github.com/SealinGp/sing-box-easy/app/pkg/clientprofile"
	"github.com/SealinGp/sing-box-easy/app/pkg/deploymentprofile"
	"github.com/cloudwego/hertz/pkg/app"
)

// PreviewDeploymentProfile only generates a draft. Saving follows the existing
// full-config review/validation/versioning API, so managed policy guards apply.
func (h *Handler) PreviewDeploymentProfile(ctx context.Context, c *app.RequestContext) {
	var request deploymentprofile.Request
	if err := c.Bind(&request); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid profile request")
		return
	}
	result, err := deploymentprofile.Generate(ctx, request)
	if err != nil {
		respondOperationError(ctx, c, err)
		return
	}
	respOK(ctx, c, result)
}

// PreviewClientProfile converts a supplied listener without changing configuration.
func (h *Handler) PreviewClientProfile(ctx context.Context, c *app.RequestContext) {
	var request clientprofile.Request
	if err := c.Bind(&request); err != nil {
		respErr(ctx, c, CodeBadRequest, "invalid client profile request")
		return
	}
	result, err := clientprofile.Generate(request)
	if err != nil {
		respErr(ctx, c, CodeBadRequest, err.Error())
		return
	}
	respOK(ctx, c, result)
}
