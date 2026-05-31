package service

import (
	"github.com/gin-gonic/gin"
)

// filterAllowedIDs 从上下文中获取 cmdb_allowed_asset_ids，过滤ID列表
// 管理员（或没有设置该context时）返回全部
func filterAllowedIDs(c *gin.Context, ids []uint) []uint {
	allowedIDs, exists := c.Get("cmdb_allowed_asset_ids")
	if !exists {
		return ids
	}
	idSet, ok := allowedIDs.(map[uint]bool)
	if !ok || len(idSet) == 0 {
		return ids
	}
	var filtered []uint
	for _, id := range ids {
		if idSet[id] {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

// isAdminContext 检查当前请求是否为管理员（跳过资产ID过滤的条件）
func isAdminContext(c *gin.Context) bool {
	allowedIDs, exists := c.Get("cmdb_allowed_asset_ids")
	if !exists {
		return true
	}
	idSet, ok := allowedIDs.(map[uint]bool)
	if !ok || len(idSet) == 0 {
		return true
	}
	return false
}
