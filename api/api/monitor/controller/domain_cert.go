package controller

import (
	"net/http"
	"strconv"
	"strings"

	"dodevops-api/api/monitor/model"
	"dodevops-api/api/monitor/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

type DomainCertController struct {
	domainCertService service.DomainCertServiceInterface
}

func NewDomainCertController() *DomainCertController {
	return &DomainCertController{
		domainCertService: service.NewDomainCertService(),
	}
}

func (c *DomainCertController) GetList(ctx *gin.Context) {
	var req model.DomainCertListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	c.domainCertService.GetList(ctx, &req)
}

func (c *DomainCertController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.domainCertService.GetByID(ctx, uint(id))
}

func (c *DomainCertController) Add(ctx *gin.Context) {
	var req model.DomainCertAddReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.domainCertService.Add(ctx, &req)
}

func (c *DomainCertController) Update(ctx *gin.Context) {
	var req model.DomainCertUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.domainCertService.Update(ctx, &req)
}

func (c *DomainCertController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.domainCertService.Delete(ctx, uint(id))
}

func (c *DomainCertController) BatchDelete(ctx *gin.Context) {
	var ids []uint
	if err := ctx.ShouldBindJSON(&ids); err != nil {
		raw := ctx.GetString("ids")
		if raw == "" {
			result.Failed(ctx, http.StatusBadRequest, "参数错误")
			return
		}
		for _, s := range strings.Split(raw, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
			if err != nil {
				result.Failed(ctx, http.StatusBadRequest, "无效的ID格式")
				return
			}
			ids = append(ids, uint(id))
		}
	}
	c.domainCertService.BatchDelete(ctx, ids)
}

func (c *DomainCertController) CheckCert(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.domainCertService.CheckCert(ctx, uint(id))
}

func (c *DomainCertController) CheckAllCerts(ctx *gin.Context) {
	c.domainCertService.CheckAllCerts(ctx)
}