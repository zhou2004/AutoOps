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

type APIEndpointController struct {
	apiEndpointService service.APIEndpointServiceInterface
}

func NewAPIEndpointController() *APIEndpointController {
	return &APIEndpointController{
		apiEndpointService: service.NewAPIEndpointService(),
	}
}

func (c *APIEndpointController) GetList(ctx *gin.Context) {
	var req model.APIEndpointListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误")
		return
	}
	c.apiEndpointService.GetList(ctx, &req)
}

func (c *APIEndpointController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.apiEndpointService.GetByID(ctx, uint(id))
}

func (c *APIEndpointController) Add(ctx *gin.Context) {
	var req model.APIEndpointAddReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.apiEndpointService.Add(ctx, &req)
}

func (c *APIEndpointController) Update(ctx *gin.Context) {
	var req model.APIEndpointUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	c.apiEndpointService.Update(ctx, &req)
}

func (c *APIEndpointController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.apiEndpointService.Delete(ctx, uint(id))
}

func (c *APIEndpointController) BatchDelete(ctx *gin.Context) {
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
	c.apiEndpointService.BatchDelete(ctx, ids)
}

func (c *APIEndpointController) CheckEndpoint(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的ID")
		return
	}
	c.apiEndpointService.CheckEndpoint(ctx, uint(id))
}

func (c *APIEndpointController) CheckAllEndpoints(ctx *gin.Context) {
	c.apiEndpointService.CheckAllEndpoints(ctx)
}