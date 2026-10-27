package middleware

import (
	"dodevops-api/api/k8s/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Resolve a request's grants once, excluding disabled groups and invalid role bindings.
func (m *K8sPermissionMiddleware) autoscalingAccess(user, cluster uint, ns string) func(string, string, string) bool {
	if admin, err := m.permDao.IsAdmin(user); err == nil && admin {
		return func(string, string, string) bool { return true }
	}
	level := ""
	takeLevel := func(next string) {
		if permissionLevelRank[next] > permissionLevelRank[level] {
			level = next
		}
	}
	if perm, err := m.permDao.CheckPermission(user, cluster, ns); err == nil && perm != nil {
		takeLevel(perm.PermissionType)
	}
	subjects := []struct {
		Type string
		ID   uint
	}{{Type: "User", ID: user}}
	groupIDs, _ := m.getUserGroupIDs(user)
	for _, gid := range groupIDs {
		group, err := m.groupDao.GetByID(gid)
		if err != nil || group.Status != 1 {
			continue
		}
		if perm, err := m.gPermDao.CheckGroupPermission(gid, cluster, ns); err == nil && perm != nil {
			takeLevel(perm.PermissionType)
		}
		subjects = append(subjects, struct {
			Type string
			ID   uint
		}{Type: "Group", ID: gid})
	}
	var rules []model.K8sRule
	bindings, err := m.rbacDao.GetBindingsBySubjects(subjects)
	if err == nil {
		for _, binding := range bindings {
			if binding.ClusterID != cluster || (binding.Namespace != "" && binding.Namespace != ns) {
				continue
			}
			role, err := m.rbacDao.GetRoleByID(binding.RoleID)
			if err != nil || role.ClusterID != cluster || (role.Namespace != "" && role.Namespace != binding.Namespace) {
				continue
			}
			var parsed []model.K8sRule
			if json.Unmarshal([]byte(role.Rules), &parsed) == nil {
				rules = append(rules, parsed...)
			}
		}
	}
	return func(group, resource, verb string) bool {
		if level == "admin" || level == "write" || (level == "readonly" && (verb == "get" || verb == "list")) {
			return true
		}
		for _, rule := range rules {
			resources := make([]string, len(rule.Resources))
			for i, r := range rule.Resources {
				resources[i] = canonicalK8sResource(r)
			}
			if autoscalingContains(rule.APIGroups, group) && autoscalingContains(resources, resource) && autoscalingContains(rule.Verbs, verb) {
				return true
			}
		}
		return false
	}
}
func autoscalingContains(values []string, value string) bool {
	for _, v := range values {
		if v == "*" || v == value {
			return true
		}
	}
	return false
}

func (m *K8sPermissionMiddleware) CheckAutoscalingPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, 401, "未授权")
			c.Abort()
			return
		}
		cluster, err := strconv.ParseUint(c.Param("id"), 10, 32)
		ns := c.Param("namespaceName")
		if err != nil || cluster == 0 || ns == "all" || len(validation.IsDNS1123Label(ns)) > 0 {
			result.Failed(c, 400, "请选择有效集群和单个命名空间")
			c.Abort()
			return
		}
		resources := map[string][2]string{"hpa": {"autoscaling", "horizontalpodautoscalers"}, "vpa": {"autoscaling.k8s.io", "verticalpodautoscalers"}, "kpa": {"serving.knative.dev", "services"}}
		allowed := m.autoscalingAccess(user.ID, uint(cluster), ns)
		permissions := map[string]map[string]bool{}
		anyAllowed := false
		for kind, r := range resources {
			permissions[kind] = map[string]bool{}
			for _, verb := range []string{"get", "list", "create", "update", "delete"} {
				actual := verb
				if kind == "kpa" && (verb == "create" || verb == "delete") {
					actual = "update"
				}
				permissions[kind][verb] = allowed(r[0], r[1], actual)
				anyAllowed = anyAllowed || permissions[kind][verb]
			}
		}
		c.Set("autoscalingPermissions", permissions)
		kind := c.Param("kind")
		if kind == "" {
			if !anyAllowed {
				result.Failed(c, 403, "没有此命名空间的弹性伸缩权限")
				c.Abort()
				return
			}
			c.Next()
			return
		}
		if _, ok := resources[kind]; !ok {
			result.Failed(c, 400, "仅支持 hpa、vpa、kpa")
			c.Abort()
			return
		}
		verb := "list"
		switch c.Request.Method {
		case http.MethodPost:
			verb = "create"
		case http.MethodPut:
			verb = "update"
		case http.MethodDelete:
			verb = "delete"
		case http.MethodGet:
			if c.Param("name") != "" {
				verb = "get"
			}
		}
		if !permissions[kind][verb] {
			result.Failed(c, 403, fmt.Sprintf("需要 %s/%s 的 %s 权限", resources[kind][0], resources[kind][1], verb))
			c.Abort()
			return
		}
		c.Next()
	}
}
