package database

import (
	"dodevops-api/api/database/controller"
	"dodevops-api/common"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	c := controller.New(common.GetDB())
	g := r.Group("/database")
	g.GET("/groups", c.Groups)
	g.GET("/audit-policy", c.AuditPolicy)
	g.PUT("/audit-policy", c.SaveAuditPolicy)
	g.DELETE("/audit/:kind/:id", c.DeleteAudit)
	g.GET("/repositories", c.Repositories)
	g.POST("/repositories", c.SaveRepository)
	g.POST("/repositories/:id/check", c.CheckRepository)
	g.GET("/clusters", c.List)
	g.POST("/clusters", c.Create)
	g.POST("/discovery", c.Discover)
	g.GET("/clusters/:id", c.Detail)
	g.GET("/clusters/:id/topology", c.Detail)
	g.GET("/clusters/:id/members", c.Members)
	g.GET("/clusters/:id/capabilities", c.Capabilities)
	g.POST("/clusters/:id/operation-plans", c.Plan)
	g.POST("/clusters/:id/operations", c.Submit)
	g.GET("/operations/:id", c.Operation)
	g.GET("/storage", c.Storage)
	g.GET("/storage-thresholds", c.StorageThreshold)
	g.POST("/storage-thresholds", c.SaveStorageThreshold)
	g.GET("/clusters/:id/configurations", c.Configurations)
	g.POST("/backup-policies", c.SavePolicy)
	g.GET("/backups/:id", c.Backup)
	g.PATCH("/backup-policies/:id", c.SavePolicy)
	for _, kind := range []string{"operations", "events", "backups", "backup-policies", "config-revisions"} {
		g.GET("/"+kind, c.Records(kind))
	}
}
