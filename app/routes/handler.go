package routes

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (r *Route) ListNodes(ctx context.Context, c *app.RequestContext) {
	type ReqBody struct {
		Lines string `json:"lines"`
	}

	var reqBody ReqBody
	if err := c.Bind(&reqBody); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}

	nodes, err := r.modules.SubscriptionManager.ImportPreview(ctx, reqBody.Lines)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": err.Error()})
		return
	}

	c.JSON(consts.StatusOK, utils.H{"nodes": nodes})
}
