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

type K8sGroupPermissionController struct {
	service service.IK8sGroupPermissionService
}

func NewK8sGroupPermissionController(db *gorm.DB) *K8sGroupPermissionController {
	return &K8sGroupPermissionController{
		service: service.NewK8sGroupPermissionService(db),
	}
}

// Create 创建用户组权限
// @Summary 创建用户组权限
// @Description 为用户组授予指定集群命名空间的访问权限
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param permission body model.CreateGroupPermissionRequest true "权限参数"
// @Success 200 {object} result.Result{data=model.K8sGroupPermission} "创建成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission [post]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) Create(c *gin.Context) {
	var req model.CreateGroupPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Create(c, &req)
}

// BatchCreate 批量创建用户组权限
// @Summary 批量创建用户组权限
// @Description 为用户组批量授予指定集群多个命名空间的访问权限
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param permission body model.BatchCreateGroupPermissionRequest true "批量权限参数"
// @Success 200 {object} result.Result{data=[]model.K8sGroupPermission} "创建成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission/batch [post]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) BatchCreate(c *gin.Context) {
	var req model.BatchCreateGroupPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.BatchCreate(c, &req)
}

// Update 更新用户组权限
// @Summary 更新用户组权限
// @Description 更新指定用户组权限记录的权限类型
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param id path uint true "权限ID"
// @Param permission body model.UpdateGroupPermissionRequest true "更新参数"
// @Success 200 {object} result.Result "更新成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission/{id} [put]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的权限ID")
		return
	}

	var req model.UpdateGroupPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Update(c, uint(id), &req)
}

// Delete 删除用户组权限
// @Summary 删除用户组权限
// @Description 删除指定用户组权限记录
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param id path uint true "权限ID"
// @Success 200 {object} result.Result "删除成功"
// @Failure 400 {object} result.Result "参数错误"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission/{id} [delete]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的权限ID")
		return
	}
	ctrl.service.Delete(c, uint(id))
}

// GetList 获取用户组权限列表
// @Summary 获取用户组权限列表
// @Description 分页查询用户组权限列表
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param groupId query int false "用户组ID"
// @Param clusterId query int false "集群ID"
// @Param namespace query string false "命名空间"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} result.Result{data=model.GroupPermissionListResponse} "查询成功"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission [get]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) GetList(c *gin.Context) {
	var query model.GroupPermissionQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.GetList(c, query)
}

// GetGroupPermissions 获取指定用户组的所有权限
// @Summary 获取用户组的所有权限
// @Description 获取指定用户组的所有集群命名空间权限
// @Tags K8s用户组权限管理
// @Accept json
// @Produce json
// @Param groupId path uint true "用户组ID"
// @Success 200 {object} result.Result{data=[]model.K8sGroupPermissionVo} "查询成功"
// @Failure 403 {object} result.Result "无权限"
// @Router /api/v1/k8s/group-permission/group/{groupId} [get]
// @Security ApiKeyAuth
func (ctrl *K8sGroupPermissionController) GetGroupPermissions(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的用户组ID")
		return
	}
	ctrl.service.GetGroupPermissions(c, uint(groupID))
}
