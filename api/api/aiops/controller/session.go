package controller

import (
	"context"
	model "dodevops-api/api/aiops/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/result"
	"dodevops-api/middleware"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func keyAccess(g *gin.Context) (uint, bool) {
	operate := false
	if value, ok := g.Get(middleware.IdentityContextKey); ok {
		operate = value.(authorization.Identity).Allows("aiops:operate")
	}
	if value, ok := g.Get(apikey.ContextKey); ok {
		key := value.(systemmodel.APIKey)
		return key.ID, operate && apikey.Has(key, "aiops:operate")
	}
	return 0, operate
}
func (c *Controller) Targets(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	rows, err := c.Service.ToolTargets(g.Request.Context(), user)
	if err != nil {
		result.Failed(g, 500, "读取授权资产失败")
		return
	}
	result.Success(g, rows)
}
func (c *Controller) Sessions(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	rows := []model.Session{}
	if c.Service.DB.WithContext(g.Request.Context()).Where("user_id = ?", user).Order("updated_at DESC").Limit(100).Find(&rows).Error != nil {
		result.Failed(g, 500, "读取会话失败")
		return
	}
	result.Success(g, rows)
}
func (c *Controller) CreateSession(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 8192)
	var r model.SessionRequest
	if g.ShouldBindJSON(&r) != nil {
		result.Failed(g, 400, "请求无效")
		return
	}
	row, err := c.Service.CreateSession(g.Request.Context(), user, r)
	if err != nil {
		result.Failed(g, 400, err.Error())
		return
	}
	result.Success(g, row)
}
func (c *Controller) SessionDetail(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	row, err := c.Service.Session(g.Request.Context(), user, id)
	if err != nil {
		result.Failed(g, 404, err.Error())
		return
	}
	messages := []model.Message{}
	calls := []model.ToolCall{}
	db := c.Service.DB.WithContext(g.Request.Context())
	if db.Where("session_id = ?", id).Order("id DESC").Limit(200).Find(&messages).Error != nil || db.Where("session_id = ?", id).Order("id DESC").Limit(200).Find(&calls).Error != nil {
		result.Failed(g, 500, "读取会话内容失败")
		return
	}
	for i := range messages {
		if messages[i].Status == "running" && (row.BusyUntil == nil || row.BusyUntil.Before(time.Now())) {
			messages[i].Status = "interrupted"
		}
	}
	result.Success(g, gin.H{"session": row, "messages": messages, "calls": calls})
}
func (c *Controller) Grant(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	_, operate := keyAccess(g)
	if !operate {
		result.Failed(g, 403, "API Key 未授予操作权限")
		return
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 8192)
	var r model.GrantRequest
	if g.ShouldBindJSON(&r) != nil {
		result.Failed(g, 400, "授权参数无效")
		return
	}
	row, err := c.Service.Grant(g.Request.Context(), user, id, r)
	if err != nil {
		result.Failed(g, 400, err.Error())
		return
	}
	result.Success(g, row)
}
func (c *Controller) Chat(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 32<<10)
	var r struct {
		Prompt string `json:"prompt"`
	}
	if g.ShouldBindJSON(&r) != nil {
		result.Failed(g, 400, "请输入问题")
		return
	}
	keyID, operate := keyAccess(g)
	clusters := middleware.GetUserAllowedClusterIDsFromContext(g)
	c.stream(g, func(ctx context.Context, emit func(model.Event) error) error {
		return c.Service.Chat(ctx, user, id, keyID, operate, clusters, r.Prompt, emit)
	})
}
func (c *Controller) Tool(g *gin.Context) {
	user, ok := c.identity(g)
	if !ok {
		return
	}
	id, ok := parseID(g)
	if !ok {
		return
	}
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 8192)
	var r model.ToolRequest
	if g.ShouldBindJSON(&r) != nil {
		result.Failed(g, 400, "工具请求无效")
		return
	}
	_, operate := keyAccess(g)
	ctx, cancel := context.WithTimeout(g.Request.Context(), 90*time.Second)
	defer cancel()
	call, err := c.Service.ExecuteTool(ctx, user, id, operate, r)
	if err != nil {
		result.Failed(g, 403, err.Error())
		return
	}
	result.Success(g, call)
}

// This public route accepts only a short-lived per-turn callback ticket.
func (c *Controller) HarnessTool(g *gin.Context) {
	g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, 8192)
	var r model.ToolRequest
	if g.ShouldBindJSON(&r) != nil {
		g.AbortWithStatusJSON(400, gin.H{"error": "工具请求无效"})
		return
	}
	ctx, cancel := context.WithTimeout(g.Request.Context(), 90*time.Second)
	defer cancel()
	call, err := c.Service.HarnessTool(ctx, strings.TrimPrefix(g.GetHeader("Authorization"), "Bearer "), r)
	if err != nil {
		g.AbortWithStatusJSON(403, gin.H{"error": err.Error()})
		return
	}
	g.JSON(200, call)
}
