package controller

import (
	"net/http"
	"strings"

	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

// ComposeList 列出 Compose 项目。
func (ctrl *DockerOperationController) ComposeList(c *gin.Context) {
	ctrl.audited(c, "docker_view", "compose", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.ComposeList(c.Request.Context(), kind, id, connection)
		return gin.H{"list": items}, err
	})
}

// ComposeGet 读取项目 yml 配置。
func (ctrl *DockerOperationController) ComposeGet(c *gin.Context) {
	project := c.Param("project")
	ctrl.audited(c, "docker_view", "compose/"+project, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		return ctrl.service.ComposeGet(c.Request.Context(), kind, id, connection, project)
	})
}

// ComposeSave 保存 yml 配置并可选启动。
func (ctrl *DockerOperationController) ComposeSave(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Project      string `json:"project" binding:"required"`
		Content      string `json:"content" binding:"required"`
		Up           bool   `json:"up"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "Compose 配置参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "compose/"+req.Project, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ComposeSave(c.Request.Context(), kind, id, connection, req.Project, req.Content, req.Up)
		return gin.H{"output": out}, err
	})
}

// ComposeAction 执行 Compose 生命周期操作。
func (ctrl *DockerOperationController) ComposeAction(c *gin.Context) {
	project := c.Param("project")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Action       string `json:"action" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "Compose 操作参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "compose/"+project+"/"+req.Action, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ComposeAction(c.Request.Context(), kind, id, connection, project, req.Action)
		return gin.H{"output": strings.TrimSpace(out)}, err
	})
}

// ComposeRemove 删除 Compose 项目。
func (ctrl *DockerOperationController) ComposeRemove(c *gin.Context) {
	project := c.Param("project")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		ConfirmName  string `json:"confirmName" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "Compose 删除参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "compose/"+project, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ComposeRemove(c.Request.Context(), kind, id, connection, project, req.ConfirmName)
		return gin.H{"output": out}, err
	})
}
