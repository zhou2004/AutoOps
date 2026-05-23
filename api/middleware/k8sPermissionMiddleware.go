// K8s权限中间件
// 用于K8s模块的命名空间级别权限控制
package middleware

import (
	"net/http"
	"strconv"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// K8sPermissionMiddleware K8s权限中间件
// 检查用户是否有权限访问指定集群的命名空间
// 管理员拥有所有权限，普通用户只能访问被授权的命名空间
type K8sPermissionMiddleware struct {
	permDao *dao.K8sPermissionDao
}

func NewK8sPermissionMiddleware() *K8sPermissionMiddleware {
	db := common.GetDB()
	return &K8sPermissionMiddleware{
		permDao: dao.NewK8sPermissionDao(db),
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

		// 检查用户是否有该集群命名空间的权限
		hasPerm, _ := m.hasNamespacePermission(admin.ID, clusterIDStr, namespace)
		if !hasPerm {
			result.Failed(c, http.StatusForbidden, "您没有该命名空间的访问权限")
			c.Abort()
			return
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

		// 获取用户允许的命名空间列表，存入上下文
		allowedNS, _ := m.permDao.GetUserAllowedNamespaces(admin.ID)
		c.Set("k8s_allowed_namespaces", allowedNS)
		c.Set("k8s_current_cluster_id", uint(clusterID))
		c.Next()
	}
}

// hasNamespacePermission 检查用户是否有指定集群命名空间的权限
func (m *K8sPermissionMiddleware) hasNamespacePermission(userID uint, clusterIDStr, namespace string) (bool, error) {
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		return false, err
	}

	perm, err := m.permDao.CheckPermission(userID, uint(clusterID), namespace)
	if err != nil {
		return false, err
	}
	return perm != nil, nil
}

// hasAnyClusterPermission 检查用户是否有该集群的任意权限
func (m *K8sPermissionMiddleware) hasAnyClusterPermission(userID uint, clusterIDStr string) (bool, error) {
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		return false, err
	}

	perms, err := m.permDao.GetByUserAndCluster(userID, uint(clusterID))
	if err != nil {
		return false, err
	}
	return len(perms) > 0, nil
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
// 对于非管理员用户，只返回其有权限的集群
func FilterAllowedClusters(permDao *dao.K8sPermissionDao) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员不过滤
		isAdmin, _ := permDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		// 获取用户有权限访问的集群ID列表，存入上下文
		allowedClusters, _ := permDao.GetUserAllowedClusterIDs(admin.ID)
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
