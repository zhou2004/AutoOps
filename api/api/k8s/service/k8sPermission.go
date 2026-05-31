package service

import (
	"encoding/json"
	"fmt"
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
	GetMyPermissions(c *gin.Context)
}

// K8sPermissionServiceImpl K8s权限服务实现
type K8sPermissionServiceImpl struct {
	dao        *dao.K8sPermissionDao
	groupDao   *dao.K8sUserGroupDao
	gPermDao   *dao.K8sGroupPermissionDao
	rbacDao    *dao.K8sRbacDao
	clusterDao *dao.KubeClusterDao
}

func NewK8sPermissionService(db *gorm.DB) IK8sPermissionService {
	return &K8sPermissionServiceImpl{
		dao:        dao.NewK8sPermissionDao(db),
		groupDao:   dao.NewK8sUserGroupDao(db),
		gPermDao:   dao.NewK8sGroupPermissionDao(db),
		rbacDao:    dao.NewK8sRbacDao(db),
		clusterDao: dao.NewKubeClusterDao(db),
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

// MyPermissionItem 当前用户权限项
type MyPermissionItem struct {
	ClusterID      uint            `json:"clusterId"`
	ClusterName    string          `json:"clusterName"`
	Namespace      string          `json:"namespace"`
	PermissionType string          `json:"permissionType"`
	Source         string          `json:"source"` // "direct" / "group" / "rbac"
	Rules          []model.K8sRule `json:"rules,omitempty"`
}

// GetMyPermissions 获取当前用户的权限列表（含用户组继承 + RBAC规则）
func (s *K8sPermissionServiceImpl) GetMyPermissions(c *gin.Context) {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "未授权")
		return
	}

	// 如果是系统管理员，返回所有权限
	isAdmin, _ := s.dao.IsAdmin(admin.ID)
	if isAdmin {
		result.Success(c, []MyPermissionItem{
			{ClusterID: 0, Namespace: "*", PermissionType: "admin", Source: "system"},
		})
		return
	}

	var items []MyPermissionItem

	// 构建集群名称映射（从数据库查询所有集群）
	clusterNameMap := make(map[uint]string)
	allClusters, _ := s.clusterDao.GetAll()
	for _, cluster := range allClusters {
		clusterNameMap[cluster.ID] = cluster.Name
	}

	// 1. 获取直接授权
	perms, err := s.dao.GetByUser(admin.ID)
	if err == nil {
		for _, p := range perms {
			cn := clusterNameMap[p.ClusterID]
			items = append(items, MyPermissionItem{
				ClusterID:      p.ClusterID,
				ClusterName:    cn,
				Namespace:      p.Namespace,
				PermissionType: p.PermissionType,
				Source:         "direct",
			})
		}
	}

	// 2. 获取用户组继承权限
	groupIDs, err := s.groupDao.GetUserGroupIDs(admin.ID)
	if err == nil && len(groupIDs) > 0 {
		for _, gid := range groupIDs {
			gPerms, err := s.gPermDao.GetByGroup(gid)
			if err == nil {
				for _, gp := range gPerms {
					duplicate := false
					for _, item := range items {
						if item.ClusterID == gp.ClusterID && item.Namespace == gp.Namespace {
							duplicate = true
							break
						}
					}
					if !duplicate {
						cn := gp.ClusterName
						if cn == "" {
							cn = clusterNameMap[gp.ClusterID]
						}
						items = append(items, MyPermissionItem{
							ClusterID:      gp.ClusterID,
							ClusterName:    cn,
							Namespace:      gp.Namespace,
							PermissionType: gp.PermissionType,
							Source:         "group",
						})
					}
				}
			}
		}
	}

	// 3. 获取RBAC规则权限
	rbacItems := s.getMyRbacPermissions(admin.ID, clusterNameMap)
	// 合并RBAC items（去重）
	for _, rbacItem := range rbacItems {
		duplicate := false
		for _, item := range items {
			if item.ClusterID == rbacItem.ClusterID && item.Namespace == rbacItem.Namespace {
				duplicate = true
				break
			}
		}
		if !duplicate {
			items = append(items, rbacItem)
		}
	}

	result.Success(c, items)
}

// getMyRbacPermissions 获取用户的RBAC规则权限列表
func (s *K8sPermissionServiceImpl) getMyRbacPermissions(userID uint, clusterNameMap map[uint]string) []MyPermissionItem {
	groupIDs, _ := s.groupDao.GetUserGroupIDs(userID)

	subjects := []struct {
		Type string
		ID   uint
	}{
		{Type: "User", ID: userID},
	}
	for _, gid := range groupIDs {
		subjects = append(subjects, struct {
			Type string
			ID   uint
		}{Type: "Group", ID: gid})
	}

	bindings, err := s.rbacDao.GetBindingsBySubjects(subjects)
	if err != nil || len(bindings) == 0 {
		return nil
	}

	roleIDs := make([]uint, 0)
	roleIDMap := make(map[uint]bool)
	for _, b := range bindings {
		if !roleIDMap[b.RoleID] {
			roleIDMap[b.RoleID] = true
			roleIDs = append(roleIDs, b.RoleID)
		}
	}

	roles, err := s.rbacDao.GetRolesByIDs(roleIDs)
	if err != nil || len(roles) == 0 {
		return nil
	}

	roleMap := make(map[uint]model.K8sRbacRole)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	keyMap := make(map[string]*MyPermissionItem)
	for _, b := range bindings {
		role, ok := roleMap[b.RoleID]
		if !ok {
			continue
		}

		var rules []model.K8sRule
		if err := json.Unmarshal([]byte(role.Rules), &rules); err != nil {
			continue
		}

		key := fmt.Sprintf("%d:%s", b.ClusterID, b.Namespace)
		if existing, ok := keyMap[key]; ok {
			existing.Rules = append(existing.Rules, rules...)
		} else {
			clusterName := clusterNameMap[b.ClusterID]

			// 推导permissionType
			permType := derivePermissionTypeFromRules(rules)
			keyMap[key] = &MyPermissionItem{
				ClusterID:      b.ClusterID,
				ClusterName:    clusterName,
				Namespace:      b.Namespace,
				PermissionType: permType,
				Source:         "rbac",
				Rules:          rules,
			}
		}
	}

	var items []MyPermissionItem
	for _, v := range keyMap {
		items = append(items, *v)
	}
	return items
}

// derivePermissionTypeFromRules 从RBAC rules中推导权限级别
func derivePermissionTypeFromRules(rules []model.K8sRule) string {
	hasRead := false
	hasWrite := false
	for _, rule := range rules {
		for _, v := range rule.Verbs {
			switch v {
			case "*":
				return "admin"
			case "create", "update", "delete", "patch":
				hasWrite = true
			case "get", "list", "watch":
				hasRead = true
			}
		}
	}
	if hasWrite {
		return "write"
	}
	if hasRead {
		return "readonly"
	}
	return ""
}
