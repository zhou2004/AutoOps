package middleware

import (
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"strings"
)

const IdentityContextKey = "autoops.identity"

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Some legacy groups register authentication twice.
		if _, ok := c.Get(IdentityContextKey); ok {
			c.Next()
			return
		}
		token := ""
		if header := c.GetHeader("Authorization"); header != "" {
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				denyAuth(c, 401, "认证格式错误")
				return
			}
			token = parts[1]
		} else {
			token = c.Query("token") // SSE, downloads and browser WebSocket clients.
			if token == "" && strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
				token = strings.TrimSpace(strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",")[0])
			}
		}
		if token == "" {
			denyAuth(c, 401, "请先登录或提供 API Key")
			return
		}
		var user uint
		var key *systemmodel.APIKey
		if strings.HasPrefix(token, "aops_") {
			k, err := apikey.Authenticate(c.Request.Context(), common.GetDB(), token)
			if err != nil {
				denyAuth(c, 401, "API Key 无效、已过期或已撤销")
				return
			}
			key = &k
			user = k.UserID
		} else {
			claims, err := jwt.ValidateToken(token)
			if err != nil {
				denyAuth(c, 401, "登录会话无效")
				return
			}
			user = claims.ID
		}
		identity, err := authorization.Load(c.Request.Context(), common.GetDB(), user)
		if err != nil {
			denyAuth(c, 401, "账号不可用或无法验证权限")
			return
		}
		required := authorization.Route(c.Request.Method, c.FullPath())
		ownerAdmin := identity.Admin
		if key != nil {
			identity, err = apikey.Restrict(identity, *key)
			if err != nil {
				denyAuth(c, 403, "API Key 权限范围无效")
				return
			}
			if key.GrantMode == apikey.Scoped {
				// Personal password/profile writes are not module grants; keys cannot change login credentials.
				if required.Self && c.Request.Method != "GET" && c.Request.Method != "HEAD" {
					denyAuth(c, 403, "API Key 不能修改个人登录信息")
					return
				}
			} else if !apikey.Has(*key, apikey.Inherit) {
				scope := apikey.RouteScope(c.Request.Method, c.FullPath())
				if scope == "" || !apikey.Has(*key, scope) {
					denyAuth(c, 403, "此旧版 API Key 未授权该接口，请联系管理员重新分配 Key")
					return
				}
			}
			c.Set(apikey.ContextKey, *key)
			c.Request = c.Request.WithContext(apikey.WithKey(c.Request.Context(), key.ID))
		}
		if !required.Self && (required.Permission == "" || required.AdminOnly && !ownerAdmin || !identity.Allows(required.Permission)) {
			denyAuth(c, 403, "无权执行此操作，请联系管理员分配权限")
			return
		}
		if !checkHostTargets(c, identity) {
			return
		}
		c.Set(IdentityContextKey, identity)
		c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: user, Username: identity.Username})
		c.Next()
	}
}

func denyAuth(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"code": status, "message": message})
}
