package controller

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

// RegistryRepositories 镜像仓库列表（分页）。
func (ctrl *DockerOperationController) RegistryRepositories(c *gin.Context) {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	items, total, err := ctrl.service.RegistryRepositories(q, page, pageSize)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, gin.H{"list": items, "total": total, "page": page, "pageSize": pageSize})
}

// ImagePull 拉取镜像。
func (ctrl *DockerOperationController) ImagePull(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Image        string `json:"image" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "镜像参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", req.Image, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ImagePull(c.Request.Context(), kind, id, connection, req.Image)
		return gin.H{"output": out}, err
	})
}

// ImageRemove 批量删除镜像。
func (ctrl *DockerOperationController) ImageRemove(c *gin.Context) {
	var req struct {
		ConnectionID uint     `json:"connectionId"`
		Images       []string `json:"images"`
		Force        bool     `json:"force"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "镜像删除参数无效")
		return
	}
	ctrl.audited(c, "docker_remove", strings.Join(req.Images, ","), req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ImageRemove(c.Request.Context(), kind, id, connection, req.Images, req.Force)
		return gin.H{"output": out}, err
	})
}

// ImageTag 为镜像打标签。
func (ctrl *DockerOperationController) ImageTag(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Source       string `json:"source" binding:"required"`
		Target       string `json:"target" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "镜像标签参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", req.Source+" -> "+req.Target, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ImageTag(c.Request.Context(), kind, id, connection, req.Source, req.Target)
		return gin.H{"output": out}, err
	})
}

// ImageRename 重命名镜像。
func (ctrl *DockerOperationController) ImageRename(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Source       string `json:"source" binding:"required"`
		Target       string `json:"target" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "镜像重命名参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", req.Source+" -> "+req.Target, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		out, err := ctrl.service.ImageRename(c.Request.Context(), kind, id, connection, req.Source, req.Target)
		return gin.H{"output": out}, err
	})
}

// ImageTransfer 跨机迁移/复制镜像到另一台平台主机/物理机的 Docker。
func (ctrl *DockerOperationController) ImageTransfer(c *gin.Context) {
	var req struct {
		ConnectionID    uint   `json:"connectionId"`
		Image           string `json:"image" binding:"required"`
		TargetAssetType string `json:"targetAssetType" binding:"required"`
		TargetAssetID   uint   `json:"targetAssetId" binding:"required"`
		Migrate         bool   `json:"migrate"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "镜像迁移参数无效")
		return
	}
	if req.TargetAssetType != "host" && req.TargetAssetType != "physical" {
		result.Failed(c, http.StatusBadRequest, "Docker 目标仅支持主机或物理机资产")
		return
	}
	userID, assetType, assetID, connectionID, ok := ctrl.authorize(c, "docker_operate", req.ConnectionID)
	if !ok {
		return
	}
	if err := service.AssetExists(ctrl.db, req.TargetAssetType, req.TargetAssetID); err != nil {
		result.Failed(c, http.StatusNotFound, "目标资产不存在")
		return
	}
	targetAllowed, err := ctrl.permissions.CanUserAccess(userID, req.TargetAssetType, req.TargetAssetID, "docker_operate")
	if err != nil || !targetAllowed {
		result.Failed(c, http.StatusForbidden, "没有目标资产的 Docker 操作权限")
		return
	}
	started := time.Now()
	out, opErr := ctrl.service.ImageTransfer(c.Request.Context(), assetType, assetID, connectionID, req.Image, req.TargetAssetType, req.TargetAssetID, req.Migrate)
	finished := time.Now().UTC()
	status, message := "success", ""
	if opErr != nil {
		status, message = "failed", opErr.Error()
	}
	target := req.Image + " -> " + req.TargetAssetType + ":" + strconv.FormatUint(uint64(req.TargetAssetID), 10)
	ctrl.recordAudit(userID, assetType, assetID, connectionID, "docker_operate", target, status, message, started, finished)
	if opErr != nil {
		result.Failed(c, http.StatusBadRequest, opErr.Error())
		return
	}
	result.Success(c, gin.H{"output": out})
}

// ImageExport 导出镜像为 tar 包下载。
func (ctrl *DockerOperationController) ImageExport(c *gin.Context) {
	images := strings.Split(c.Query("images"), ",")
	for i := range images {
		images[i] = strings.TrimSpace(images[i])
	}
	userID, assetType, assetID, connID, ok := ctrl.authorize(c, "docker_operate", dockerConnectionID(c))
	if !ok {
		return
	}
	started := time.Now()
	data, filename, err := ctrl.service.ImageExport(c.Request.Context(), assetType, assetID, connID, images)
	finished := time.Now().UTC()
	if err != nil {
		ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", "images/export", "failed", err.Error(), started, finished)
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", "images/export", "success", "", started, finished)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// ImageImport 从 tar 包导入镜像。
func (ctrl *DockerOperationController) ImageImport(c *gin.Context) {
	userID, assetType, assetID, connID, ok := ctrl.authorize(c, "docker_operate", dockerConnectionID(c))
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "未找到导入文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无法读取导入文件")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, 2<<30))
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "读取导入文件失败")
		return
	}
	started := time.Now()
	out, err := ctrl.service.ImageImport(c.Request.Context(), assetType, assetID, connID, content)
	finished := time.Now().UTC()
	if err != nil {
		ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", "images/import", "failed", err.Error(), started, finished)
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", "images/import", "success", "", started, finished)
	result.Success(c, gin.H{"output": out})
}
