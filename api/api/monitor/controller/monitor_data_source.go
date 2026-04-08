package controller

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/api/monitor/service"
	"dodevops-api/common/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MonitorDataSourceController struct {
	dataSourceService service.MonitorDataSourceService
}

func NewMonitorDataSourceController() *MonitorDataSourceController {
	return &MonitorDataSourceController{
		dataSourceService: service.NewMonitorDataSourceService(),
	}
}

// Create 创建监控数据源
func (c *MonitorDataSourceController) Create(ctx *gin.Context) {
	var data model.MonitorDataSource
	if err := ctx.ShouldBindJSON(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败")
		return
	}

	if err := c.dataSourceService.Create(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "创建监控数据源失败")
		return
	}
	result.Success(ctx, data)
}

// Delete 删除监控数据源
func (c *MonitorDataSourceController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "无效的ID")
		return
	}

	if err := c.dataSourceService.Delete(uint(id)); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "删除监控数据源失败")
		return
	}
	result.Success(ctx, nil)
}

// Update 更新监控数据源
func (c *MonitorDataSourceController) Update(ctx *gin.Context) {
	var data model.MonitorDataSource
	if err := ctx.ShouldBindJSON(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败")
		return
	}

	if data.ID == 0 {
		result.Failed(ctx, int(result.ApiCode.FAILED), "监控数据源ID不能为空")
		return
	}

	if err := c.dataSourceService.Update(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "更新监控数据源失败")
		return
	}
	result.Success(ctx, data)
}

// GetByID 获取单个监控数据源
func (c *MonitorDataSourceController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "无效的ID")
		return
	}

	data, err := c.dataSourceService.GetByID(uint(id))
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "获取监控数据源失败")
		return
	}
	result.Success(ctx, data)
}

// GetList 获取监控数据源列表
func (c *MonitorDataSourceController) GetList(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	list, total, err := c.dataSourceService.GetList(page, pageSize)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "获取监控数据源列表失败")
		return
	}
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}
