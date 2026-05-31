package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// CmdbAssetPermissionMiddleware CMDB 资产授权中间件
// 类似 JumpServer 的资产授权控制，检查用户是否有权限访问指定资产
type CmdbAssetPermissionMiddleware struct {
	permDao          *dao.CmdbAssetPermissionDao
	cachePermDao     *dao.CmdbAssetPermissionDao
	groupDao         dao.CmdbGroupDao
	cmdbUserGroupDao *dao.CmdbUserGroupDao
	hostDao          dao.CmdbHostDao
	physicalDao      *dao.CmdbPhysicalMachineDao
	networkDao       *dao.CmdbNetworkDeviceDao
}

func NewCmdbAssetPermissionMiddleware() *CmdbAssetPermissionMiddleware {
	db := common.GetDB()
	return &CmdbAssetPermissionMiddleware{
		permDao:          dao.NewCmdbAssetPermissionDao(db),
		cachePermDao:     nil,
		groupDao:         dao.NewCmdbGroupDao(),
		cmdbUserGroupDao: dao.NewCmdbUserGroupDao(),
		hostDao:          dao.NewCmdbHostDao(),
		physicalDao:      dao.NewCmdbPhysicalMachineDao(db),
		networkDao:       dao.NewCmdbNetworkDeviceDao(db),
	}
}

// getUserEffectivePermissions 获取用户的有效授权 (使用CMDB用户组)
func (m *CmdbAssetPermissionMiddleware) getUserEffectivePermissions(userID uint) []model.CmdbAssetPermission {
	// 获取CMDB用户组ID
	userGroupIDs, _ := m.cmdbUserGroupDao.GetUserGroupIDs(userID)
	// 同时兼容旧的sys_role admin分组
	sysGroupIDs, _ := m.getSystemAdminGroupIDs(userID)
	allGroupIDs := append(userGroupIDs, sysGroupIDs...)
	perms, _ := m.permDao.GetUserPermissions(userID, allGroupIDs)
	return perms
}

// getSystemAdminGroupIDs 获取系统角色中的管理员组ID(兼容旧数据)
func (m *CmdbAssetPermissionMiddleware) getSystemAdminGroupIDs(userID uint) ([]uint, error) {
	var ids []uint
	db := common.GetDB()
	err := db.Table("sys_admin_role").
		Joins("JOIN sys_role ON sys_role.id = sys_admin_role.role_id").
		Where("sys_admin_role.admin_id = ? AND sys_role.role_key = ?", userID, "admin").
		Pluck("sys_role.id", &ids).Error
	return ids, err
}

// CheckHostPermission 检查是否有权访问指定主机
func (m *CmdbAssetPermissionMiddleware) CheckHostPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员跳过检查
		isAdmin, _ := m.groupDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		hostIDStr := c.Param("id")
		if hostIDStr == "" {
			hostIDStr = c.Query("id")
		}
		if hostIDStr == "" {
			c.Next()
			return
		}

		hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, "无效的主机ID")
			c.Abort()
			return
		}

		perms := m.getUserEffectivePermissions(admin.ID)
		if !m.hasHostPermission(perms, uint(hostID)) {
			result.Failed(c, http.StatusForbidden, "您没有该主机的访问权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// FilterAllowedHosts 过滤主机列表，只返回用户有权限的主机
func (m *CmdbAssetPermissionMiddleware) FilterAllowedHosts() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		isAdmin, _ := m.groupDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		perms := m.getUserEffectivePermissions(admin.ID)
		allowedIDs := make(map[uint]bool)

		for _, p := range perms {
			// 具体主机ID
			var hostIDs []uint
			json.Unmarshal([]byte(p.HostIDs), &hostIDs)
			for _, id := range hostIDs {
				allowedIDs[id] = true
			}
			// 主机分组
			var hgIDs []uint
			json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
			for _, gid := range hgIDs {
				hosts := m.hostDao.GetCmdbHostsByGroupId(gid)
				for _, h := range hosts {
					allowedIDs[h.ID] = true
				}
			}
		}

		c.Set("cmdb_allowed_host_ids", allowedIDs)
		c.Next()
	}
}

// hasHostPermission 检查是否有指定主机的权限(包含具体主机ID和分组)
func (m *CmdbAssetPermissionMiddleware) hasHostPermission(perms []model.CmdbAssetPermission, hostID uint) bool {
	for _, p := range perms {
		// 检查具体主机ID
		var hostIDs []uint
		json.Unmarshal([]byte(p.HostIDs), &hostIDs)
		for _, id := range hostIDs {
			if id == hostID {
				return true
			}
		}
		// 检查主机分组
		var hgIDs []uint
		json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
		for _, gid := range hgIDs {
			hosts := m.hostDao.GetCmdbHostsByGroupId(gid)
			for _, h := range hosts {
				if h.ID == hostID {
					return true
				}
			}
		}
	}
	return false
}

// hasAssetType 检查授权是否包含指定资产类型
func hasAssetType(perms []model.CmdbAssetPermission, assetType string) bool {
	for _, p := range perms {
		var types []string
		json.Unmarshal([]byte(p.AssetTypes), &types)
		for _, t := range types {
			if t == assetType || t == "*" {
				return true
			}
		}
	}
	return false
}

// CheckActionPermission 检查是否有指定操作权限
func (m *CmdbAssetPermissionMiddleware) CheckActionPermission(requiredAction string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		isAdmin, _ := m.groupDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		perms := m.getUserEffectivePermissions(admin.ID)
		for _, p := range perms {
			var actions []string
			json.Unmarshal([]byte(p.PermissionActions), &actions)
			for _, a := range actions {
				if a == requiredAction || a == "admin" {
					c.Next()
					return
				}
			}
		}

		result.Failed(c, http.StatusForbidden, "您没有该操作的权限")
		c.Abort()
	}
}

// RequireCmdbPermission 校验用户有指定资产类型的指定操作权限，并在Context中注入允许的资产ID列表
// assetType: host/physical/network/database/idc/cabinet/group
// 自动根据 HTTP Method 映射 action: GET→list/get, POST→create, PUT→update, DELETE→delete
// Context注入: cmdb_allowed_asset_ids(map[uint]bool), cmdb_permission_actions([]string)
func (m *CmdbAssetPermissionMiddleware) RequireCmdbPermission(assetType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		// 管理员拥有所有权限
		isAdmin, _ := m.groupDao.IsAdmin(admin.ID)
		if isAdmin {
			c.Next()
			return
		}

		perms := m.getUserEffectivePermissions(admin.ID)
		if len(perms) == 0 {
			result.Failed(c, http.StatusForbidden, "您没有CMDB的访问权限，请联系管理员授权")
			c.Abort()
			return
		}

		// 根据HTTP方法检查操作权限
		requiredAction := methodToAction(c.Request.Method)

		// 在同一条规则中同时检查：资产类型 + 操作权限
		if !m.hasPermissionForAsset(perms, assetType, requiredAction) {
			if requiredAction != "" {
				result.Failed(c, http.StatusForbidden, "您没有对"+assetTypeLabel(assetType)+"的"+actionLabel(requiredAction)+"权限")
			} else {
				result.Failed(c, http.StatusForbidden, "您没有"+assetTypeLabel(assetType)+"的访问权限")
			}
			c.Abort()
			return
		}

		// 在请求上下文中存储用户的权限actions
		var allActions []string
		for _, p := range perms {
			var actions []string
			json.Unmarshal([]byte(p.PermissionActions), &actions)
			for _, a := range actions {
				if !contains(allActions, a) {
					allActions = append(allActions, a)
				}
			}
		}
		c.Set("cmdb_permission_actions", allActions)

		// 计算并注入用户被允许的资产ID集合（用于列表过滤）
		allowedIDs := m.computeAllowedAssetIDs(perms, assetType)
		c.Set("cmdb_allowed_asset_ids", allowedIDs)
		c.Next()
	}
}

// computeAllowedAssetIDs 从授权规则中提取用户被允许的指定资产类型的ID集合
func (m *CmdbAssetPermissionMiddleware) computeAllowedAssetIDs(perms []model.CmdbAssetPermission, assetType string) map[uint]bool {
	result := make(map[uint]bool)

	for _, p := range perms {
		// 先检查该规则是否有此资产类型
		var types []string
		json.Unmarshal([]byte(p.AssetTypes), &types)
		hasType := false
		for _, t := range types {
			if t == assetType || t == "*" {
				hasType = true
				break
			}
		}
		if !hasType {
			continue
		}

		switch assetType {
		case "host":
			// 具体主机ID
			var hostIDs []uint
			json.Unmarshal([]byte(p.HostIDs), &hostIDs)
			for _, id := range hostIDs {
				result[id] = true
			}
			// 主机分组下的所有主机
			var hgIDs []uint
			json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
			for _, gid := range hgIDs {
				hosts := m.hostDao.GetCmdbHostsByGroupId(gid)
				for _, h := range hosts {
					result[h.ID] = true
				}
			}
		case "physical":
			var ids []uint
			json.Unmarshal([]byte(p.PhysicalIDs), &ids)
			for _, id := range ids {
				result[id] = true
			}
		case "network":
			var ids []uint
			json.Unmarshal([]byte(p.NetworkIDs), &ids)
			for _, id := range ids {
				result[id] = true
			}
		}
	}
	return result
}

// hasPermissionForAsset 在同一条规则中同时检查：是否有指定资产类型的授权，以及是否有对应的操作权限
// 只有当同一条规则（同一个CmdbAssetPermission）同时包含assetType和action（或admin）时才返回true
func (m *CmdbAssetPermissionMiddleware) hasPermissionForAsset(perms []model.CmdbAssetPermission, assetType, action string) bool {
	for _, p := range perms {
		// 检查该规则是否包含此资产类型
		var types []string
		json.Unmarshal([]byte(p.AssetTypes), &types)
		hasType := false
		for _, t := range types {
			if t == assetType || t == "*" {
				hasType = true
				break
			}
		}
		if !hasType {
			continue
		}

		// 如果没有指定action，只要有资产类型就通过（纯GET查看）
		if action == "" {
			return true
		}

		// 检查该规则是否包含此操作权限（或admin）
		var actions []string
		json.Unmarshal([]byte(p.PermissionActions), &actions)
		for _, a := range actions {
			if a == action || a == "admin" {
				return true
			}
		}
	}
	return false
}

// actionLabel 操作中文名
func methodToAction(method string) string {
	switch method {
	case http.MethodGet:
		return "list" // GET可能用于列表或详情，都允许
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return ""
	}
}

// assetTypeLabel 资产类型中文名
func assetTypeLabel(t string) string {
	switch t {
	case "host":
		return "主机"
	case "physical":
		return "物理机"
	case "network":
		return "网络设备"
	case "database":
		return "数据库"
	case "idc":
		return "机房"
	case "cabinet":
		return "机柜"
	case "group":
		return "资产分组"
	default:
		return t
	}
}

// actionLabel 操作中文名
func actionLabel(a string) string {
	switch a {
	case "list":
		return "查看列表"
	case "get":
		return "查看详情"
	case "create":
		return "创建"
	case "update":
		return "修改"
	case "delete":
		return "删除"
	case "connect":
		return "连接"
	default:
		return a
	}
}

// AdminOnly 仅管理员可访问（用于授权管理、用户组管理、凭据管理等后台管理接口）
func (m *CmdbAssetPermissionMiddleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			result.Failed(c, http.StatusUnauthorized, "未授权")
			c.Abort()
			return
		}

		isAdmin, _ := m.groupDao.IsAdmin(admin.ID)
		if !isAdmin {
			result.Failed(c, http.StatusForbidden, "仅管理员可执行此操作")
			c.Abort()
			return
		}
		c.Next()
	}
}

func contains(arr []string, s string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}

// ParseIDList 辅助函数：从查询参数解析逗号分隔的ID列表
func ParseIDList(str string) []uint {
	if str == "" {
		return nil
	}
	parts := strings.Split(str, ",")
	var ids []uint
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if id, err := strconv.ParseUint(p, 10, 64); err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

// getEffectiveAssetIDs 获取指定类型的授权资产ID集合（用于过滤列表）
func (m *CmdbAssetPermissionMiddleware) getEffectiveAssetIDs(userID uint, assetType string) map[uint]bool {
	perms := m.getUserEffectivePermissions(userID)
	result := make(map[uint]bool)

	for _, p := range perms {
		switch assetType {
		case "host":
			var hostIDs []uint
			json.Unmarshal([]byte(p.HostIDs), &hostIDs)
			for _, id := range hostIDs {
				result[id] = true
			}
			var hgIDs []uint
			json.Unmarshal([]byte(p.HostGroupIDs), &hgIDs)
			for _, gid := range hgIDs {
				hosts := m.hostDao.GetCmdbHostsByGroupId(gid)
				for _, h := range hosts {
					result[h.ID] = true
				}
			}
		case "physical":
			var ids []uint
			json.Unmarshal([]byte(p.PhysicalIDs), &ids)
			for _, id := range ids {
				result[id] = true
			}
		case "network":
			var ids []uint
			json.Unmarshal([]byte(p.NetworkIDs), &ids)
			for _, id := range ids {
				result[id] = true
			}
		}
	}
	return result
}
