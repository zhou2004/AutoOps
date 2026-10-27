package controller

import (
	"dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
	"net/http"
	"strconv"
)

func currentIdentity(c *gin.Context) (authorization.Identity, bool) {
	user, err := jwt.GetAdminId(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "请先登录"})
		return authorization.Identity{}, false
	}
	i, err := apikey.EffectiveIdentity(c.Request.Context(), common.GetDB(), user)
	if err != nil {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "账号或权限不可用"})
		return i, false
	}
	return i, true
}

func requirePlatformAdmin(c *gin.Context) (authorization.Identity, bool) {
	i, ok := currentIdentity(c)
	if ok && !i.Admin {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "仅管理员可分配权限和 API Key"})
		ok = false
	}
	return i, ok
}

func MyPermissions(c *gin.Context) {
	i, ok := currentIdentity(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, i)
}

func PermissionCatalog(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	roles := []model.SysRole{}
	if common.GetDB().Order("id").Find(&roles).Error != nil {
		result.Failed(c, 500, "读取角色失败")
		return
	}
	result.Success(c, gin.H{"modules": authorization.Catalog, "roles": roles})
}

func RolePermissions(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "角色 ID 无效")
		return
	}
	var role model.SysRole
	if common.GetDB().First(&role, id).Error != nil {
		result.Failed(c, 404, "角色不存在")
		return
	}
	grants, legacy, err := authorization.ForRole(c.Request.Context(), common.GetDB(), uint(id))
	if err != nil {
		result.Failed(c, 500, "读取权限失败")
		return
	}
	result.Success(c, gin.H{"permissions": grants, "legacy": legacy, "isAdmin": role.RoleKey == "admin"})
}

func SaveRolePermissions(c *gin.Context) {
	i, ok := requirePlatformAdmin(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "角色 ID 无效")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	var body struct {
		Permissions *[]string `json:"permissions"`
	}
	if c.ShouldBindJSON(&body) != nil || body.Permissions == nil {
		result.Failed(c, 400, "请提供权限数组，空数组表示撤销全部模块权限")
		return
	}
	var role model.SysRole
	if common.GetDB().First(&role, id).Error != nil {
		result.Failed(c, 404, "角色不存在")
		return
	}
	if role.RoleKey == "admin" {
		result.Failed(c, 400, "管理员角色拥有全部权限，不能通过此界面修改")
		return
	}
	set := map[string]bool{}
	for _, p := range *body.Permissions {
		if !authorization.Known(p) || set[p] {
			result.Failed(c, 400, "权限不存在或重复")
			return
		}
		set[p] = true
	}
	encoded, _ := json.Marshal(*body.Permissions)
	row := model.RolePermission{RoleID: uint(id), Permissions: string(encoded), UpdatedBy: i.UserID}
	if common.GetDB().WithContext(c.Request.Context()).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "role_id"}}, DoUpdates: clause.AssignmentColumns([]string{"permissions", "updated_by", "updated_at"})}).Create(&row).Error != nil {
		result.Failed(c, 500, "保存权限失败")
		return
	}
	result.Success(c, true)
}
