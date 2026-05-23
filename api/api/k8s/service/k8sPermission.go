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

// IK8sPermissionService K8s权限服务接口
type IK8sPermissionService interface {
	Create(c *gin.Context, req *model.CreateK8sPermissionRequest)
	BatchCreate(c *gin.Context, req *model.K8sPermissionBatchCreateRequest)
	Update(c *gin.Context, id uint, req *model.UpdateK8sPermissionRequest)
	Delete(c *gin.Context, id uint)
	GetList(c *gin.Context, query model.K8sPermissionQuery)
	GetUserPermissions(c *gin.Context, userID uint)
	GetClusterPermissions(c *gin.Context, clusterID uint)
}

// K8sPermissionServiceImpl K8s权限服务实现
type K8sPermissionServiceImpl struct {
	dao *dao.K8sPermissionDao
}

func NewK8sPermissionService(db *gorm.DB) IK8sPermissionService {
	return &K8sPermissionServiceImpl{
		dao: dao.NewK8sPermissionDao(db),
	}
}

// checkAdmin 检查当前用户是否为管理员
func (s *K8sPermissionServiceImpl) checkAdmin(c *gin.Context) bool {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		return false
	}
	isAdmin, _ := s.dao.IsAdmin(admin.ID)
	return isAdmin
}

// Create 创建权限
func (s *K8sPermissionServiceImpl) Create(c *gin.Context, req *model.CreateK8sPermissionRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作K8s权限")
		return
	}

	perm := model.K8sPermission{
		UserID:         req.UserID,
		ClusterID:      req.ClusterID,
		Namespace:      req.Namespace,
		PermissionType: req.PermissionType,
	}

	if err := s.dao.Create(&perm); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建权限失败: "+err.Error())
		return
	}

	result.Success(c, perm)
}

// BatchCreate 批量创建权限
func (s *K8sPermissionServiceImpl) BatchCreate(c *gin.Context, req *model.K8sPermissionBatchCreateRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作K8s权限")
		return
	}

	var perms []model.K8sPermission
	for _, ns := range req.Namespaces {
		perms = append(perms, model.K8sPermission{
			UserID:         req.UserID,
			ClusterID:      req.ClusterID,
			Namespace:      ns,
			PermissionType: req.PermissionType,
		})
	}

	if err := s.dao.BatchCreate(perms); err != nil {
		result.Failed(c, http.StatusInternalServerError, "批量创建权限失败: "+err.Error())
		return
	}

	result.Success(c, perms)
}

// Update 更新权限
func (s *K8sPermissionServiceImpl) Update(c *gin.Context, id uint, req *model.UpdateK8sPermissionRequest) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作K8s权限")
		return
	}

	if err := s.dao.Update(id, req.PermissionType); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新权限失败: "+err.Error())
		return
	}

	result.Success(c, nil)
}

// Delete 删除权限
func (s *K8sPermissionServiceImpl) Delete(c *gin.Context, id uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可操作K8s权限")
		return
	}

	if err := s.dao.Delete(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除权限失败: "+err.Error())
		return
	}

	result.Success(c, nil)
}

// GetList 获取权限列表
func (s *K8sPermissionServiceImpl) GetList(c *gin.Context, query model.K8sPermissionQuery) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看K8s权限列表")
		return
	}

	list, total, err := s.dao.GetList(query)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询权限列表失败: "+err.Error())
		return
	}

	result.Success(c, model.K8sPermissionListResponse{
		List:  list,
		Total: total,
	})
}

// GetUserPermissions 获取用户的所有权限
func (s *K8sPermissionServiceImpl) GetUserPermissions(c *gin.Context, userID uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看用户权限")
		return
	}

	perms, err := s.dao.GetByUser(userID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询用户权限失败: "+err.Error())
		return
	}

	result.Success(c, perms)
}

// GetClusterPermissions 获取集群的所有权限分配
func (s *K8sPermissionServiceImpl) GetClusterPermissions(c *gin.Context, clusterID uint) {
	if !s.checkAdmin(c) {
		result.Failed(c, http.StatusForbidden, "仅管理员可查看集群权限")
		return
	}

	perms, err := s.dao.GetByCluster(clusterID)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询集群权限失败: "+err.Error())
		return
	}

	result.Success(c, perms)
}
