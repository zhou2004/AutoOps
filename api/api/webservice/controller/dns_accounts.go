package controller

import (
	"dodevops-api/common/result"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (c *Controller) DNSCredentials(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	u, ok := user(ctx)
	if !ok {
		return
	}
	rows, err := c.S.DNSCredentials(ctx.Request.Context(), u, ctx.DefaultQuery("provider", "cloudflare"))
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, rows)
}

func (c *Controller) DNSAccountZones(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	u, ok := user(ctx)
	if !ok {
		return
	}
	rows, err := c.S.DNSAccountZones(ctx.Request.Context(), u, id(ctx))
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, rows)
}

func (c *Controller) CheckDNSAccount(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	u, ok := user(ctx)
	if !ok {
		return
	}
	var request struct {
		Domain string `json:"domain"`
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 4096)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&request)
	if err != nil && !errors.Is(err, io.EOF) {
		result.Failed(ctx, 400, "DNS 检查参数无效")
		return
	}
	if err == nil && decoder.Decode(new(any)) != io.EOF {
		result.Failed(ctx, 400, "DNS 检查参数无效")
		return
	}
	checked, err := c.S.CheckDNSAccount(ctx.Request.Context(), u, id(ctx), request.Domain)
	if err != nil {
		fail(ctx, err)
		return
	}
	result.Success(ctx, checked)
}
