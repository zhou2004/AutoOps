package controller

import (
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Traffic(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	data, err := c.S.Traffic(ctx.Request.Context(), u, service.TrafficFilter{Range: ctx.DefaultQuery("range", "1h"), GroupID: queryID(ctx, "groupId"), ClusterID: queryID(ctx, "clusterId"), InstanceID: queryID(ctx, "instanceId")})
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, data)
}
