package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/k8s/model"
	"dodevops-api/api/k8s/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type K8sPermissionController struct {
	service service.IK8sPermissionService
}

func NewK8sPermissionController(db *gorm.DB) *K8sPermissionController {
	return &K8sPermissionController{
		service: service.NewK8sPermissionService(db),
	}
}

// Create 创建K8s权限
// @Summary 创建K8s权限
// @Description 为用户授予指定集群命名空间的访问权限
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param permission body model.CreateK8sPermissionRequest true "权限参数"
// @Success 200 {object} result.Result{data=model.K8sPermission} "创建成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission [post]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) Create(c *gin.Context) {
	var req model.CreateK8sPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Create(c, &req)
}

// BatchCreate 批量创建K8s权限
// @Summary 批量创建K8s权限
// @Description 为用户批量授予指定集群多个命名空间的访问权限
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param permission body model.K8sPermissionBatchCreateRequest true "批量权限参数"
// @Success 200 {object} result.Result{data=[]model.K8sPermission} "创建成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission/batch [post]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) BatchCreate(c *gin.Context) {
	var req model.K8sPermissionBatchCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.BatchCreate(c, &req)
}

// Update 更新K8s权限
// @Summary 更新K8s权限
// @Description 更新指定权限记录的权限类型
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param id path uint true "权限ID"
// @Param permission body model.UpdateK8sPermissionRequest true "更新参数"
// @Success 200 {object} result.Result "更新成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission/{id} [put]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	var req model.UpdateK8sPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Update(c, uint(id), &req)
}

// Delete 删除K8s权限
// @Summary 删除K8s权限
// @Description 删除指定权限记录
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param id path uint true "权限ID"
// @Success 200 {object} result.Result "删除成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission/{id} [delete]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的权限ID")
		return
	}
	ctrl.service.Delete(c, uint(id))
}

// GetList 获取K8s权限列表
// @Summary 获取K8s权限列表
// @Description 分页查询K8s权限列表
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param userId query int false "用户ID"
// @Param clusterId query int false "集群ID"
// @Param namespace query string false "命名空间"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} result.Result{data=model.K8sPermissionListResponse} "查询成功"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission [get]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) GetList(c *gin.Context) {
	var query model.K8sPermissionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.GetList(c, query)
}

// GetUserPermissions 获取用户的所有权限
// @Summary 获取用户的所有K8s权限
// @Description 获取指定用户的所有K8s集群命名空间权限
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param userId path uint true "用户ID"
// @Success 200 {object} result.Result{data=[]model.K8sPermission} "查询成功"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission/user/{userId} [get]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) GetUserPermissions(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的用户ID")
		return
	}
	ctrl.service.GetUserPermissions(c, uint(userID))
}

// GetClusterPermissions 获取集群的所有权限分配
// @Summary 获取集群的所有权限分配
// @Description 获取指定集群的所有用户权限分配
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Param clusterId path uint true "集群ID"
// @Success 200 {object} result.Result{data=[]model.K8sPermission} "查询成功"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/permission/cluster/{clusterId} [get]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) GetClusterPermissions(c *gin.Context) {
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的集群ID")
		return
	}
	ctrl.service.GetClusterPermissions(c, uint(clusterID))
}

// GetMyPermissions 获取当前用户的所有权限
// @Summary 获取当前用户的K8s权限
// @Description 获取当前登录用户的所有K8s集群命名空间权限（含用户组继承）
// @Tags K8s权限管理
// @Accept json
// @Produce json
// @Success 200 {object} result.Result{data=[]service.MyPermissionItem} "查询成功"
// @Router /api/v1/k8s/permission/my [get]
// @Security ApiKeyAuth
func (ctrl *K8sPermissionController) GetMyPermissions(c *gin.Context) {
	ctrl.service.GetMyPermissions(c)
}
