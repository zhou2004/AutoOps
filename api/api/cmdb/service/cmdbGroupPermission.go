package service

import (
	"dodevops-api/api/cmdb/model"
	"github.com/gin-gonic/gin"
)

func filterAllowedGroups(c *gin.Context, groups []model.CmdbGroup) []model.CmdbGroup {
	scope := assetIDScope(c)
	if scope == nil {
		return groups
	}
	filtered := make([]model.CmdbGroup, 0)
	for _, group := range groups {
		if !scope[group.ID] {
			continue
		}
		group.Hosts = nil
		group.Children = nil
		if !scope[group.ParentID] {
			group.ParentID = 0
		}
		filtered = append(filtered, group)
	}
	return filtered
}

func filterGroupHosts(c *gin.Context, hosts []model.CmdbHost) []model.CmdbHost {
	value, exists := c.Get("cmdb_allowed_host_ids")
	if !exists {
		return hosts
	}
	scope, ok := value.(map[uint]bool)
	filtered := make([]model.CmdbHost, 0)
	if !ok {
		return filtered
	}
	for _, host := range hosts {
		if scope[host.ID] {
			filtered = append(filtered, host)
		}
	}
	return filtered
}
