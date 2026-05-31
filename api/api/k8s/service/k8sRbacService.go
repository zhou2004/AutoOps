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

type IK8sRbacService interface {
	CreateRole(c *gin.Context, req *model.CreateRbacRoleReq)
	UpdateRole(c *gin.Context, id uint, req *model.CreateRbacRoleReq)
	DeleteRole(c *gin.Context, id uint)
	GetRoleList(c *gin.Context, clusterID uint, namespace, name string, page, size int)

	CreateBinding(c *gin.Context, req *model.CreateRbacBindingReq)
	UpdateBinding(c *gin.Context, id uint, req *model.CreateRbacBindingReq)
	DeleteBinding(c *gin.Context, id uint)
	GetBindingList(c *gin.Context, clusterID uint, namespace, subjectType, subjectName string, page, size int)

	GetMyPermissions(c *gin.Context)
}

type K8sRbacServiceImpl struct {
	dao      *dao.K8sRbacDao
	groupDao *dao.K8sUserGroupDao
}

func NewK8sRbacService(db *gorm.DB) IK8sRbacService {
	return &K8sRbacServiceImpl{
		dao:      dao.NewK8sRbacDao(db),
		groupDao: dao.NewK8sUserGroupDao(db),
	}
}

func (s *K8sRbacServiceImpl) CreateRole(c *gin.Context, req *model.CreateRbacRoleReq) {
	rulesJson, _ := json.Marshal(req.Rules)
	role := &model.K8sRbacRole{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		Name:      req.Name,
		Rules:     string(rulesJson),
	}
	if err := s.dao.CreateRole(role); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建角色失败: "+err.Error())
		return
	}
	result.Success(c, role)
}

func (s *K8sRbacServiceImpl) UpdateRole(c *gin.Context, id uint, req *model.CreateRbacRoleReq) {
	role, err := s.dao.GetRoleByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "角色未找到")
		return
	}

	rulesJson, _ := json.Marshal(req.Rules)
	role.Name = req.Name
	role.Namespace = req.Namespace
	role.Rules = string(rulesJson)

	if err := s.dao.UpdateRole(role); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新角色失败: "+err.Error())
		return
	}
	result.Success(c, role)
}

func (s *K8sRbacServiceImpl) DeleteRole(c *gin.Context, id uint) {
	if err := s.dao.DeleteRole(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除角色失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *K8sRbacServiceImpl) GetRoleList(c *gin.Context, clusterID uint, namespace, name string, page, size int) {
	vos, total, err := s.dao.GetRoleList(clusterID, namespace, name, page, size)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取角色列表失败: "+err.Error())
		return
	}

	// 解析 rules JSON
	for i := range vos {
		var rules []model.K8sRule
		json.Unmarshal([]byte(vos[i].RulesStr), &rules)
		vos[i].Rules = rules
	}
	result.Success(c, gin.H{
		"list":  vos,
		"total": total,
	})
}

func (s *K8sRbacServiceImpl) CreateBinding(c *gin.Context, req *model.CreateRbacBindingReq) {
	binding := &model.K8sRbacBinding{
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		RoleID:      req.RoleID,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
	}
	if err := s.dao.CreateBinding(binding); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建绑定失败: "+err.Error())
		return
	}
	result.Success(c, binding)
}

func (s *K8sRbacServiceImpl) UpdateBinding(c *gin.Context, id uint, req *model.CreateRbacBindingReq) {
	binding, err := s.dao.GetBindingByID(id)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "绑定未找到")
		return
	}
	binding.ClusterID = req.ClusterID
	binding.Namespace = req.Namespace
	binding.RoleID = req.RoleID
	binding.SubjectType = req.SubjectType
	binding.SubjectID = req.SubjectID
	if err := s.dao.UpdateBinding(binding); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新绑定失败: "+err.Error())
		return
	}
	result.Success(c, binding)
}

func (s *K8sRbacServiceImpl) DeleteBinding(c *gin.Context, id uint) {
	if err := s.dao.DeleteBinding(id); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除绑定失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (s *K8sRbacServiceImpl) GetBindingList(c *gin.Context, clusterID uint, namespace, subjectType, subjectName string, page, size int) {
	vos, total, err := s.dao.GetBindingList(clusterID, namespace, subjectType, subjectName, page, size)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取绑定列表失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{
		"list":  vos,
		"total": total,
	})
}

func (s *K8sRbacServiceImpl) GetMyPermissions(c *gin.Context) {
	claims, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "获取用户信息失败")
		return
	}
	userID := claims.ID

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

	bindings, err := s.dao.GetBindingsBySubjects(subjects)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取RBAC绑定失败: "+err.Error())
		return
	}

	if len(bindings) == 0 {
		result.Success(c, []interface{}{})
		return
	}

	roleIDs := make([]uint, 0)
	roleIDMap := make(map[uint]bool)
	for _, b := range bindings {
		if !roleIDMap[b.RoleID] {
			roleIDMap[b.RoleID] = true
			roleIDs = append(roleIDs, b.RoleID)
		}
	}

	roles, err := s.dao.GetRolesByIDs(roleIDs)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取RBAC角色失败: "+err.Error())
		return
	}

	roleMap := make(map[uint]model.K8sRbacRole)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	type Permission struct {
		ClusterID uint            `json:"clusterId"`
		Namespace string          `json:"namespace"`
		Rules     []model.K8sRule `json:"rules"`
	}

	permKeyMap := make(map[string]*Permission)

	for _, b := range bindings {
		role, ok := roleMap[b.RoleID]
		if !ok {
			continue
		}

		var rules []model.K8sRule
		json.Unmarshal([]byte(role.Rules), &rules)

		key := fmt.Sprintf("%d:%s", b.ClusterID, b.Namespace)
		if p, ok := permKeyMap[key]; ok {
			p.Rules = append(p.Rules, rules...)
		} else {
			permKeyMap[key] = &Permission{
				ClusterID: b.ClusterID,
				Namespace: b.Namespace,
				Rules:     rules,
			}
		}
	}

	var finalPerms []*Permission
	for _, v := range permKeyMap {
		finalPerms = append(finalPerms, v)
	}

	result.Success(c, finalPerms)
}
