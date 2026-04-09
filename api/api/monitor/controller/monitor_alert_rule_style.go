package controller

import (
"dodevops-api/api/monitor/model"
"dodevops-api/api/monitor/service"
"dodevops-api/common/result"
"github.com/gin-gonic/gin"
"strconv"
)

type MonitorAlertRuleStyleController struct {
styleService service.MonitorAlertRuleStyleService
}

func NewMonitorAlertRuleStyleController() *MonitorAlertRuleStyleController {
return &MonitorAlertRuleStyleController{styleService: service.NewMonitorAlertRuleStyleService()}
}

func (c *MonitorAlertRuleStyleController) CreateStyle(ctx *gin.Context) {
var data model.MonitorAlertRuleStyle
if err := ctx.ShouldBindJSON(&data); err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败: "+err.Error())
return
}
if err := c.styleService.CreateStyle(&data); err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "创建失败: "+err.Error())
return
}
result.Success(ctx, data)
}

func (c *MonitorAlertRuleStyleController) UpdateStyle(ctx *gin.Context) {
var data model.MonitorAlertRuleStyle
if err := ctx.ShouldBindJSON(&data); err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败: "+err.Error())
return
}
if err := c.styleService.UpdateStyle(&data); err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "更新失败: "+err.Error())
return
}
result.Success(ctx, data)
}

func (c *MonitorAlertRuleStyleController) DeleteStyle(ctx *gin.Context) {
idStr := ctx.Param("id")
id, _ := strconv.Atoi(idStr)
if err := c.styleService.DeleteStyle(uint(id)); err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "删除失败: "+err.Error())
return
}
result.Success(ctx, nil)
}

func (c *MonitorAlertRuleStyleController) GetStyleList(ctx *gin.Context) {
list, err := c.styleService.GetStyleList()
if err != nil {
result.Failed(ctx, int(result.ApiCode.FAILED), "获取失败: "+err.Error())
return
}
result.Success(ctx, list)
}
