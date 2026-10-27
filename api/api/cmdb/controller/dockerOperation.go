package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DockerOperationController struct {
	db          *gorm.DB
	service     *service.DockerOperationService
	permissions *dao.CmdbAssetPermissionDao
}

func NewDockerOperationController(db *gorm.DB) *DockerOperationController {
	return &DockerOperationController{db: db, service: service.NewDockerOperationService(db), permissions: dao.NewCmdbAssetPermissionDao(db)}
}

func (ctrl *DockerOperationController) Register(group *gin.RouterGroup) {
	group.GET("/docker/targets", ctrl.Targets)
	group.GET("/docker/assets/:assetType/:assetId/connections", NewAssetConnectionController(ctrl.db).List)
	base := "/docker/:assetType/:assetId"
	group.GET(base+"/info", ctrl.Info)
	group.GET(base+"/containers", ctrl.Containers)
	group.GET(base+"/images", ctrl.Images)
	group.GET(base+"/volumes", ctrl.Volumes)
	group.GET(base+"/networks", ctrl.Networks)
	group.GET(base+"/containers/:container/logs", ctrl.Logs)
	group.POST(base+"/containers/:container/action", ctrl.Action)
	group.POST(base+"/containers/:container/transfer", ctrl.ContainerTransfer)
	group.DELETE(base+"/containers/:container", ctrl.Remove)
	group.POST(base+"/containers", ctrl.Create)
	group.GET(base+"/audits", ctrl.Audits)

	// 容器终端与文件管理
	group.GET(base+"/containers/:container/terminal", ctrl.Terminal)
	group.GET(base+"/containers/:container/inspect", ctrl.ContainerInspect)
	group.GET(base+"/containers/:container/files", ctrl.ContainerFileList)
	group.GET(base+"/containers/:container/files/content", ctrl.ContainerFileContent)
	group.PUT(base+"/containers/:container/files/content", ctrl.ContainerFileUpdate)
	group.POST(base+"/containers/:container/files/directory", ctrl.ContainerFileMkdir)
	group.POST(base+"/containers/:container/files/upload", ctrl.ContainerFileUpload)
	group.GET(base+"/containers/:container/files/download", ctrl.ContainerFileDownload)
	group.DELETE(base+"/containers/:container/files", ctrl.ContainerFileDelete)

	// 镜像管理
	group.GET("/docker/registry/repositories", ctrl.RegistryRepositories)
	group.POST(base+"/images/pull", ctrl.ImagePull)
	group.DELETE(base+"/images", ctrl.ImageRemove)
	group.POST(base+"/images/tag", ctrl.ImageTag)
	group.POST(base+"/images/rename", ctrl.ImageRename)
	group.POST(base+"/images/transfer", ctrl.ImageTransfer)
	group.GET(base+"/images/export", ctrl.ImageExport)
	group.POST(base+"/images/import", ctrl.ImageImport)

	// 数据卷管理
	group.POST(base+"/volumes", ctrl.VolumeCreate)
	group.GET(base+"/volumes/:volume", ctrl.VolumeInspect)
	group.DELETE(base+"/volumes/:volume", ctrl.VolumeRemove)
	group.GET(base+"/volumes/:volume/containers", ctrl.VolumeContainers)

	// 网络管理
	group.POST(base+"/networks", ctrl.NetworkCreate)
	group.GET(base+"/networks/:network", ctrl.NetworkInspect)
	group.DELETE(base+"/networks/:network", ctrl.NetworkRemove)
	group.POST(base+"/networks/:network/connect", ctrl.NetworkConnect)
	group.POST(base+"/networks/:network/disconnect", ctrl.NetworkDisconnect)

	// Docker Compose 管理
	group.GET(base+"/compose", ctrl.ComposeList)
	group.GET(base+"/compose/:project", ctrl.ComposeGet)
	group.POST(base+"/compose", ctrl.ComposeSave)
	group.DELETE(base+"/compose/:project", ctrl.ComposeRemove)
	group.POST(base+"/compose/:project/action", ctrl.ComposeAction)
}

func (ctrl *DockerOperationController) authorize(c *gin.Context, action string, requestedConnection uint) (userID uint, assetType string, assetID, connectionID uint, ok bool) {
	assetType = c.Param("assetType")
	if assetType != "host" && assetType != "physical" {
		result.Failed(c, http.StatusBadRequest, "Docker 仅支持主机或物理机资产")
		return
	}
	parsed, err := strconv.ParseUint(c.Param("assetId"), 10, 32)
	if err != nil || parsed == 0 || service.AssetExists(ctrl.db, assetType, uint(parsed)) != nil {
		result.Failed(c, http.StatusNotFound, "资产不存在")
		return
	}
	assetID = uint(parsed)
	userID, err = jwt.GetAdminId(c)
	if err != nil || userID == 0 {
		result.Failed(c, http.StatusUnauthorized, "请先登录")
		return
	}
	allowed, err := ctrl.permissions.CanUserAccess(userID, assetType, assetID, action)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "权限校验失败")
		return
	}
	if !allowed {
		result.Failed(c, http.StatusForbidden, "没有该资产的 Docker 操作权限")
		return
	}
	if action == "audit" {
		ok = true
		return
	}
	var profile model.CmdbAssetConnection
	query := ctrl.db.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", assetType, assetID, "ssh", true)
	if requestedConnection > 0 {
		query = query.Where("id = ?", requestedConnection)
	}
	if err := query.Order("id ASC").First(&profile).Error; err != nil {
		result.Failed(c, http.StatusBadRequest, "未找到已启用的 SSH 连接档案")
		return
	}
	connectionID, ok = profile.ID, true
	return
}

func dockerConnectionID(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Query("connectionId"), 10, 32)
	return uint(id)
}

func (ctrl *DockerOperationController) audited(c *gin.Context, action, target string, requestedConnection uint, fn func(string, uint, uint) (any, error)) {
	userID, assetType, assetID, connectionID, ok := ctrl.authorize(c, action, requestedConnection)
	if !ok {
		return
	}
	started := time.Now()
	data, operationErr := fn(assetType, assetID, connectionID)
	finished := time.Now().UTC()
	status, message := "success", ""
	if operationErr != nil {
		status, message = "failed", operationErr.Error()
	}
	if len(target) > 255 {
		target = target[:255]
	}
	audit := model.CmdbDockerAudit{AssetType: assetType, AssetID: assetID, ConnectionID: connectionID, UserID: userID, Action: action, Target: target, Status: status, Message: message, ClientIP: c.ClientIP(), DurationMS: time.Since(started).Milliseconds(), CreatedAt: started.UTC(), FinishedAt: &finished}
	if ctrl.db.Create(&audit).Error != nil {
		result.Failed(c, http.StatusInternalServerError, "Docker 操作审计写入失败")
		return
	}
	if operationErr != nil {
		result.Failed(c, http.StatusBadRequest, operationErr.Error())
		return
	}
	result.Success(c, data)
}

func (ctrl *DockerOperationController) Info(c *gin.Context) {
	ctrl.audited(c, "docker_view", "daemon", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		return ctrl.service.Info(c.Request.Context(), kind, id, connection)
	})
}

func (ctrl *DockerOperationController) Containers(c *gin.Context) {
	ctrl.audited(c, "docker_view", "containers", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.Containers(c.Request.Context(), kind, id, connection)
		return gin.H{"list": items}, err
	})
}

func (ctrl *DockerOperationController) Images(c *gin.Context) {
	ctrl.audited(c, "docker_view", "images", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.Images(c.Request.Context(), kind, id, connection)
		return gin.H{"list": items}, err
	})
}

func (ctrl *DockerOperationController) Volumes(c *gin.Context) {
	ctrl.audited(c, "docker_view", "volumes", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.Volumes(c.Request.Context(), kind, id, connection)
		return gin.H{"list": items}, err
	})
}

func (ctrl *DockerOperationController) Networks(c *gin.Context) {
	ctrl.audited(c, "docker_view", "networks", dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		items, err := ctrl.service.Networks(c.Request.Context(), kind, id, connection)
		return gin.H{"list": items}, err
	})
}

func (ctrl *DockerOperationController) Logs(c *gin.Context) {
	container := c.Param("container")
	tail, _ := strconv.Atoi(c.DefaultQuery("tail", "200"))
	ctrl.audited(c, "docker_logs", container, dockerConnectionID(c), func(kind string, id, connection uint) (any, error) {
		logs, err := ctrl.service.Logs(c.Request.Context(), kind, id, connection, container, tail)
		return gin.H{"logs": logs}, err
	})
}

func (ctrl *DockerOperationController) Action(c *gin.Context) {
	container := c.Param("container")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		Action       string `json:"action" binding:"required"`
		ConfirmName  string `json:"confirmName"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "容器操作参数无效")
		return
	}
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	ctrl.audited(c, "docker_operate", container+"/"+req.Action, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		output, err := ctrl.service.Action(c.Request.Context(), kind, id, connection, container, req.Action, req.ConfirmName)
		return gin.H{"output": strings.TrimSpace(output)}, err
	})
}

func (ctrl *DockerOperationController) Remove(c *gin.Context) {
	container := c.Param("container")
	var req struct {
		ConnectionID uint   `json:"connectionId"`
		ConfirmName  string `json:"confirmName" binding:"required"`
	}
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "容器删除参数无效")
		return
	}
	ctrl.audited(c, "docker_remove", container, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		output, err := ctrl.service.Remove(c.Request.Context(), kind, id, connection, container, req.ConfirmName)
		return gin.H{"output": strings.TrimSpace(output)}, err
	})
}

func (ctrl *DockerOperationController) Create(c *gin.Context) {
	var req service.DockerCreateRequest
	if c.ShouldBindJSON(&req) != nil {
		result.Failed(c, http.StatusBadRequest, "容器创建参数无效")
		return
	}
	ctrl.audited(c, "docker_operate", req.Name, req.ConnectionID, func(kind string, id, connection uint) (any, error) {
		req.ConnectionID = connection
		containerID, err := ctrl.service.Create(c.Request.Context(), kind, id, &req)
		return gin.H{"containerId": containerID}, err
	})
}

func (ctrl *DockerOperationController) Audits(c *gin.Context) {
	userID, assetType, assetID, _, ok := ctrl.authorize(c, "audit", dockerConnectionID(c))
	_ = userID
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	query := ctrl.db.Model(&model.CmdbDockerAudit{}).Where("asset_type = ? AND asset_id = ?", assetType, assetID)
	var list []model.CmdbDockerAudit
	var total int64
	if query.Count(&total).Error != nil || query.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error != nil {
		result.Failed(c, 500, "读取 Docker 审计失败")
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}
