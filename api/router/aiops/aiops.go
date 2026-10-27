package aiops

import (
	"dodevops-api/api/aiops/controller"
	"dodevops-api/api/aiops/service"
	"dodevops-api/common"
	"dodevops-api/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup) {
	c := &controller.Controller{Service: service.New(common.GetDB())}
	g := router.Group("/aiops")
	g.GET("/catalog", c.Catalog)
	g.GET("/targets", c.Targets)
	g.GET("/sessions", c.Sessions)
	g.POST("/sessions", c.CreateSession)
	g.GET("/sessions/:id", c.SessionDetail)
	g.PUT("/sessions/:id/grants", c.Grant)
	g.POST("/sessions/:id/messages", middleware.NewK8sPermissionMiddleware().FilterAllowedClusters(), c.Chat)
	g.POST("/sessions/:id/tools", c.Tool)
	g.POST("/configs", c.Save)
	g.PUT("/configs/:id", c.Save)
	g.DELETE("/configs/:id", c.Delete)
	g.POST("/models/:id/test", c.Test)
	g.GET("/runs", c.History)
	g.GET("/runs/:id", c.Detail)
	g.POST("/diagnose", middleware.NewK8sPermissionMiddleware().FilterAllowedClusters(), c.Stream)
}

func RegisterHarnessCallback(router *gin.RouterGroup) {
	c := &controller.Controller{Service: service.New(common.GetDB())}
	router.POST("/aiops/harness/tools", c.HarnessTool)
}
