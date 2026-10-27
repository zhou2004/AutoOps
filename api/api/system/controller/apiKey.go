package controller

import (
	"dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

func ListAPIKeys(c *gin.Context) {
	i, ok := currentIdentity(c)
	if !ok {
		return
	}
	type keyView struct {
		model.APIKey
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}
	rows := []keyView{}
	query := common.GetDB().WithContext(c.Request.Context()).Table("sys_api_key k").Select("k.*, u.username, u.nickname").Joins("LEFT JOIN sys_admin u ON u.id = k.user_id")
	if !i.Admin {
		query = query.Where("k.user_id = ?", i.UserID)
	} else if c.Query("userId") != "" {
		id, err := strconv.ParseUint(c.Query("userId"), 10, 32)
		if err != nil || id == 0 {
			result.Failed(c, 400, "用户 ID 无效")
			return
		}
		query = query.Where("k.user_id = ?", id)
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	var total int64
	if query.Count(&total).Error != nil || query.Order("k.id DESC").Limit(50).Offset((page-1)*50).Scan(&rows).Error != nil {
		result.Failed(c, 500, "读取 Key 失败")
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, gin.H{"items": rows, "total": total, "pageSize": 50, "canManage": i.Admin, "permissions": i.Permissions, "modules": authorization.Catalog})
}

func APIKeyUsers(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	var users = []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Status   int    `json:"status"`
	}{}
	query := common.GetDB().Table("sys_admin").Select("id, username, nickname, status")
	if q := c.Query("q"); q != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if query.Order("id").Limit(100).Find(&users).Error != nil {
		result.Failed(c, 500, "读取用户失败")
		return
	}
	result.Success(c, users)
}

func APIKeyUserPermissions(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "用户 ID 无效")
		return
	}
	i, err := authorization.Load(c.Request.Context(), common.GetDB(), uint(id))
	if err != nil {
		result.Failed(c, 400, "账号或权限不可用")
		return
	}
	result.Success(c, i)
}

func CreateAPIKey(c *gin.Context) {
	i, ok := requirePlatformAdmin(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var body struct {
		UserID uint     `json:"userId"`
		Name   string   `json:"name"`
		Days   int      `json:"days"`
		Scopes []string `json:"scopes"`
	}
	if c.ShouldBindJSON(&body) != nil || body.UserID == 0 {
		result.Failed(c, 400, "请选择 Key 所属用户")
		return
	}
	// New keys require explicit independent module grants; resource authorization remains with the owner.
	var key model.APIKey
	var token string
	err := common.GetDB().WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var err error
		key, token, err = apikey.CreateScoped(c.Request.Context(), tx, body.UserID, body.Name, body.Scopes, body.Days)
		if err != nil {
			return err
		}
		key.IssuedBy = i.UserID
		return tx.Model(&key).Update("issued_by", i.UserID).Error
	})
	if err != nil {
		result.Failed(c, 400, "分配失败，请检查账号、名称、权限范围和有效期")
		return
	}
	c.Header("Cache-Control", "no-store")
	result.Success(c, gin.H{"key": key, "token": token})
}

func RevokeAPIKey(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, 400, "ID 无效")
		return
	}
	tx := common.GetDB().WithContext(c.Request.Context()).Model(&model.APIKey{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", time.Now().UTC())
	if tx.Error != nil {
		result.Failed(c, 500, "撤销失败")
		return
	}
	if tx.RowsAffected != 1 {
		result.Failed(c, 404, "Key 不存在或已撤销")
		return
	}
	result.Success(c, nil)
}

func APIKeyCatalog(c *gin.Context) {
	if _, ok := requirePlatformAdmin(c); !ok {
		return
	}
	result.Success(c, gin.H{"modules": authorization.Catalog})
}
