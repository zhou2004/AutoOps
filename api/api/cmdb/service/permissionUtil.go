package service

import "github.com/gin-gonic/gin"

// assetIDScope distinguishes administrator requests (no scope) from a denied empty scope.
func assetIDScope(c *gin.Context) map[uint]bool {
	value, exists := c.Get("cmdb_allowed_asset_ids")
	if !exists {
		return nil
	}
	ids, ok := value.(map[uint]bool)
	if !ok || ids == nil {
		return map[uint]bool{}
	}
	return ids
}

func filterAllowedIDs(c *gin.Context, ids []uint) []uint {
	scope := assetIDScope(c)
	if scope == nil {
		return ids
	}
	filtered := make([]uint, 0)
	for _, id := range ids {
		if scope[id] {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func isAdminContext(c *gin.Context) bool { return assetIDScope(c) == nil }

// AssetIDScope allows controllers with context-free services to pass the same scope.
func AssetIDScope(c *gin.Context) map[uint]bool { return assetIDScope(c) }
