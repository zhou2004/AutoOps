package controller

import (
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func (c *Controller) TrafficPolicy(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	out, err := c.S.TrafficPolicy(ctx.Request.Context(), u, id(ctx), queryID(ctx, "instanceId"))
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, out)
}
func (c *Controller) PreviewTrafficPolicy(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var input service.TrafficPolicyInput
	if !bind(ctx, &input) {
		return
	}
	out, err := c.S.PreviewTrafficPolicy(ctx.Request.Context(), u, id(ctx), input)
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, out)
}
func (c *Controller) DraftTrafficPolicy(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	var input service.TrafficPolicyInput
	if !bind(ctx, &input) {
		return
	}
	out, err := c.S.DraftTrafficPolicy(ctx.Request.Context(), u, id(ctx), input)
	if err != nil {
		fail(ctx, err)
		return
	}
	c.S.Redact(u, "configurations", &out.Draft)
	result.Success(ctx, out)
}
