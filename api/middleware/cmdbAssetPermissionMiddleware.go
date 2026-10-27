package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type CmdbAssetPermissionMiddleware struct{ permDao *dao.CmdbAssetPermissionDao }

func NewCmdbAssetPermissionMiddleware() *CmdbAssetPermissionMiddleware {
	return &CmdbAssetPermissionMiddleware{permDao: dao.NewCmdbAssetPermissionDao(common.GetDB())}
}

func permissionFailure(c *gin.Context, status int, message string) {
	result.Failed(c, status, message)
	c.Abort()
}

// requestAssetID reads the same source as the controller. A query ID must never override a JSON mutation ID.
func requestAssetID(c *gin.Context, param string) (uint, bool, error) {
	value := c.Param(param)
	if value == "" && (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || param != "id" || (c.Request.Method == http.MethodDelete && strings.HasSuffix(c.FullPath(), "/cmdb/database"))) {
		value = c.Query(param)
	}
	if value != "" {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil || id == 0 {
			return 0, true, fmt.Errorf("无效的资产ID")
		}
		return uint(id), true, nil
	}
	if param != "id" || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Body == nil {
		return 0, false, nil
	}
	if !strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		return 0, false, nil
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20+1))
	if err != nil {
		return 0, false, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > 2<<20 {
		return 0, false, fmt.Errorf("请求体过大")
	}
	var bodyID struct {
		ID *uint `json:"id"`
	}
	if err = json.Unmarshal(body, &bodyID); err != nil {
		return 0, false, fmt.Errorf("无效的资产ID")
	}
	if bodyID.ID == nil {
		return 0, false, nil
	}
	if *bodyID.ID == 0 {
		return 0, true, fmt.Errorf("无效的资产ID")
	}
	return *bodyID.ID, true, nil
}

func (m *CmdbAssetPermissionMiddleware) RequireCmdbPermission(assetType string) gin.HandlerFunc {
	return m.require(assetType, "", "id", false)
}

// RequireAssetAction enforces an explicit action and the ID in the path/query/body.
func (m *CmdbAssetPermissionMiddleware) RequireAssetAction(assetType, action string) gin.HandlerFunc {
	return m.require(assetType, action, "id", action != "create" && action != "list")
}

func (m *CmdbAssetPermissionMiddleware) RequireAssetActionForID(assetType, action, idParam string) gin.HandlerFunc {
	return m.require(assetType, action, idParam, true)
}

func (m *CmdbAssetPermissionMiddleware) require(assetType, action, idParam string, needsID bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			permissionFailure(c, http.StatusUnauthorized, "未授权")
			return
		}
		isAdmin, err := m.permDao.IsAdministrator(admin.ID)
		if err != nil {
			permissionFailure(c, http.StatusInternalServerError, "权限校验失败")
			return
		}
		if isAdmin {
			c.Set("cmdb_is_admin", true)
			c.Next()
			return
		}
		id, present, err := requestAssetID(c, idParam)
		if err != nil {
			permissionFailure(c, http.StatusBadRequest, err.Error())
			return
		}
		requireID := needsID
		required := action
		if required == "" {
			required = methodToAction(c.Request.Method)
			if required == "list" && (c.Param(idParam) != "" || strings.HasSuffix(c.FullPath(), "/hostinfo") || strings.HasSuffix(c.FullPath(), "/database/info") || strings.HasSuffix(c.FullPath(), "/groupbyname")) {
				required = "get"
			}
			if strings.HasSuffix(c.FullPath(), "/hostsync") {
				required = "update"
				requireID = true
			}
		}
		if !dao.ValidPermissionAction(required) {
			permissionFailure(c, http.StatusForbidden, "不支持的资产操作")
			return
		}
		if (requireID || required == "update" || required == "delete") && !present {
			permissionFailure(c, http.StatusBadRequest, "缺少资产ID")
			return
		}
		perms, err := m.permDao.EffectivePermissions(admin.ID)
		if err != nil {
			permissionFailure(c, http.StatusInternalServerError, "权限校验失败")
			return
		}
		hasAction := false
		for _, p := range perms {
			if dao.RuleAllowsAction(p, assetType, required) {
				hasAction = true
				break
			}
		}
		if !hasAction {
			permissionFailure(c, http.StatusForbidden, "您没有该资产操作的权限")
			return
		}
		ids, err := m.permDao.AllowedAssetIDs(perms, assetType, required)
		if err != nil {
			permissionFailure(c, http.StatusInternalServerError, "权限范围查询失败")
			return
		}
		if present && !ids[id] {
			permissionFailure(c, http.StatusForbidden, "您没有该资产操作的权限")
			return
		}
		// Only inventory CRUD has placement fields; connection settings also use update.
		if action == "" && (required == "create" || required == "update") {
			placementID := id
			if required == "create" {
				placementID = 0
			}
			if err := m.checkPlacement(c, perms, assetType, placementID, required); err != nil {
				permissionFailure(c, http.StatusForbidden, err.Error())
				return
			}
		}
		c.Set("cmdb_allowed_asset_ids", ids)
		hostIDs := ids
		if assetType == "group" {
			hostIDs, err = m.permDao.AllowedAssetIDs(perms, "host", "list")
			if err != nil {
				permissionFailure(c, http.StatusInternalServerError, "主机范围查询失败")
				return
			}
		}
		c.Set("cmdb_allowed_host_ids", hostIDs)
		c.Set("cmdb_asset_type", assetType)
		c.Set("cmdb_permission_action", required)
		c.Next()
	}
}

func (m *CmdbAssetPermissionMiddleware) CheckHostPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := "connect_ssh"
		path := c.FullPath()
		switch {
		case strings.Contains(path, "/command/"):
			action = "command"
		case strings.Contains(path, "/upload/"):
			action = "file_upload"
		case strings.Contains(path, "/download/"):
			action = "file_download"
		case strings.Contains(path, "/delete/"):
			action = "file_delete"
		case strings.Contains(path, "/files/"):
			action = "file_list"
		}
		m.RequireAssetAction("host", action)(c)
	}
}

func (m *CmdbAssetPermissionMiddleware) FilterAllowedHosts() gin.HandlerFunc {
	return m.RequireAssetAction("host", "list")
}

func (m *CmdbAssetPermissionMiddleware) AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := jwt.GetAdmin(c)
		if err != nil {
			permissionFailure(c, http.StatusUnauthorized, "未授权")
			return
		}
		allowed, err := m.permDao.IsAdministrator(admin.ID)
		if err != nil {
			permissionFailure(c, http.StatusInternalServerError, "权限校验失败")
			return
		}
		if !allowed {
			permissionFailure(c, http.StatusForbidden, "仅管理员可执行此操作")
			return
		}
		c.Next()
	}
}

func methodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return "list"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	}
	return ""
}

func ParseIDList(value string) []uint {
	var ids []uint
	for _, part := range strings.Split(value, ",") {
		if id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func (m *CmdbAssetPermissionMiddleware) checkPlacement(c *gin.Context, perms []model.CmdbAssetPermission, assetType string, assetID uint, action string) error {
	if strings.HasSuffix(c.FullPath(), "/hostsync") {
		return nil
	}
	if assetType != "host" && assetType != "physical" && assetType != "network" && assetType != "cabinet" && assetType != "group" {
		return nil
	}
	var placement struct {
		GroupID  uint `json:"groupId"`
		IDCID    uint `json:"idcId"`
		ParentID uint `json:"parentId"`
	}
	if strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
		value, err := strconv.ParseUint(c.PostForm("groupId"), 10, 64)
		if err != nil {
			return fmt.Errorf("无效的资产分组")
		}
		placement.GroupID = uint(value)
	} else {
		if c.Request.Body == nil {
			return fmt.Errorf("缺少资产归属信息")
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20+1))
		if err != nil {
			return fmt.Errorf("无法读取资产归属信息")
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if len(body) > 2<<20 || json.Unmarshal(body, &placement) != nil {
			return fmt.Errorf("无效的资产归属信息")
		}
	}
	destination := placement.IDCID
	if assetType == "host" {
		destination = placement.GroupID
	} else if assetType == "group" {
		destination = placement.ParentID
	}
	allowed, err := m.permDao.CanPlaceAsset(perms, assetType, assetID, destination, action)
	if err != nil {
		return fmt.Errorf("资产归属权限校验失败")
	}
	if !allowed {
		return fmt.Errorf("您没有在目标分组或机房创建、迁移该资产的权限")
	}
	return nil
}
