package controller

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/api/monitor/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
	"strconv"
)

type MonitorCategoryController struct {
	service *service.MonitorCategoryService
}

func NewMonitorCategoryController() *MonitorCategoryController {
	return &MonitorCategoryController{service: service.NewMonitorCategoryService()}
}
func (c *MonitorCategoryController) List(ctx *gin.Context) {
	list, err := c.service.List(ctx.Query("scope"))
	if err != nil {
		result.Failed(ctx, 400, err.Error())
		return
	}
	result.Success(ctx, list)
}
func (c *MonitorCategoryController) Save(ctx *gin.Context) {
	var req model.MonitorCategoryReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, 400, "分类参数错误")
		return
	}
	if ctx.Request.Method == "PUT" && req.ID == 0 {
		result.Failed(ctx, 400, "分类ID不能为空")
		return
	}
	if ctx.Request.Method == "POST" {
		req.ID = 0
	}
	if err := c.service.Save(&req); err != nil {
		result.Failed(ctx, 400, err.Error())
		return
	}
	result.Success(ctx, nil)
}
func (c *MonitorCategoryController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(ctx, 400, "无效的分类ID")
		return
	}
	if err := c.service.Delete(ctx.Query("scope"), uint(id)); err != nil {
		result.Failed(ctx, 400, err.Error())
		return
	}
	result.Success(ctx, nil)
}
func (c *MonitorCategoryController) Assign(ctx *gin.Context) {
	var req model.MonitorCategoryAssignReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, 400, "请选择监控目标和有效分类")
		return
	}
	if err := c.service.Assign(&req); err != nil {
		result.Failed(ctx, 400, err.Error())
		return
	}
	result.Success(ctx, nil)
}
