package controller

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func RackTopologyHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			result.Failed(c, http.StatusBadRequest, "无效机房 ID")
			return
		}
		var scopes map[string]map[uint]bool
		if !c.GetBool("cmdb_is_admin") {
			user, err := jwt.GetAdmin(c)
			if err != nil {
				result.Failed(c, http.StatusUnauthorized, "未授权")
				return
			}
			permissions := dao.NewCmdbAssetPermissionDao(db)
			rules, err := permissions.EffectivePermissions(user.ID)
			if err != nil {
				result.Failed(c, 500, "权限查询失败")
				return
			}
			scopes = map[string]map[uint]bool{}
			for _, kind := range []string{"cabinet", "physical", "network"} {
				ids, err := permissions.AllowedAssetIDs(rules, kind, "get")
				if err != nil {
					result.Failed(c, 500, "权限查询失败")
					return
				}
				if ids == nil {
					ids = map[uint]bool{}
				}
				scopes[kind] = ids
			}
		}
		data, err := service.LoadRackTopology(db, uint(id), scopes)
		if err != nil {
			result.Failed(c, 500, "无法读取机房布局")
			return
		}
		result.Success(c, data)
	}
}
