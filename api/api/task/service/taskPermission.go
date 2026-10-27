package service

import (
	cmdbdao "dodevops-api/api/cmdb/dao"
	"dodevops-api/common"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func AuthorizeTaskTargets(c *gin.Context, groups map[string][]uint) bool {
	user, err := jwt.GetAdminId(c)
	if err != nil {
		c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "未认证"})
		return false
	}
	d := cmdbdao.NewCmdbAssetPermissionDao(common.GetDB().WithContext(c.Request.Context()))
	if admin, err := d.IsAdministrator(user); err == nil && admin {
		return true
	}
	count := 0
	for _, ids := range groups {
		for _, id := range ids {
			count++
			allowed, err := d.CanUserAccess(user, "host", id, "connect_ssh")
			if err != nil || !allowed {
				c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "没有任务目标主机的连接权限"})
				return false
			}
			var host struct{ SSHKeyID uint }
			if common.GetDB().Table("cmdb_host").Select("ssh_key_id").Where("id = ?", id).Take(&host).Error != nil {
				return denyTaskCredential(c)
			}
			if host.SSHKeyID == 0 {
				return denyTaskCredential(c)
			}
		}
	}
	if count == 0 {
		c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "任务必须包含已授权的主机"})
		return false
	}
	return true
}
func denyTaskCredential(c *gin.Context) bool {
	c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": "任务目标主机无效或尚未配置 SSH 连接"})
	return false
}
func AuthorizeTaskHostString(c *gin.Context, hosts string) bool {
	ids := []uint{}
	if strings.HasPrefix(strings.TrimSpace(hosts), "[") {
		if json.Unmarshal([]byte(hosts), &ids) != nil {
			return denyTaskCredential(c)
		}
	} else {
		for _, value := range strings.Split(hosts, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
			if err != nil || id == 0 {
				return denyTaskCredential(c)
			}
			ids = append(ids, uint(id))
		}
	}
	return AuthorizeTaskTargets(c, map[string][]uint{"hosts": ids})
}
