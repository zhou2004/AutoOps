package controller

import (
	"net/http"
	"strings"

	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

// NetworkCreate 创建网络。
func (ctrl *DockerOperationController) NetworkCreate(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Name         string `json:"name" binding:"required"`
		Driver       string `json:"driver"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "网络参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "networks/"+req.Name, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.NetworkCreate(c.Request.Context(), kind, id, connection, req.Name, req.Driver)
		return gin.H{"output": out}, err
	})
}

// NetworkInspect 查看网络详情及连接容器。
func (ctrl *DockerOperationController) NetworkInspect(c *gin.Context) {
	network := c.Param("network")
	ctrl.audited(c, "docker_view", "networks/"+network, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		info, err := ctrl.service.NetworkInspect(c.Request.Context(), kind, id, connection, network)
		if err != nil {
			return nil, err
		}
		containers, cerr := ctrl.service.NetworkContainers(c.Request.Context(), kind, id, connection, network)
		return gin.H{"info": info, "containers": containers}, cerr
	})
}

// NetworkRemove 删除网络。
func (ctrl *DockerOperationController) NetworkRemove(c *gin.Context) {
	network := c.Param("network")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		ConfirmName  string `json:"confirmName" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "网络删除参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "networks/"+network, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.NetworkRemove(c.Request.Context(), kind, id, connection, network, req.ConfirmName)
		return gin.H{"output": strings.TrimSpace(out)}, err
	})
}

// NetworkConnect 将容器连接到网络。
func (ctrl *DockerOperationController) NetworkConnect(c *gin.Context) {
	network := c.Param("network")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Container    string `json:"container" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "网络连接参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "networks/"+network+"/connect", req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.NetworkConnect(c.Request.Context(), kind, id, connection, network, req.Container)
		return gin.H{"output": out}, err
	})
}

// NetworkDisconnect 将容器从网络断开。
func (ctrl *DockerOperationController) NetworkDisconnect(c *gin.Context) {
	network := c.Param("network")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Container    string `json:"container" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "网络断开参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", "networks/"+network+"/disconnect", req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.NetworkDisconnect(c.Request.Context(), kind, id, connection, network, req.Container)
		return gin.H{"output": out}, err
	})
}
