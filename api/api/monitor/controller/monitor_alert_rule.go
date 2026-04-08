package controller

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/api/monitor/service"
	"dodevops-api/common/result"
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MonitorAlertRuleController struct {
	ruleService service.MonitorAlertRuleService
}

// AlertRuleReq 接收前端带有对象形式 labels 的结构体
type AlertRuleReq struct {
	ID           uint                   `json:"id"`
	DataSourceID uint                   `json:"dataSourceId"`
	Name         string                 `json:"name"`
	Labels       map[string]interface{} `json:"labels"`
	RuleContent  string                 `json:"ruleContent"`
	Status       string                 `json:"status"`
}

func NewMonitorAlertRuleController() *MonitorAlertRuleController {
	return &MonitorAlertRuleController{
		ruleService: service.NewMonitorAlertRuleService(),
	}
}

// Create 创建告警规则并应用到数据源
func (c *MonitorAlertRuleController) Create(ctx *gin.Context) {
	var req AlertRuleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败: "+err.Error())
		return
	}

	if req.DataSourceID == 0 {
		result.Failed(ctx, int(result.ApiCode.FAILED), "请选择数据源")
		return
	}
	if req.RuleContent == "" {
		result.Failed(ctx, int(result.ApiCode.FAILED), "告警规则内容不可为空")
		return
	}

	labelsBytes, _ := json.Marshal(req.Labels)
	data := model.MonitorAlertRule{
		DataSourceID: req.DataSourceID,
		Name:         req.Name,
		Labels:       string(labelsBytes),
		RuleContent:  req.RuleContent,
		Status:       req.Status,
	}

	if err := c.ruleService.Create(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}
	result.Success(ctx, data)
}

// Delete 删除告警规则并从数据源移除
func (c *MonitorAlertRuleController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "无效的ID")
		return
	}

	if err := c.ruleService.Delete(uint(id)); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}
	result.Success(ctx, nil)
}

// Update 更新告警规则并重新应用到数据源
func (c *MonitorAlertRuleController) Update(ctx *gin.Context) {
	var req AlertRuleReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数绑定失败: "+err.Error())
		return
	}

	if req.ID == 0 {
		result.Failed(ctx, int(result.ApiCode.FAILED), "告警规则ID不能为空")
		return
	}

	labelsBytes, _ := json.Marshal(req.Labels)
	data := model.MonitorAlertRule{
		ID:           req.ID,
		DataSourceID: req.DataSourceID,
		Name:         req.Name,
		Labels:       string(labelsBytes),
		RuleContent:  req.RuleContent,
		Status:       req.Status,
	}

	if err := c.ruleService.Update(&data); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}
	result.Success(ctx, data)
}

// GetByID 获取单个告警规则
func (c *MonitorAlertRuleController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "无效的ID")
		return
	}

	data, err := c.ruleService.GetByID(uint(id))
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "获取告警规则失败")
		return
	}
	result.Success(ctx, data)
}

// GetList 获取告警规则列表
func (c *MonitorAlertRuleController) GetList(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	list, total, err := c.ruleService.GetList(page, pageSize)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "获取告警规则列表失败")
		return
	}
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}
