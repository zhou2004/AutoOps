package service

import (
	"net/http"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IK8sGroupPermissionService 用户组权限服务接口
type IK8sGroupPermissionService interface {
	Create(c *gin.Context, req *model.CreateGroupPermissionRequest)
	BatchCreate(c *gin.Context, req *model.BatchCreateGroupPermissionRequest)
	Update(c *gin.Context, id uint, req *model.UpdateGroupPermissionRequest)
	Delete(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.GroupPermissionQuery)
	GetGroupPermissions(c *gin.Context, groupID uint)
}

// K8sGroupPermissionServiceImpl 用户组权限服务实现
type K8sGroupPermissionServiceImpl struct {
	groupPermDao *dao.K8sGroupPermissionDao
	permDao      *dao.K8sPermissionDao
}

func NewK8sGroupPermissionService(db *gorm.DB) IK8sGroupPermissionService {
	return &K8sGroupPermissionServiceImpl{
		groupPermDao: dao.NewK8sGroupPermissionDao(db),
		permDao:      dao.NewK8sPermissionDao(db),
	}
}

// checkAdmin 检查当前用户是否为管理员
func (s *K8sGroupPermissionServiceImpl) checkAdmin(c *gin.Context) bool {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		return false
	}
	isAdmin, _ := s.permDao.IsAdmin(admin.ID)
	return isAdmin
}

// Create 创建用户组权限
func (s *K8sGroupPermissionServiceImpl) Create(c *gin.Context, req *model.CreateGroupPermissionRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	perm := model.K8sGroupPermission{
		GroupID:        req.GroupID,
		ClusterID:      req.ClusterID,
		Namespace:      req.Namespace,
		PermissionType: req.PermissionType,
	}

	if err := s.groupPermDao.Create(&perm); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建用户组权限失败: "+err.Error())
		return
	}

	result.Success(c, perm)
}

// BatchCreate 批量创建用户组权限
func (s *K8sGroupPermissionServiceImpl) BatchCreate(c *gin.Context, req *model.BatchCreateGroupPermissionRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	var perms []model.K8sGroupPermission
	for _, ns := range req.Namespaces {
		perms = append(perms, model.K8sGroupPermission{
			GroupID:        req.GroupID,
			ClusterID:      req.ClusterID,
			Namespace:      ns,
			PermissionType: req.PermissionType,
		})
	}

	if err := s.groupPermDao.BatchCreate(perms); err != nil {
		result.Failed(c, http.StatusInternalServerError, "批量创建用户组权限失败: "+err.Error())
		return
	}

	result.Success(c, perms)
}

// Update 更新用户组权限
func (s *K8sGroupPermissionServiceImpl) Update(c *gin.Context, id uint, req *model.UpdateGroupPermissionRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	if err := s.groupPermDao.Update(id, req.PermissionType); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新用户组权限失败: "+err.Error())
		return
	}

	result.Success(c, nil)
}

// Delete 删除用户组权限
func (s *K8sGroupPermissionServiceImpl) Delete(c *gin.Context, id uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作")
		return
	}

	if err := s.groupPermDao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除用户组权限失败: "+err.Error())
		return
	}

	result.Success(c, nil)
}

// GetList 获取用户组权限列表（分页）
func (s *K8sGroupPermissionServiceImpl) GetList(c *gin.Context, query model.GroupPermissionQuery) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看")
		return
	}

	list, total, err := s.groupPermDao.GetGroupPermissionList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组权限列表失败: "+err.Error())
		return
	}

	result.Success(c, model.GroupPermissionListResponse{
		List:  list,
		Total: total,
	})
}

// GetGroupPermissions 获取指定用户组的所有权限
func (s *K8sGroupPermissionServiceImpl) GetGroupPermissions(c *gin.Context, groupID uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看")
		return
	}

	perms, err := s.groupPermDao.GetByGroup(groupID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户组权限失败: "+err.Error())
		return
	}

	result.Success(c, perms)
}
