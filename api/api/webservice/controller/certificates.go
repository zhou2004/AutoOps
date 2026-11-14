package controller

import (
	"dodevops-api/api/webservice/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

func (c *Controller) ImportNewCertificate(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	u, ok := user(ctx)
	if !ok {
		return
	}
	var input service.CertificateImportInput
	if !bind(ctx, &input) {
		return
	}
	value, err := c.S.ImportNewCertificate(ctx.Request.Context(), u, input)
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, value)
}
