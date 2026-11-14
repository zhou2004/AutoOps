package controller

import (
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func (c *Controller) SourceDiscovery(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var input service.SourceDiscoveryInput
	if !bind(ctx, &input) {
		return
	}
	value, err := c.S.DiscoverSources(ctx.Request.Context(), u, input)
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, value)
}
func (c *Controller) Topology(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	value, err := c.S.Topology(ctx.Request.Context(), u, id(ctx))
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, value)
}
