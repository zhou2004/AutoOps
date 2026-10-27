package controller

import (
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var dockerUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// recordAudit 写入 Docker 操作审计记录（用于二进制/流式接口）。
func (ctrl *DockerOperationController) recordAudit(userID uint, assetType string, assetID, connectionID uint, action, target, status, message string, started, finished time.Time) {
	if len(target) > 255 {
		target = target[:255]
	}
	if len(message) > 500 {
		message = message[:500]
	}
	audit := model.CmdbDockerAudit{AssetType: assetType, AssetID: assetID, ConnectionID: connectionID, UserID: userID, Action: action, Target: target, Status: status, Message: message, DurationMS: time.Since(started).Milliseconds(), CreatedAt: started.UTC(), FinishedAt: &finished}
	_ = ctrl.db.Create(&audit)
}

// Terminal 通过 WebSocket 连接容器终端。
func (ctrl *DockerOperationController) Terminal(c *gin.Context) {
	container := c.Param("container")
	connectionID := dockerConnectionID(c)
	userID, assetType, assetID, connID, ok := ctrl.authorize(c, "docker_operate", connectionID)
	if !ok {
		return
	}
	shell := c.DefaultQuery("shell", "/bin/sh")
	conn, err := dockerUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Docker 终端 WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	started := time.Now()
	err = ctrl.service.ContainerTerminal(c.Request.Context(), assetType, assetID, connID, container, shell, conn)
	finished := time.Now().UTC()
	status, message := "success", ""
	if err != nil {
		status, message = "failed", err.Error()
	}
	ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", container+"/terminal", status, message, started, finished)
}

// ContainerInspect 查看容器详情。
func (ctrl *DockerOperationController) ContainerInspect(c *gin.Context) {
	container := c.Param("container")
	ctrl.audited(c, "docker_view", container, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		return ctrl.service.ContainerInspect(c.Request.Context(), kind, id, connection, container)
	})
}

// ContainerFileList 容器内文件列表。
func (ctrl *DockerOperationController) ContainerFileList(c *gin.Context) {
	container := c.Param("container")
	path := c.DefaultQuery("path", "/")
	ctrl.audited(c, "docker_view", container+"/files", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.ContainerFileList(c.Request.Context(), kind, id, connection, container, path)
		return gin.H{"list": items}, err
	})
}

// ContainerFileContent 容器内文件内容。
func (ctrl *DockerOperationController) ContainerFileContent(c *gin.Context) {
	container := c.Param("container")
	path := c.Query("path")
	ctrl.audited(c, "docker_view", container+"/file", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		return ctrl.service.ContainerFileContent(c.Request.Context(), kind, id, connection, container, path)
	})
}

// ContainerFileUpdate 覆盖写入容器内文件。
func (ctrl *DockerOperationController) ContainerFileUpdate(c *gin.Context) {
	container := c.Param("container")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Path         string `json:"path"`
		Content      string `json:"content"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "文件更新参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", container+"/file", req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		err := ctrl.service.ContainerFileWrite(c.Request.Context(), kind, id, connection, container, req.Path, []byte(req.Content))
		return gin.H{}, err
	})
}

// ContainerFileMkdir 容器内创建目录。
func (ctrl *DockerOperationController) ContainerFileMkdir(c *gin.Context) {
	container := c.Param("container")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Path         string `json:"path"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "目录参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", container+"/mkdir", req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		err := ctrl.service.ContainerFileMkdir(c.Request.Context(), kind, id, connection, container, req.Path)
		return gin.H{}, err
	})
}

// ContainerFileDelete 删除容器内文件。
func (ctrl *DockerOperationController) ContainerFileDelete(c *gin.Context) {
	container := c.Param("container")
	path := c.Query("path")
	ctrl.audited(c, "docker_operate", container+"/delete", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		err := ctrl.service.ContainerFileDelete(c.Request.Context(), kind, id, connection, container, path)
		return gin.H{}, err
	})
}

// ContainerFileDownload 下载容器内文件。
func (ctrl *DockerOperationController) ContainerFileDownload(c *gin.Context) {
	container := c.Param("container")
	path := c.Query("path")
	userID, assetType, assetID, connID, ok := ctrl.authorize(c, "docker_view", dockerConnectionID(c))
	if !ok {
		return
	}
	started := time.Now()
	data, err := ctrl.service.ContainerFileDownload(c.Request.Context(), assetType, assetID, connID, container, path)
	finished := time.Now().UTC()
	if err != nil {
		ctrl.recordAudit(userID, assetType, assetID, connID, "docker_view", container+"/download", "failed", err.Error(), started, finished)
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.recordAudit(userID, assetType, assetID, connID, "docker_view", container+"/download", "success", "", started, finished)
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(path))
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// ContainerFileUpload 上传文件到容器。
func (ctrl *DockerOperationController) ContainerFileUpload(c *gin.Context) {
	container := c.Param("container")
	path := c.PostForm("path")
	userID, assetType, assetID, connID, ok := ctrl.authorize(c, "docker_operate", dockerConnectionID(c))
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "未找到上传文件")
		return
	}
	f, err := file.Open()
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无法读取上传文件")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, 512<<20))
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "读取上传文件失败")
		return
	}
	started := time.Now()
	err = ctrl.service.ContainerFileWrite(c.Request.Context(), assetType, assetID, connID, container, path, content)
	finished := time.Now().UTC()
	if err != nil {
		ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", container+"/upload", "failed", err.Error(), started, finished)
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	ctrl.recordAudit(userID, assetType, assetID, connID, "docker_operate", container+"/upload", "success", "", started, finished)
	result.Success(c, "文件上传成功")
}
