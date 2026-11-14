package controller

import (
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func (c *Controller) Environments(ctx *gin.Context) {
	u, ok := user(ctx)
	if !ok {
		return
	}
	values, err := c.S.Environments(ctx.Request.Context(), u)
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, values)
}
