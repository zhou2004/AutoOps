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

type K8sUserGroupController struct {
	service service.IK8sUserGroupService
}

func NewK8sUserGroupController(db *gorm.DB) *K8sUserGroupController {
	return &K8sUserGroupController{
		service: service.NewK8sUserGroupService(db),
	}
}

// Create 创建用户组
func (ctrl *K8sUserGroupController) Create(c *gin.Context) {
	var req model.CreateUserGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Create(c, &req)
}

// Update 更新用户组
func (ctrl *K8sUserGroupController) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	var req model.UpdateUserGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.Update(c, uint(id), &req)
}

// Delete 删除用户组
func (ctrl *K8sUserGroupController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.service.Delete(c, uint(id))
}

// GetList 获取用户组列表
func (ctrl *K8sUserGroupController) GetList(c *gin.Context) {
	var query model.UserGroupQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.GetList(c, query)
}

// GetMembers 获取组成员
func (ctrl *K8sUserGroupController) GetMembers(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	ctrl.service.GetMembers(c, uint(id))
}

// AddMembers 添加组成员
func (ctrl *K8sUserGroupController) AddMembers(c *gin.Context) {
	var req model.AddGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.AddMembers(c, &req)
}

// RemoveMember 移除组成员
func (ctrl *K8sUserGroupController) RemoveMember(c *gin.Context) {
	var req model.RemoveGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数验证失败: "+err.Error())
		return
	}
	ctrl.service.RemoveMember(c, &req)
}

// GetUserGroups 获取用户所属用户组
func (ctrl *K8sUserGroupController) GetUserGroups(c *gin.Context) {
	idStr := c.Param("userId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的用户ID")
		return
	}
	ctrl.service.GetUserGroups(c, uint(id))
}