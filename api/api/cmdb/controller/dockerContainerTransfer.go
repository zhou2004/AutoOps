package controller

import (
	"fmt"
	"net/http"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

func (ctrl *DockerOperationController) ContainerTransfer(c *gin.Context) {
	var req service.DockerContainerTransferRequest
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "容器复制或迁移参数无效")
		return
	}
	userID, kind, id, connection, ok := ctrl.authorize(c, "docker_operate", req.ConnectionID)
	if !ok {
		return
	}
	req.ConnectionID = connection
	if req.Migrate {
		allowed, err := ctrl.permissions.CanUserAccess(userID, kind, id, "docker_remove")
		if err != nil || !allowed {
			result.Failed(c, http.StatusForbidden, "迁移需要源资产的 Docker 删除权限")
			return
		}
	}
	if req.TargetAssetType != "host" && req.TargetAssetType != "physical" {
		result.Failed(c, http.StatusBadRequest, "Docker 目标仅支持主机或物理机资产")
		return
	}
	if service.AssetExists(ctrl.db, req.TargetAssetType, req.TargetAssetID) != nil {
		result.Failed(c, http.StatusNotFound, "目标资产不存在")
		return
	}
	allowed, err := ctrl.permissions.CanUserAccess(userID, req.TargetAssetType, req.TargetAssetID, "docker_operate")
	if err != nil || !allowed {
		result.Failed(c, http.StatusForbidden, "没有目标资产的 Docker 操作权限")
		return
	}
	var target model.CmdbAssetConnection
	if ctrl.db.Where("id = ? AND asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", req.TargetConnectionID, req.TargetAssetType, req.TargetAssetID, "ssh", true).First(&target).Error != nil {
		result.Failed(c, http.StatusBadRequest, "目标 SSH 连接档案不存在或未启用")
		return
	}
	started := time.Now()
	output, opErr := ctrl.service.ContainerTransfer(c.Request.Context(), kind, id, c.Param("container"), req)
	status, message := "success", ""
	if opErr != nil {
		status, message = "failed", opErr.Error()
	}
	action := "container_copy"
	if req.Migrate {
		action = "container_migrate"
	}
	ctrl.recordAudit(userID, kind, id, connection, action, fmt.Sprintf("%s -> %s:%d/%s", c.Param("container"), req.TargetAssetType, req.TargetAssetID, req.Name), status, message, started, time.Now().UTC())
	if opErr != nil {
		result.Failed(c, http.StatusBadRequest, opErr.Error())
		return
	}
	result.Success(c, output)
}
