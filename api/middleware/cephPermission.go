package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"dodevops-api/api/k8s/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// Ceph access is scoped to the storage administration namespace. Access to an
// unrelated application namespace must not grant cluster-wide storage access.
func (m *K8sPermissionMiddleware) cephVerbAllowed(userID, clusterID uint, verb string) bool {
	if admin, err := m.permDao.IsAdmin(userID); err == nil && admin {
		return true
	}
	legacyAllowed := func(level string) bool {
		return level == "admin" || level == "write" || (verb == "get" && level == "readonly")
	}
	if perm, err := m.permDao.CheckPermission(userID, clusterID, "rook-ceph"); err == nil && perm != nil && legacyAllowed(perm.PermissionType) {
		return true
	}
	// Ceph owns cluster storage credentials. Do not inherit disabled groups or
	// bindings to a namespaced role outside that role's declared scope.
	subjects := []struct {
		Type string
		ID   uint
	}{{Type: "User", ID: userID}}
	groupIDs, _ := m.getUserGroupIDs(userID)
	for _, groupID := range groupIDs {
		group, err := m.groupDao.GetByID(groupID)
		if err != nil || group.Status != 1 {
			continue
		}
		if perm, err := m.gPermDao.CheckGroupPermission(groupID, clusterID, "rook-ceph"); err == nil && perm != nil && legacyAllowed(perm.PermissionType) {
			return true
		}
		subjects = append(subjects, struct {
			Type string
			ID   uint
		}{Type: "Group", ID: groupID})
	}
	bindings, err := m.rbacDao.GetBindingsBySubjects(subjects)
	if err != nil {
		return false
	}
	for _, binding := range bindings {
		if binding.ClusterID != clusterID || (binding.Namespace != "rook-ceph" && binding.Namespace != "") {
			continue
		}
		role, err := m.rbacDao.GetRoleByID(binding.RoleID)
		if err != nil || role.ClusterID != clusterID || (role.Namespace != "" && role.Namespace != binding.Namespace) {
			continue
		}
		var rules []model.K8sRule
		if json.Unmarshal([]byte(role.Rules), &rules) != nil {
			continue
		}
		for _, rule := range rules {
			group, resource, allowed := false, false, false
			for _, v := range rule.APIGroups {
				if v == "*" || v == "autoops.io" {
					group = true
				}
			}
			for _, v := range rule.Resources {
				if v == "*" || v == "cephmaintenance" {
					resource = true
				}
			}
			for _, v := range rule.Verbs {
				if v == "*" || v == verb {
					allowed = true
				}
			}
			if group && resource && allowed {
				return true
			}
		}
	}
	return false
}

func (m *K8sPermissionMiddleware) CheckCephPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, 401, "未授权")
			c.Abort()
			return
		}
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil || id == 0 {
			result.Failed(c, 400, "无效的集群ID")
			c.Abort()
			return
		}
		// All Ceph mutations share update permission, including tasks which may
		// delete storage. This is an explicit maintenance privilege, not jobs/create.
		verb := "get"
		if c.Request.Method != http.MethodGet {
			verb = "update"
		}
		if !m.cephVerbAllowed(admin.ID, uint(id), verb) {
			result.Failed(c, 403, "需要 rook-ceph 命名空间的相应权限，或 autoops.io/cephmaintenance 的 get/update 权限")
			c.Abort()
			return
		}
		c.Set("cephCanWrite", m.cephVerbAllowed(admin.ID, uint(id), "update"))
		c.Next()
	}
}
