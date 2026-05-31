package service

import (
	"net/http"
	"time"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IK8sUserGroupService interface {
	Create(c *gin.Context, req *model.CreateUserGroupRequest)
	Update(c *gin.Context, id uint, req *model.UpdateUserGroupRequest)
	Delete(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.UserGroupQuery)
	GetMembers(c *gin.Context, groupID uint)
	AddMembers(c *gin.Context, req *model.AddGroupMemberRequest)
	RemoveMember(c *gin.Context, req *model.RemoveGroupMemberRequest)
	GetUserGroups(c *gin.Context, userID uint)
}

type K8sUserGroupServiceImpl struct {
	groupDao     *dao.K8sUserGroupDao
	permDao      *dao.K8sPermissionDao
}

func NewK8sUserGroupService(db *gorm.DB) IK8sUserGroupService {
	return &K8sUserGroupServiceImpl{
		groupDao: dao.NewK8sUserGroupDao(db),
		permDao:  dao.NewK8sPermissionDao(db),
	}
}

func (s *K8sUserGroupServiceImpl) checkAdmin(c *gin.Context) bool {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		return false
	}
	isAdmin, _ := s.permDao.IsAdmin(admin.ID)
	return isAdmin
}

// Create 创建用户组
func (s *K8sUserGroupServiceImpl) Create(c *gin.Context, req *model.CreateUserGroupRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	group := &model.K8sUserGroup{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
	}
	if err := s.groupDao.Create(group); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建用户组失败: "+err.Error())
		return
	}
	result.Success(c, group)
}

// Update 更新用户组
func (s *K8sUserGroupServiceImpl) Update(c *gin.Context, id uint, req *model.UpdateUserGroupRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Code != "" {
		updates["code"] = req.Code
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.groupDao.Update(id, updates); err != nil {
			result.Failed(c, http.StatusInternalServerError, "更新用户组失败: "+err.Error())
			return
		}
	}
	result.Success(c, nil)
}

// Delete 删除用户组
func (s *K8sUserGroupServiceImpl) Delete(c *gin.Context, id uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}
	if err := s.groupDao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除用户组失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

// GetList 获取用户组列表
func (s *K8sUserGroupServiceImpl) GetList(c *gin.Context, query model.UserGroupQuery) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看")
		return
	}
	list, total, err := s.groupDao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组列表失败: "+err.Error())
		return
	}
	result.Success(c, model.UserGroupListResponse{List: list, Total: total})
}

// GetMembers 获取组成员列表
func (s *K8sUserGroupServiceImpl) GetMembers(c *gin.Context, groupID uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看")
		return
	}
	members, err := s.groupDao.GetMembers(groupID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询组成员失败: "+err.Error())
		return
	}
	result.Success(c, members)
}

// AddMembers 添加组成员
func (s *K8sUserGroupServiceImpl) AddMembers(c *gin.Context, req *model.AddGroupMemberRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}
	if err := s.groupDao.AddMembers(req.GroupID, req.UserIDs); err != nil {
		result.Failed(c, http.StatusInternalServerError, "添加组成员失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

// RemoveMember 移除组成员
func (s *K8sUserGroupServiceImpl) RemoveMember(c *gin.Context, req *model.RemoveGroupMemberRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}
	if err := s.groupDao.RemoveMember(req.GroupID, req.UserID); err != nil {
		result.Failed(c, http.StatusInternalServerError, "移除组成员失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

// GetUserGroups 获取用户所属用户组
func (s *K8sUserGroupServiceImpl) GetUserGroups(c *gin.Context, userID uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看")
		return
	}
	ids, err := s.groupDao.GetUserGroupIDs(userID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组失败: "+err.Error())
		return
	}
	result.Success(c, ids)
}