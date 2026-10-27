package controller

import (
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
)

// Targets exposes only Docker-authorized assets. Container operators do not need
// CMDB inventory permissions merely to discover their Docker connections.
func (ctrl *DockerOperationController) Targets(c *gin.Context) {
	uid, err := jwt.GetAdminId(c)
	if err != nil || uid == 0 {
		result.Failed(c, 401, "请先登录")
		return
	}
	active, err := ctrl.permissions.UserActive(uid)
	if err != nil || !active {
		result.Failed(c, 403, "账户不存在或已停用")
		return
	}
	admin, err := ctrl.permissions.IsAdministrator(uid)
	if err != nil {
		result.Failed(c, 500, "权限校验失败")
		return
	}
	perms, err := ctrl.permissions.EffectivePermissions(uid)
	if err != nil {
		result.Failed(c, 500, "权限校验失败")
		return
	}
	type target struct {
		ID       uint   `json:"id"`
		HostName string `json:"hostName"`
	}
	output := gin.H{}
	for _, kind := range []string{"host", "physical"} {
		table, key := "cmdb_host", "hosts"
		if kind == "physical" {
			table, key = "cmdb_physical_machine", "physical"
		}
		query := ctrl.db.Table(table).Select("id", "host_name")
		if !admin {
			ids, e := ctrl.permissions.AllowedAssetIDs(perms, kind, "docker_view")
			if e != nil {
				result.Failed(c, 500, "权限校验失败")
				return
			}
			allowed := make([]uint, 0, len(ids))
			for id := range ids {
				allowed = append(allowed, id)
			}
			query = query.Where("id IN ?", allowed)
		}
		query = query.Where("id IN (?)", ctrl.db.Model(&model.CmdbAssetConnection{}).Select("asset_id").Where("asset_type = ? AND protocol = ? AND enabled = ?", kind, "ssh", true))
		items := []target{}
		if err := query.Order("id ASC").Find(&items).Error; err != nil {
			result.Failed(c, 500, "读取 Docker 资产失败")
			return
		}
		output[key] = items
	}
	result.Success(c, output)
}
