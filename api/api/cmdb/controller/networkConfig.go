package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NetworkConfigController struct {
	svc         *service.NetworkConfigService
	permissions *dao.CmdbAssetPermissionDao
}

func NewNetworkConfigController(db *gorm.DB) *NetworkConfigController {
	return &NetworkConfigController{svc: service.NewNetworkConfigService(db), permissions: dao.NewCmdbAssetPermissionDao(db)}
}

// Register expects the authenticated CMDB router group. Every endpoint also
// enforces the asset and action together, including direct handler invocation.
func (ctrl *NetworkConfigController) Register(group *gin.RouterGroup) {
	group.GET("/network/:id/config/capabilities", ctrl.Capabilities)
	group.GET("/network/:id/config/snapshots", ctrl.Snapshots)
	group.GET("/network/:id/config/snapshots/:snapshotId", ctrl.Snapshot)
	group.GET("/network/:id/config/diff", ctrl.Diff)
	group.GET("/network/:id/config/audits", ctrl.Audits)
	group.POST("/network/:id/config/backup", ctrl.Backup)
	group.POST("/network/:id/config/restore", ctrl.Restore)
}

func (ctrl *NetworkConfigController) authorize(c *gin.Context, action string) (userID, networkID uint, ok bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, http.StatusBadRequest, "无效的设备 ID")
		return
	}
	networkID = uint(id)
	userID, err = jwt.GetAdminId(c)
	if err != nil || userID == 0 {
		result.Failed(c, http.StatusUnauthorized, "请先登录")
		return
	}
	allowed, err := ctrl.permissions.CanUserAccess(userID, "network", networkID, action)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "权限校验失败")
		return
	}
	if !allowed {
		result.Failed(c, http.StatusForbidden, "没有该设备的配置操作权限")
		return
	}
	ok = true
	return
}

func (ctrl *NetworkConfigController) Capabilities(c *gin.Context) {
	if _, _, ok := ctrl.authorize(c, "config_view"); !ok {
		return
	}
	list, ready := ctrl.svc.Capabilities()
	result.Success(c, gin.H{"list": list, "encryptionReady": ready})
}

func configQueryPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return page, size
}

func (ctrl *NetworkConfigController) Snapshots(c *gin.Context) {
	_, networkID, ok := ctrl.authorize(c, "config_view")
	if !ok {
		return
	}
	page, size := configQueryPage(c)
	list, total, err := ctrl.svc.Snapshots(networkID, page, size)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询配置版本失败")
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (ctrl *NetworkConfigController) Snapshot(c *gin.Context) {
	_, networkID, ok := ctrl.authorize(c, "config_view")
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("snapshotId"), 10, 32)
	if err != nil || id == 0 {
		result.Failed(c, http.StatusBadRequest, "无效的配置版本 ID")
		return
	}
	detail, err := ctrl.svc.Snapshot(networkID, uint(id))
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, detail)
}

func (ctrl *NetworkConfigController) Diff(c *gin.Context) {
	_, networkID, ok := ctrl.authorize(c, "config_view")
	if !ok {
		return
	}
	from, errA := strconv.ParseUint(c.Query("from"), 10, 32)
	to, errB := strconv.ParseUint(c.Query("to"), 10, 32)
	if errA != nil || errB != nil || from == 0 || to == 0 {
		result.Failed(c, http.StatusBadRequest, "请选择两个配置版本")
		return
	}
	diff, err := ctrl.svc.Diff(networkID, uint(from), uint(to))
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, diff)
}

func (ctrl *NetworkConfigController) Audits(c *gin.Context) {
	_, networkID, ok := ctrl.authorize(c, "config_view")
	if !ok {
		return
	}
	page, size := configQueryPage(c)
	list, total, err := ctrl.svc.Audits(networkID, page, size)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "查询配置操作记录失败")
		return
	}
	result.Success(c, gin.H{"list": list, "total": total})
}

func (ctrl *NetworkConfigController) Backup(c *gin.Context) {
	userID, networkID, ok := ctrl.authorize(c, "config_backup")
	if !ok {
		return
	}
	var req service.NetworkConfigBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "请选择设备平台，备注最多 500 字符")
		return
	}
	snapshot, err := ctrl.svc.Backup(c.Request.Context(), userID, networkID, req)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, snapshot)
}

func (ctrl *NetworkConfigController) Restore(c *gin.Context) {
	userID, networkID, ok := ctrl.authorize(c, "config_restore")
	if !ok {
		return
	}
	var req service.NetworkConfigRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, http.StatusBadRequest, "请选择配置版本并输入设备名称确认")
		return
	}
	output, err := ctrl.svc.Restore(c.Request.Context(), userID, networkID, req)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	result.Success(c, output)
}
