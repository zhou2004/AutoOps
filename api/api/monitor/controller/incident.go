package controller

import (
	"dodevops-api/api/monitor/service"
	"github.com/gin-gonic/gin"
)

type IncidentController struct {
	incidentService service.IncidentServiceInterface
}

func NewIncidentController() *IncidentController {
	return &IncidentController{
		incidentService: service.NewIncidentService(),
	}
}

func (c *IncidentController) GetList(ctx *gin.Context) {
	c.incidentService.GetList(ctx)
}

func (c *IncidentController) GetStats(ctx *gin.Context) {
	c.incidentService.GetStats(ctx)
}

func (c *IncidentController) Resolve(ctx *gin.Context) {
	c.incidentService.Resolve(ctx)
}

func (c *IncidentController) Delete(ctx *gin.Context) {
	c.incidentService.Delete(ctx)
}