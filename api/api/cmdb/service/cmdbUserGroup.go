package service

import (
	"net/http"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

type ICmdbUserGroupService interface {
	Create(c *gin.Context, req *model.CmdbCreateUserGroupReq)
	Update(c *gin.Context, id uint, req *model.CmdbUpdateUserGroupReq)
	Delete(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbUserGroupQuery)
	GetMembers(c *gin.Context, groupID uint)
	AddMembers(c *gin.Context, req *model.CmdbAddGroupMemberReq)
	RemoveMember(c *gin.Context, req *model.CmdbRemoveGroupMemberReq)
	GetUserGroups(c *gin.Context, userID uint)
	GetAll(c *gin.Context)
}

type CmdbUserGroupServiceImpl struct {
	groupDao *dao.CmdbUserGroupDao
}

func NewCmdbUserGroupService() ICmdbUserGroupService {
	return &CmdbUserGroupServiceImpl{
		groupDao: dao.NewCmdbUserGroupDao(),
	}
}

func (s *CmdbUserGroupServiceImpl) checkAdmin(c *gin.Context) bool {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		return false
	}
	// 从数据库获取真实DB
	db := s.groupDao
	if db == nil {
		return false
	}
	// 使用系统角色检查
	return s.isAdminUser(admin.ID)
}

func (s *CmdbUserGroupServiceImpl) isAdminUser(userID uint) bool {
	// 纯数据库查询的方式
	return false // 暂时由前端控制
}

func (s *CmdbUserGroupServiceImpl) Create(c *gin.Context, req *model.CmdbCreateUserGroupReq) {
	group := &model.CmdbUserGroup{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Status:      1,
	}
	if err := s.groupDao.Create(group); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建用户组失败: "+err.Error())
		return
	}
	result.Success(c, group)
}

func (s *CmdbUserGroupServiceImpl) Update(c *gin.Context, id uint, req *model.CmdbUpdateUserGroupReq) {
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

func (s *CmdbUserGroupServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.groupDao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除用户组失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbUserGroupServiceImpl) GetList(c *gin.Context, query model.CmdbUserGroupQuery) {
	list, total, err := s.groupDao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组列表失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbUserGroupServiceImpl) GetMembers(c *gin.Context, groupID uint) {
	members, err := s.groupDao.GetMembers(groupID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询组成员失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": members})
}

func (s *CmdbUserGroupServiceImpl) AddMembers(c *gin.Context, req *model.CmdbAddGroupMemberReq) {
	if err := s.groupDao.AddMembers(req.GroupID, req.UserIDs); err != nil {
		result.Failed(c, http.StatusInternalServerError, "添加组成员失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbUserGroupServiceImpl) RemoveMember(c *gin.Context, req *model.CmdbRemoveGroupMemberReq) {
	if err := s.groupDao.RemoveMember(req.GroupID, req.UserID); err != nil {
		result.Failed(c, http.StatusInternalServerError, "移除组成员失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbUserGroupServiceImpl) GetUserGroups(c *gin.Context, userID uint) {
	groups, err := s.groupDao.GetUserGroups(userID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": groups})
}

func (s *CmdbUserGroupServiceImpl) GetAll(c *gin.Context) {
	list, err := s.groupDao.GetAll()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": len(list)})
}

// ======================= 凭据授权 =======================

type ICmdbCredentialPermissionService interface {
	Create(c *gin.Context, req *model.CmdbCredentialPermissionReq)
	Update(c *gin.Context, id uint, req *model.CmdbCredentialPermissionReq)
	Delete(c *gin.Context, id uint)
	GetByID(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.CmdbCredentialPermissionQuery)
	GetMyCredentials(c *gin.Context)
}

type CmdbCredentialPermissionServiceImpl struct {
	dao *dao.CmdbCredentialPermissionDao
}

func NewCmdbCredentialPermissionService() ICmdbCredentialPermissionService {
	return &CmdbCredentialPermissionServiceImpl{
		dao: dao.NewCmdbCredentialPermissionDao(),
	}
}

func (s *CmdbCredentialPermissionServiceImpl) Create(c *gin.Context, req *model.CmdbCredentialPermissionReq) {
	data := &model.CmdbCredentialPermission{
		Name:         req.Name,
		CredentialID: req.CredentialID,
		UserIDs:      toJSONStr(req.UserIDs),
		GroupIDs:     toJSONStr(req.GroupIDs),
		IsActive:     req.IsActive,
	}
	if data.IsActive == 0 {
		data.IsActive = 1
	}
	if err := s.dao.Create(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建凭据授权失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbCredentialPermissionServiceImpl) Update(c *gin.Context, id uint, req *model.CmdbCredentialPermissionReq) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "凭据授权不存在")
		return
	}
	data.Name = req.Name
	data.CredentialID = req.CredentialID
	data.UserIDs = toJSONStr(req.UserIDs)
	data.GroupIDs = toJSONStr(req.GroupIDs)
	data.IsActive = req.IsActive
	if err := s.dao.Update(data); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新凭据授权失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

func (s *CmdbCredentialPermissionServiceImpl) Delete(c *gin.Context, id uint) {
	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除凭据授权失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *CmdbCredentialPermissionServiceImpl) GetByID(c *gin.Context, id uint) {
	data, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "凭据授权不存在")
		return
	}
	result.Success(c, data)
}

func (s *CmdbCredentialPermissionServiceImpl) GetList(c *gin.Context, query model.CmdbCredentialPermissionQuery) {
	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询凭据授权失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (s *CmdbCredentialPermissionServiceImpl) GetMyCredentials(c *gin.Context) {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "未授权")
		return
	}
	groupDao := dao.NewCmdbUserGroupDao()
	userGroupIDs, _ := groupDao.GetUserGroupIDs(admin.ID)
	creds, err := s.dao.GetUserCredentials(admin.ID, userGroupIDs)
	if err != nil {
		result.Success(c, gin.H{"list": []model.CmdbCredentialPermission{}})
		return
	}
	result.Success(c, gin.H{"list": creds})
}
