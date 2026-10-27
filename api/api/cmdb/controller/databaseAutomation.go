package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DatabaseAutomationController struct {
	db      *gorm.DB
	service *service.DatabaseAutomationService
}

func NewDatabaseAutomationController(db *gorm.DB) *DatabaseAutomationController {
	return &DatabaseAutomationController{db: db, service: service.NewDatabaseAutomationService(db)}
}
func (ctrl *DatabaseAutomationController) List(c *gin.Context) {
	var rows []model.DatabaseDeployment
	if err := ctrl.db.Order("id DESC").Limit(200).Find(&rows).Error; err != nil {
		result.Failed(c, 500, "无法读取部署记录")
		return
	}
	result.Success(c, gin.H{"list": rows})
}
func (ctrl *DatabaseAutomationController) Create(c *gin.Context) {
	var req service.DatabaseDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "部署参数无效")
		return
	}
	user, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, 401, "请先登录")
		return
	}
	d, run, err := ctrl.service.Create(req, user)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, gin.H{"deployment": d, "run": run})
}
func deploymentID(c *gin.Context) (uint, bool) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 32)
	if e != nil || id == 0 {
		result.Failed(c, 400, "部署 ID 无效")
		return 0, false
	}
	return uint(id), true
}
func (ctrl *DatabaseAutomationController) Start(c *gin.Context) {
	id, ok := deploymentID(c)
	if !ok {
		return
	}
	var req struct {
		Action      string `json:"action"`
		ConfirmName string `json:"confirmName"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "操作参数无效")
		return
	}
	user, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, 401, "请先登录")
		return
	}
	run, err := ctrl.service.Start(id, user, req.Action, req.ConfirmName)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, run)
}
func (ctrl *DatabaseAutomationController) Runs(c *gin.Context) {
	id, ok := deploymentID(c)
	if !ok {
		return
	}
	var runs []model.DatabaseAutomationRun
	if err := ctrl.db.Where("deployment_id = ?", id).Order("id DESC").Limit(30).Find(&runs).Error; err != nil {
		result.Failed(c, 500, "执行历史读取失败")
		return
	}
	result.Success(c, gin.H{"list": runs})
}
func (ctrl *DatabaseAutomationController) HAView(c *gin.Context) {
	id, ok := deploymentID(c)
	if !ok {
		return
	}
	value, err := ctrl.service.HAView(c.Request.Context(), id, c.Param("view"))
	if err != nil {
		result.Failed(c, http.StatusBadGateway, err.Error())
		return
	}
	result.Success(c, value)
}
func (ctrl *DatabaseAutomationController) HAAction(c *gin.Context) {
	id, ok := deploymentID(c)
	if !ok {
		return
	}
	var req service.DatabaseHARequest
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, 400, "高可用操作参数无效")
		return
	}
	user, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, 401, "请先登录")
		return
	}
	value, err := ctrl.service.HAAction(c.Request.Context(), id, user, req)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, value)
}

// Access is an explicit administrator-only credential reveal. Passwords never
// appear in deployment lists, execution logs, URLs, or audit records.
func (ctrl *DatabaseAutomationController) Access(c *gin.Context) {
	id, ok := deploymentID(c)
	if !ok {
		return
	}
	user, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, 401, "请先登录")
		return
	}
	value, err := ctrl.service.OrchestratorAccess(id, user)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, value)
}
