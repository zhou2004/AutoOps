package webservice

import (
	"dodevops-api/api/webservice/controller"
	"dodevops-api/api/webservice/model"
	"dodevops-api/common"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	c := controller.New(common.GetDB())
	g := r.Group("/web")
	for kind := range model.Tables {
		g.GET("/"+kind, c.List(kind))
		g.GET("/"+kind+"/:id", c.Detail(kind))
		if kind != "instances" {
			g.POST("/"+kind, c.Save(kind))
			g.PUT("/"+kind+"/:id", c.Save(kind))
			g.DELETE("/"+kind+"/:id", c.Delete(kind))
		}
	}
	g.GET("/groups", c.Groups)
	g.GET("/environments", c.Environments)
	g.GET("/targets", c.Targets)
	g.POST("/source-discovery", c.SourceDiscovery)
	g.GET("/clusters/:id/topology", c.Topology)
	g.GET("/traffic", c.Traffic)
	g.GET("/clusters/:id/traffic-policy", c.TrafficPolicy)
	g.POST("/clusters/:id/traffic-policy/preview", c.PreviewTrafficPolicy)
	g.POST("/clusters/:id/traffic-policy/draft", c.DraftTrafficPolicy)
	g.GET("/dns-credentials", c.DNSCredentials)
	g.GET("/dns-accounts/:id/zones", c.DNSAccountZones)
	g.POST("/dns-accounts/:id/check", c.CheckDNSAccount)
	g.POST("/sources/:id/check", c.Discover)
	g.POST("/sources/:id/discovery", c.Discover)
	g.GET("/clusters/:id/members", c.Members)
	g.POST("/clusters/:id/members", c.Members)
	g.GET("/clusters/:id/capabilities", c.Capabilities)
	g.POST("/operation-plans", c.Plan)
	g.GET("/operation-plans/:id/preview", c.PlanPreview)
	g.POST("/operations", c.Submit)
	g.GET("/operations", c.Audit)
	g.GET("/operations/:id", c.Operation)
	g.POST("/operations/:id/cancel", c.Cancel)
	g.GET("/operations/:id/events", c.OperationEvents)
	g.GET("/audit", c.Audit)
	g.GET("/audit/events", c.Events)
	g.GET("/worker-status", c.WorkerStatus)
	g.GET("/config-revisions/:id/diff", c.RevisionDiff)
	g.POST("/certificates/:id/import", c.ImportCertificate)
	g.POST("/certificates/import", c.ImportNewCertificate)
	g.GET("/certificates/:id/versions", c.CertificateVersions)
	g.GET("/certificates/:id/versions/:versionId/export", c.ExportCertificate)
	g.GET("/instances/:id/config", c.InstanceConfig)
	g.POST("/instances/:id/config/validate", c.ValidateInstanceConfig)
	g.POST("/clusters/:id/config/validate", c.ValidateClusterConfig)
	g.GET("/config-revisions", c.Revisions)
	g.GET("/sites/:id/render", c.Render("sites"))
	g.GET("/upstreams/:id/render", c.Render("upstreams"))
	g.GET("/instances/:id/logs", c.Logs)
	g.GET("/instances/:id/logs/stream", c.LogStream)
}
func RegisterTerminal(r *gin.RouterGroup) {
	c := controller.New(common.GetDB())
	r.GET("/web/terminal/connect", c.Terminal)
}
