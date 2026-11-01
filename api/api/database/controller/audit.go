package controller

import (
	"dodevops-api/common/result"
	"errors"
	"github.com/gin-gonic/gin"
)

func (ctrl *Controller) AuditPolicy(c *gin.Context) {
	if _, ok := user(c); !ok {
		return
	}
	result.Success(c, ctrl.s.AuditPolicy())
}
func (ctrl *Controller) SaveAuditPolicy(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var p struct {
		RetentionDays int `json:"retentionDays"`
	}
	if c.ShouldBindJSON(&p) != nil {
		fail(c, errors.New("保存周期无效"))
		return
	}
	r, e := ctrl.s.SaveAuditPolicy(c.Request.Context(), u, p.RetentionDays)
	if e != nil {
		fail(c, e)
		return
	}
	result.Success(c, r)
}
func (ctrl *Controller) DeleteAudit(c *gin.Context) {
	u, ok := user(c)
	if !ok {
		return
	}
	var p struct {
		Confirmed bool `json:"confirmed"`
	}
	if c.ShouldBindJSON(&p) != nil {
		fail(c, errors.New("请确认删除"))
		return
	}
	if e := ctrl.s.DeleteAudit(c.Request.Context(), u, c.Param("kind"), id(c), p.Confirmed); e != nil {
		fail(c, e)
		return
	}
	result.Success(c, nil)
}
