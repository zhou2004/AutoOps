package controller

import (
	"net/http"
	"strings"

	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

// VolumeCreate 创建数据卷。
func (ctrl *DockerOperationController) VolumeCreate(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Name         string `json:"name" binding:"required"`
		Driver       string `json:"driver"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "数据卷参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "volumes/"+req.Name, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.VolumeCreate(c.Request.Context(), kind, id, connection, req.Name, req.Driver)
		return gin.H{"output": out}, err
	})
}

// VolumeInspect 查看数据卷详情及映射容器。
func (ctrl *DockerOperationController) VolumeInspect(c *gin.Context) {
	volume := c.Param("volume")
	ctrl.audited(c, "docker_view", "volumes/"+volume, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		info, err := ctrl.service.VolumeInspect(c.Request.Context(), kind, id, connection, volume)
		if err != nil {
			return nil, err
		}
		containers, cerr := ctrl.service.VolumeContainers(c.Request.Context(), kind, id, connection, volume)
		return gin.H{"info": info, "containers": containers}, cerr
	})
}

// VolumeRemove 删除数据卷。
func (ctrl *DockerOperationController) VolumeRemove(c *gin.Context) {
	volume := c.Param("volume")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		ConfirmName  string `json:"confirmName" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "数据卷删除参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "volumes/"+volume, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.VolumeRemove(c.Request.Context(), kind, id, connection, volume, req.ConfirmName)
		return gin.H{"output": strings.TrimSpace(out)}, err
	})
}

// VolumeContainers 查看使用数据卷的容器。
func (ctrl *DockerOperationController) VolumeContainers(c *gin.Context) {
	volume := c.Param("volume")
	ctrl.audited(c, "docker_view", "volumes/"+volume, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.VolumeContainers(c.Request.Context(), kind, id, connection, volume)
		return gin.H{"list": items}, err
	})
}
