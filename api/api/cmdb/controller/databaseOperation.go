package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DatabaseOperationController struct {
	db      *gorm.DB
	service *service.DatabaseOperationService
}

func NewDatabaseOperationController(db *gorm.DB) *DatabaseOperationController {
	return &DatabaseOperationController{db: db, service: service.NewDatabaseOperationService(db)}
}

func databaseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, http.StatusBadRequest, "数据源 ID 无效")
		return 0, false
	}
	return uint(id), true
}

func (ctrl *DatabaseOperationController) audited(c *gin.Context, databaseID uint, action, target string, fn func() (any, error)) {
	userID, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, http.StatusUnauthorized, "请先登录")
		return
	}
	var database model.CmdbSQL
	if err := ctrl.db.Select("account_id").First(&database, databaseID).Error; err != nil {
		result.Failed(c, http.StatusNotFound, "数据源不存在")
		return
	}
	started := time.Now()
	data, operationErr := fn()
	finished := time.Now().UTC()
	message, status := "", "success"
	if operationErr != nil {
		message, status = operationErr.Error(), "failed"
	}
	if len(target) > 255 {
		target = target[:255]
	}
	audit := model.CmdbDatabaseAudit{DatabaseID: databaseID, UserID: userID, Action: action, Target: target, Status: status, Message: message, ClientIP: c.ClientIP(), DurationMS: time.Since(started).Milliseconds(), CreatedAt: started.UTC(), FinishedAt: &finished}
	if err := ctrl.db.Create(&audit).Error; err != nil {
		result.Failed(c, http.StatusInternalServerError, "数据操作审计写入失败")
		return
	}
	if operationErr != nil {
		result.Failed(c, http.StatusBadRequest, operationErr.Error())
		return
	}
	result.Success(c, data)
}

func (ctrl *DatabaseOperationController) Overview(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	ctrl.audited(c, id, "test", "connection", func() (any, error) { return ctrl.service.Overview(c.Request.Context(), id) })
}

func (ctrl *DatabaseOperationController) Catalog(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	ctrl.audited(c, id, "catalog", "databases", func() (any, error) {
		items, err := ctrl.service.Catalog(c.Request.Context(), id)
		return gin.H{"list": items}, err
	})
}

func (ctrl *DatabaseOperationController) Tables(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	database := strings.TrimSpace(c.Query("database"))
	ctrl.audited(c, id, "tables", database, func() (any, error) {
		items, err := ctrl.service.Tables(c.Request.Context(), id, database)
		return gin.H{"list": items}, err
	})
}

type databaseSQLRequest struct {
	Database    string `json:"database"`
	SQL         string `json:"sql" binding:"required"`
	ConfirmName string `json:"confirmName"`
}

func (ctrl *DatabaseOperationController) Query(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	var req databaseSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "SQL 请求参数无效")
		return
	}
	operation, _ := service.DetectSQLOperation(req.SQL)
	if operation == "" {
		operation = "QUERY"
	}
	ctrl.audited(c, id, "query", strings.TrimSpace(req.Database)+"/"+operation, func() (any, error) {
		output, _, err := ctrl.service.Query(c.Request.Context(), id, req.Database, req.SQL)
		return output, err
	})
}

func (ctrl *DatabaseOperationController) Execute(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	var req databaseSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "SQL 请求参数无效")
		return
	}
	operation, _ := service.DetectSQLOperation(req.SQL)
	if operation == "" {
		operation = "WRITE"
	}
	ctrl.audited(c, id, "execute", strings.TrimSpace(req.Database)+"/"+operation, func() (any, error) {
		output, _, err := ctrl.service.Execute(c.Request.Context(), id, req.Database, req.SQL, req.ConfirmName)
		return output, err
	})
}

func (ctrl *DatabaseOperationController) RedisKeys(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	cursor, _ := strconv.ParseUint(c.DefaultQuery("cursor", "0"), 10, 64)
	count, _ := strconv.ParseInt(c.DefaultQuery("count", "100"), 10, 64)
	match := c.DefaultQuery("match", "*")
	ctrl.audited(c, id, "redis_scan", match, func() (any, error) {
		items, next, err := ctrl.service.RedisKeys(c.Request.Context(), id, cursor, match, count)
		return gin.H{"list": items, "nextCursor": next}, err
	})
}

func (ctrl *DatabaseOperationController) RedisValue(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	key := c.Query("key")
	ctrl.audited(c, id, "redis_get", key, func() (any, error) { return ctrl.service.RedisValue(c.Request.Context(), id, key) })
}

func (ctrl *DatabaseOperationController) SetRedisValue(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	var req struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value"`
		TTL         int64  `json:"ttl"`
		ConfirmName string `json:"confirmName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "Redis 写入参数无效")
		return
	}
	ctrl.audited(c, id, "redis_set", req.Key, func() (any, error) {
		err := ctrl.service.SetRedisString(c.Request.Context(), id, req.Key, req.Value, req.TTL, req.ConfirmName)
		return true, err
	})
}

func (ctrl *DatabaseOperationController) DeleteRedisValue(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	var req struct {
		Key        string `json:"key" binding:"required"`
		ConfirmKey string `json:"confirmKey" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "Redis 删除参数无效")
		return
	}
	ctrl.audited(c, id, "redis_delete", req.Key, func() (any, error) {
		deleted, err := ctrl.service.DeleteRedisKey(c.Request.Context(), id, req.Key, req.ConfirmKey)
		return gin.H{"deleted": deleted}, err
	})
}

func (ctrl *DatabaseOperationController) Audits(c *gin.Context) {
	id, ok := databaseID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	var list []model.CmdbDatabaseAudit
	var total int64
	query := ctrl.db.Model(&model.CmdbDatabaseAudit{}).Where("database_id = ?", id)
	if query.Count(&total).Error != nil || query.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error != nil {
		result.Failed(c, http.StatusInternalServerError, "读取数据操作审计失败")
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}
