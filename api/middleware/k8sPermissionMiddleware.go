// K8s权限中间件
// 用于K8s模块的命名空间级别权限控制
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// K8sPermissionMiddleware K8s权限中间件
// 检查用户是否有权限访问指定集群的命名空间
// 采用RBAC模型：用户 -> 用户组 -> 权限
// 管理员拥有所有权限，普通用户只能访问被授权的命名空间（直接授权+用户组继承）
// 同时支持新RBAC(verbs)和旧权限(readonly/write/admin)模型
type K8sPermissionMiddleware struct {
	permDao  *dao.K8sPermissionDao
	groupDao *dao.K8sUserGroupDao
	gPermDao *dao.K8sGroupPermissionDao
	rbacDao  *dao.K8sRbacDao
}

func NewK8sPermissionMiddleware() *K8sPermissionMiddleware {
	db := common.GetDB()
	return &K8sPermissionMiddleware{
		permDao:  dao.NewK8sPermissionDao(db),
		groupDao: dao.NewK8sUserGroupDao(db),
		gPermDao: dao.NewK8sGroupPermissionDao(db),
		rbacDao:  dao.NewK8sRbacDao(db),
	}
}

// CheckNamespacePermission 检查命名空间级别的权限
// 从URL路径中提取 clusterId 和 namespace，验证当前用户是否有权限
func (m *K8sPermissionMiddleware) CheckNamespacePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 检查是否为管理员 - 管理员拥有所有权限
		isAdmin, _ := m.permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 从URL路径中提取 clusterId
		clusterIDStr := c.Param("id")
		if clusterIDStr == "" {
			clusterIDStr = c.Param("clusterId")
		}

		if clusterIDStr == "" {
			c.Next()
			return
		}

		// 从URL路径中提取 namespace
		namespace := c.Param("namespaceName")
		if namespace == "" {
			// 没有命名空间参数，检查用户是否有该集群的任意权限
			hasPerm, _ := m.hasAnyClusterPermission(admin.ID, clusterIDStr)
			if !hasPerm {
				result.Failed(c, http.StatusForbidden, "您没有该集群的访问权限")
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// 检查用户是否有该集群命名空间的传统粗粒度权限
		hasPerm, _ := m.hasNamespacePermission(admin.ID, clusterIDStr, namespace)
		if !hasPerm {
			// 没有粗粒度权限，则检查确切的资源与动词权限
			clusterIDUint, _ := strconv.ParseUint(clusterIDStr, 10, 32)
			resource, verb := m.getResourceAndVerbFromCtx(c)
			if !m.hasRbacResourceVerb(admin.ID, uint(clusterIDUint), namespace, resource, verb) {
				result.Failed(c, http.StatusForbidden, "您没有该资源的访问权限")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CheckNamespaceWritePermission 检查命名空间的写权限
// 只允许 write 和 admin 级别的用户通过，readonly 用户将被拒绝
func (m *K8sPermissionMiddleware) CheckNamespaceWritePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员拥有所有权限
		isAdmin, _ := m.permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 从URL路径中提取 clusterId
		clusterIDStr := c.Param("id")
		if clusterIDStr == "" {
			clusterIDStr = c.Param("clusterId")
		}

		if clusterIDStr == "" {
			result.Failed(c, http.StatusBadRequest, "缺少集群ID")
			c.Abort()
			return
		}

		clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, "无效的集群ID")
			c.Abort()
			return
		}

		// 从URL路径中提取 namespace
		namespace := c.Param("namespaceName")
		if namespace == "" {
			result.Failed(c, http.StatusBadRequest, "缺少命名空间参数")
			c.Abort()
			return
		}

		// 检查用户是否有写权限（write 或 admin）或匹配细粒度RBAC规则
		hasTraditionalWrite := m.hasWritePermission(admin.ID, uint(clusterID), namespace)
		if !hasTraditionalWrite {
			resource, verb := m.getResourceAndVerbFromCtx(c)
			if !m.hasRbacResourceVerb(admin.ID, uint(clusterID), namespace, resource, verb) {
				result.Failed(c, http.StatusForbidden, "您没有该资源的写操作权限")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CheckClusterPermission 检查集群级别的权限
// 验证当前用户是否有权限访问指定集群
func (m *K8sPermissionMiddleware) CheckClusterPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员拥有所有权限
		isAdmin, _ := m.permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 从URL路径中提取 clusterId
		clusterIDStr := c.Param("id")
		if clusterIDStr == "" {
			clusterIDStr = c.Param("clusterId")
		}

		if clusterIDStr == "" {
			c.Next()
			return
		}

		// 检查用户是否有该集群的任意权限
		hasPerm, _ := m.hasAnyClusterPermission(admin.ID, clusterIDStr)
		if !hasPerm {
			result.Failed(c, http.StatusForbidden, "您没有该集群的访问权限")
			c.Abort()
			return
		}

		c.Next()
	}
}

// FilterAllowedNamespaces 过滤命名空间列表
// 对于非管理员用户，只返回其有权限的命名空间
func (m *K8sPermissionMiddleware) FilterAllowedNamespaces() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员不过滤
		isAdmin, _ := m.permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 从URL路径中提取 clusterId
		clusterIDStr := c.Param("id")
		if clusterIDStr == "" {
			clusterIDStr = c.Param("clusterId")
		}

		if clusterIDStr == "" {
			c.Next()
			return
		}

		// 将clusterIDStr转换为uint存入上下文
		clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, "无效的集群ID")
			c.Abort()
			return
		}

		// 获取用户允许的命名空间列表（直接授权+用户组继承），存入上下文
		allowedNS, _ := m.permDao.GetUserAllowedNamespaces(admin.ID)

		// 合并用户组继承的命名空间权限
		groupIDs, err := m.getUserGroupIDs(admin.ID)
		if err == nil && len(groupIDs) > 0 {
			groupNS, _ := m.gPermDao.GetGroupAllowedNamespaces(groupIDs)
			for cid, namespaces := range groupNS {
				existing, ok := allowedNS[cid]
				if !ok {
					allowedNS[cid] = namespaces
				} else {
					nsSet := make(map[string]bool)
					for _, ns := range existing {
						nsSet[ns] = true
					}
					for _, ns := range namespaces {
						if !nsSet[ns] {
							allowedNS[cid] = append(allowedNS[cid], ns)
						}
					}
				}
			}
		}

		// 合并RBAC规则的命名空间
		rbacNS := m.getUserRbacNamespaces(admin.ID)
		for cid, namespaces := range rbacNS {
			existing, ok := allowedNS[cid]
			if !ok {
				allowedNS[cid] = namespaces
			} else {
				nsSet := make(map[string]bool)
				for _, ns := range existing {
					nsSet[ns] = true
				}
				for _, ns := range namespaces {
					if !nsSet[ns] {
						allowedNS[cid] = append(allowedNS[cid], ns)
					}
				}
			}
		}

		c.Set("k8s_allowed_namespaces", allowedNS)
		c.Set("k8s_current_cluster_id", uint(clusterID))
		c.Next()
	}
}

// permissionLevelRank 权限级别权重（数值越高权限越大）
var permissionLevelRank = map[string]int{
	"":         0,
	"readonly": 1,
	"write":    2,
	"admin":    3,
}

// getUserGroupIDs 获取用户所在的所有用户组ID
func (m *K8sPermissionMiddleware) getUserGroupIDs(userID uint) ([]uint, error) {
	ids, err := m.groupDao.GetUserGroupIDs(userID)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// getUserNamespacePermissionLevel 获取用户在指定集群命名空间的最高权限级别
// 返回值: "admin" / "write" / "readonly" / "" (无权限)
// 同时检查直接授权、用户组继承和RBAC规则，取最高级别
func (m *K8sPermissionMiddleware) getUserNamespacePermissionLevel(userID, clusterID uint, namespace string) string {
	highestLevel := ""
	highestRank := 0

	// 1. 检查直接授权
	perm, err := m.permDao.CheckPermission(userID, clusterID, namespace)
	if err == nil && perm != nil {
		rank := permissionLevelRank[perm.PermissionType]
		if rank > highestRank {
			highestRank = rank
			highestLevel = perm.PermissionType
		}
	}

	// 2. 检查用户组继承权限
	groupIDs, err := m.getUserGroupIDs(userID)
	if err == nil && len(groupIDs) > 0 {
		for _, gid := range groupIDs {
			gPerm, err := m.gPermDao.CheckGroupPermission(gid, clusterID, namespace)
			if err == nil && gPerm != nil {
				rank := permissionLevelRank[gPerm.PermissionType]
				if rank > highestRank {
					highestRank = rank
					highestLevel = gPerm.PermissionType
				}
			}
		}
	}

	return highestLevel
}

// getRbacMaxLevel 从RBAC规则中推导最大权限级别
func (m *K8sPermissionMiddleware) getRbacMaxLevel(userID, clusterID uint, namespace string) string {
	ruleMap := m.getUserRbacRules(userID)
	if ruleMap == nil {
		return ""
	}

	hasRead := false
	hasWrite := false

	key := fmt.Sprintf("%d:%s", clusterID, namespace)
	rules := ruleMap[key]
	if rules == nil {
		// 也检查集群级别规则
		clusterKey := fmt.Sprintf("%d:", clusterID)
		rules = ruleMap[clusterKey]
	}
	if rules == nil {
		return ""
	}

	for _, rule := range rules {
		for _, v := range rule.Verbs {
			switch v {
			case "create", "update", "delete", "patch":
				hasWrite = true
			case "get", "list", "watch":
				hasRead = true
			case "*":
				return "admin"
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

// hasWritePermission 检查用户是否有指定命名空间的写权限（write 或 admin）
func (m *K8sPermissionMiddleware) hasWritePermission(userID, clusterID uint, namespace string) bool {
	level := m.getUserNamespacePermissionLevel(userID, clusterID, namespace)
	rank := permissionLevelRank[level]
	return rank >= permissionLevelRank["write"]
}

// hasAdminPermission 检查用户是否有指定命名空间的管理员权限
func (m *K8sPermissionMiddleware) hasAdminPermission(userID, clusterID uint, namespace string) bool {
	level := m.getUserNamespacePermissionLevel(userID, clusterID, namespace)
	return level == "admin"
}

// ======================= RBAC 规则检查方法 =======================

// getUserRbacBindings 获取用户所有RBAC绑定（直接绑定+用户组继承）
func (m *K8sPermissionMiddleware) getUserRbacBindings(userID uint) []model.K8sRbacBinding {
	groupIDs, _ := m.getUserGroupIDs(userID)

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

	bindings, err := m.rbacDao.GetBindingsBySubjects(subjects)
	if err != nil || len(bindings) == 0 {
		return nil
	}
	return bindings
}

// getUserRbacRules 获取用户有效的RBAC规则映射 cluster:namespace -> []K8sRule
func (m *K8sPermissionMiddleware) getUserRbacRules(userID uint) map[string][]model.K8sRule {
	bindings := m.getUserRbacBindings(userID)
	if len(bindings) == 0 {
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

	roles, err := m.rbacDao.GetRolesByIDs(roleIDs)
	if err != nil || len(roles) == 0 {
		return nil
	}

	roleMap := make(map[uint]model.K8sRbacRole)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	ruleMap := make(map[string][]model.K8sRule)
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
		ruleMap[key] = append(ruleMap[key], rules...)
	}
	return ruleMap
}

// hasRbacVerbPermission 检查用户是否拥有指定集群命名空间下的任意一个verb权限
// verbs: 要检查的动词列表（如 ["get","list"]），命中任一即返回true
func (m *K8sPermissionMiddleware) hasRbacVerbPermission(userID, clusterID uint, namespace string, verbs []string) bool {
	ruleMap := m.getUserRbacRules(userID)
	if ruleMap == nil {
		return false
	}

	verbSet := make(map[string]bool)
	for _, v := range verbs {
		verbSet[v] = true
	}

	// 检查精确匹配 cluster:namespace
	key := fmt.Sprintf("%d:%s", clusterID, namespace)
	if rules, ok := ruleMap[key]; ok {
		for _, rule := range rules {
			for _, v := range rule.Verbs {
				if verbSet[v] {
					return true
				}
			}
		}
	}

	// 检查集群级别的规则（空namespace表示集群级别）
	clusterKey := fmt.Sprintf("%d:", clusterID)
	if rules, ok := ruleMap[clusterKey]; ok {
		for _, rule := range rules {
			for _, v := range rule.Verbs {
				if verbSet[v] {
					return true
				}
			}
		}
	}

	return false
}

// hasRbacNamespacePermission 检查用户是否有RBAC定义的该命名空间下任意权限
func (m *K8sPermissionMiddleware) hasRbacNamespacePermission(userID, clusterID uint, namespace string) bool {
	return m.hasRbacVerbPermission(userID, clusterID, namespace, []string{"get", "list", "watch", "create", "update", "delete"})
}

// getUserRbacNamespaces 获取用户从RBAC规则中获得的命名空间列表
func (m *K8sPermissionMiddleware) getUserRbacNamespaces(userID uint) map[uint][]string {
	bindings := m.getUserRbacBindings(userID)
	if len(bindings) == 0 {
		return nil
	}

	nsMap := make(map[uint]map[string]bool)
	for _, b := range bindings {
		ns := b.Namespace
		if ns == "" {
			ns = "*" // empty means all namespaces
		}
		if _, ok := nsMap[b.ClusterID]; !ok {
			nsMap[b.ClusterID] = make(map[string]bool)
		}
		nsMap[b.ClusterID][ns] = true
	}

	result := make(map[uint][]string)
	for cid, nsSet := range nsMap {
		for ns := range nsSet {
			result[cid] = append(result[cid], ns)
		}
	}
	return result
}

// ======================= 旧权限与RBAC混合检查 =======================

// hasNamespacePermission 检查用户是否有指定集群命名空间的权限（含用户组继承 + RBAC规则）
func (m *K8sPermissionMiddleware) hasNamespacePermission(userID uint, clusterIDStr, namespace string) (bool, error) {
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		return false, err
	}

	// 1. 检查直接授权
	perm, err := m.permDao.CheckPermission(userID, uint(clusterID), namespace)
	if err == nil && perm != nil {
		return true, nil
	}

	// 2. 检查用户组继承权限
	groupIDs, err := m.getUserGroupIDs(userID)
	if err == nil && len(groupIDs) > 0 {
		for _, gid := range groupIDs {
			gPerm, err := m.gPermDao.CheckGroupPermission(gid, uint(clusterID), namespace)
			if err == nil && gPerm != nil {
				return true, nil
			}
		}
	}

	return false, nil
}

// hasAnyClusterPermission 检查用户是否有该集群的任意权限（含用户组继承 + RBAC）
func (m *K8sPermissionMiddleware) hasAnyClusterPermission(userID uint, clusterIDStr string) (bool, error) {
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		return false, err
	}

	// 1. 检查直接授权
	perms, err := m.permDao.GetByUserAndCluster(userID, uint(clusterID))
	if err == nil && len(perms) > 0 {
		return true, nil
	}

	// 2. 检查用户组继承权限
	groupIDs, err := m.getUserGroupIDs(userID)
	if err == nil && len(groupIDs) > 0 {
		allowedClusterIDs, err := m.gPermDao.GetGroupAllowedClusterIDs(groupIDs)
		if err == nil {
			for _, cid := range allowedClusterIDs {
				if cid == uint(clusterID) {
					return true, nil
				}
			}
		}
	}

	// 3. 检查RBAC规则（此集群下有任何绑定即表示有访问权限）
	bindings := m.getUserRbacBindings(userID)
	for _, b := range bindings {
		if b.ClusterID == uint(clusterID) {
			return true, nil
		}
	}

	return false, nil
}

// GetUserAllowedNamespacesFromContext 从上下文中获取用户允许的命名空间
func GetUserAllowedNamespacesFromContext(c *gin.Context) map[uint][]string {
	if v, exists := c.Get("k8s_allowed_namespaces"); exists {
		if m, ok := v.(map[uint][]string); ok {
			return m
		}
	}
	return nil
}

// GetCurrentClusterIDFromContext 从上下文中获取当前集群ID
func GetCurrentClusterIDFromContext(c *gin.Context) uint {
	if v, exists := c.Get("k8s_current_cluster_id"); exists {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// PermDao 获取权限DAO（供外部使用）
func (m *K8sPermissionMiddleware) PermDao() *dao.K8sPermissionDao {
	return m.permDao
}

// FilterAllowedClusters 过滤集群列表
// 对于非管理员用户，只返回其有权限的集群（含用户组继承）
func (m *K8sPermissionMiddleware) FilterAllowedClusters() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员不过滤
		isAdmin, _ := m.permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 获取用户有权限访问的集群ID列表（直接授权）
		allowedClusters, _ := m.permDao.GetUserAllowedClusterIDs(admin.ID)

		// 合并用户组继承的集群权限
		groupIDs, err := m.groupDao.GetUserGroupIDs(admin.ID)
		if err == nil && len(groupIDs) > 0 {
			groupClusterIDs, _ := m.gPermDao.GetGroupAllowedClusterIDs(groupIDs)
			clusterSet := make(map[uint]bool)
			for _, cid := range allowedClusters {
				clusterSet[cid] = true
			}
			for _, cid := range groupClusterIDs {
				if !clusterSet[cid] {
					allowedClusters = append(allowedClusters, cid)
					clusterSet[cid] = true
				}
			}
		}

		// 合并RBAC规则的集群
		rbacBindings := m.getUserRbacBindings(admin.ID)
		clusterSet := make(map[uint]bool)
		for _, cid := range allowedClusters {
			clusterSet[cid] = true
		}
		for _, b := range rbacBindings {
			if !clusterSet[b.ClusterID] {
				allowedClusters = append(allowedClusters, b.ClusterID)
				clusterSet[b.ClusterID] = true
			}
		}
		if allowedClusters == nil {
			allowedClusters = []uint{} // Ensure it's not nil so type assertion works
		}
		c.Set("k8s_allowed_cluster_ids", allowedClusters)
		c.Next()
	}
}

// GetUserAllowedClusterIDsFromContext 从上下文中获取用户允许的集群ID列表
func GetUserAllowedClusterIDsFromContext(c *gin.Context) []uint {
	if v, exists := c.Get("k8s_allowed_cluster_ids"); exists {
		if ids, ok := v.([]uint); ok {
			return ids
		}
	}
	return nil
}

// 确保常量被使用
var _ = constant.ContextKeyUserObj

// ======================= 细粒度资源与操作权限检查 =======================

// getResourceAndVerbFromCtx 从Gin请求上下文中提取资源类型和操作动词
func (m *K8sPermissionMiddleware) getResourceAndVerbFromCtx(c *gin.Context) (string, string) {
	path := c.FullPath()

	verb := ""
	switch c.Request.Method {
	case "GET":
		if strings.HasSuffix(path, "/:deploymentName") ||
			strings.HasSuffix(path, "/:podName") ||
			strings.HasSuffix(path, "/:serviceName") ||
			strings.HasSuffix(path, "/:configMapName") ||
			strings.HasSuffix(path, "/:workloadName") ||
			strings.HasSuffix(path, "/:secretName") ||
			strings.HasSuffix(path, "/:pvcName") ||
			strings.HasSuffix(path, "/:ingressName") ||
			strings.HasSuffix(path, "/:nodeName") ||
			strings.HasSuffix(path, "/:jobName") ||
			strings.HasSuffix(path, "/:cronJobName") ||
			strings.HasSuffix(path, "/:daemonSetName") ||
			strings.HasSuffix(path, "/:statefulSetName") ||
			strings.HasSuffix(path, "/:namespaceName") {
			verb = "get"
		} else {
			verb = "list"
		}
	case "POST":
		if strings.HasSuffix(path, "/pause") || strings.HasSuffix(path, "/resume") {
			verb = "update"
		} else {
			verb = "create"
		}
	case "PUT", "PATCH":
		verb = "update"
	case "DELETE":
		verb = "delete"
	}

	parts := strings.Split(path, "/")
	resource := ""
	for i, p := range parts {
		if p == ":namespaceName" && i+1 < len(parts) {
			resource = parts[i+1]
			break
		}
	}

	if resource == "" {
		for i, p := range parts {
			if p == ":id" && i+1 < len(parts) && parts[i+1] != "namespaces" && parts[i+1] != "yaml" {
				resource = parts[i+1]
				break
			}
		}
	}

	// 检测 namespace 路由（资源名为 namespaces）
	if resource == "" {
		for _, p := range parts {
			if p == "namespaces" {
				resource = "namespaces"
				break
			}
		}
	}

	if resource == "workloads" {
		resource = c.Param("type")
		if resource == "" {
			resource = "workloads" // generic
		}
	} else if resource == "workload-yaml" {
		resource = c.Param("workloadType")
		if resource == "" {
			resource = "workloads"
		}
	}

	return resource, verb
}

// hasRbacResourceVerb 检查特定的资源和动词权限
func (m *K8sPermissionMiddleware) hasRbacResourceVerb(userID, clusterID uint, namespace, resource, verb string) bool {
	ruleMap := m.getUserRbacRules(userID)
	if ruleMap == nil {
		return false
	}

	key := fmt.Sprintf("%d:%s", clusterID, namespace)
	rules := ruleMap[key]
	if rules == nil {
		clusterKey := fmt.Sprintf("%d:", clusterID)
		rules = ruleMap[clusterKey]
	}
	if rules == nil {
		return false
	}

	for _, rule := range rules {
		matchRes := false
		for _, r := range rule.Resources {
			if r == "*" || r == resource {
				matchRes = true
				break
			}
			// 处理资源名单复数不匹配: workloads 匹配所有工作负载类型
			if resource == "workloads" && (r == "deployments" || r == "statefulsets" || r == "daemonsets" || r == "jobs" || r == "cronjobs") {
				matchRes = true
				break
			}
			// 处理 :type 参数传过来的单数资源名 (如 "deployment" → 匹配 "deployments")
			if (r == "deployments" && resource == "deployment") ||
				(r == "statefulsets" && resource == "statefulset") ||
				(r == "daemonsets" && resource == "daemonset") ||
				(r == "jobs" && resource == "job") ||
				(r == "cronjobs" && resource == "cronjob") ||
				(r == "pods" && resource == "pod") ||
				(r == "services" && resource == "service") ||
				(r == "configmaps" && resource == "configmap") ||
				(r == "secrets" && resource == "secret") ||
				(r == "ingresses" && resource == "ingress") ||
				(r == "nodes" && resource == "node") ||
				(r == "namespaces" && resource == "namespace") ||
				(r == "events" && resource == "event") ||
				(r == "endpoints" && resource == "endpoint") ||
				(r == "persistentvolumes" && resource == "persistentvolume") ||
				(r == "persistentvolumeclaims" && resource == "persistentvolumeclaim") {
				matchRes = true
				break
			}
		}

		matchVerb := false
		for _, v := range rule.Verbs {
			if v == "*" || v == verb {
				matchVerb = true
				break
			}
		}

		if matchRes && matchVerb {
			return true
		}
	}
	return false
}
