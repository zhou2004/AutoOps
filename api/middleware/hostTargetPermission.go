package middleware

import (
	"bytes"
	cmdbdao "dodevops-api/api/cmdb/dao"
	"dodevops-api/common"
	"dodevops-api/pkg/authorization"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Legacy monitoring/deployment routes must obey the same asset and credential grants as CMDB.
func checkHostTargets(c *gin.Context, identity authorization.Identity) bool {
	if identity.Admin {
		return true
	}
	p := strings.TrimPrefix(c.FullPath(), "/api/v1")
	ids := []uint{}
	action := "get"
	matched := false
	parse := func(s string) bool {
		id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
		if err != nil || id == 0 {
			return false
		}
		ids = append(ids, uint(id))
		return true
	}
	if strings.HasPrefix(p, "/monitor/host/") || strings.HasPrefix(p, "/monitor/hosts/") || strings.HasPrefix(p, "/monitor/agent/status/") {
		matched = true
		if !parse(c.Param("id")) {
			denyAuth(c, 400, "主机 ID 无效")
			return false
		}
	}
	if p == "/monitor/hosts" {
		matched = true
		for _, s := range strings.Split(c.Query("ids"), ",") {
			if !parse(s) {
				denyAuth(c, 400, "主机 ID 无效")
				return false
			}
		}
	}
	if strings.HasPrefix(p, "/monitor/agent/restart/") || strings.HasPrefix(p, "/monitor/agent/delete/") {
		matched = true
		action = "connect_ssh"
		if !parse(c.Param("id")) {
			denyAuth(c, 400, "主机 ID 无效")
			return false
		}
	}
	if p == "/monitor/agent/deploy" || p == "/monitor/agent/uninstall" || p == "/monitor/agent/scan-node-exporter" || p == "/tool/deploy" && c.Request.Method == "POST" {
		matched = true
		action = "connect_ssh"
		data, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
		if err != nil {
			denyAuth(c, 400, "请求过大")
			return false
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(data))
		var body struct {
			HostIDs []uint `json:"hostIds"`
			HostID  uint   `json:"hostId"`
		}
		if json.Unmarshal(data, &body) != nil {
			denyAuth(c, 400, "主机参数无效")
			return false
		}
		ids = body.HostIDs
		if body.HostID > 0 {
			ids = append(ids, body.HostID)
		}
	}
	if p == "/tool/deploy/:id" || p == "/tool/deploy/:id/status" {
		matched = true
		if c.Request.Method != "GET" {
			action = "connect_ssh"
		}
		var row struct{ HostID uint }
		if common.GetDB().Table("tool_service_deploy").Select("host_id").Where("id = ?", c.Param("id")).Take(&row).Error != nil {
			denyAuth(c, 404, "部署记录不存在")
			return false
		}
		ids = append(ids, row.HostID)
	}

	// Resolve stored task targets before task execution, mutation or log streaming.
	taskID := ""
	table, column := "task_job", "host_ids"
	if strings.HasPrefix(p, "/task/:id/") {
		taskID = c.Param("id")
	}
	if strings.HasPrefix(p, "/taskjob/") {
		taskID = c.Query("taskId")
	}
	if strings.HasPrefix(p, "/task/ansible/:id") || strings.HasPrefix(p, "/ws/task/ansible/:id/") {
		taskID = c.Param("id")
		table, column = "task_ansible", "all_host_ids"
	}
	if taskID != "" {
		matched = true
		if c.Request.Method != "GET" {
			action = "connect_ssh"
		}
		var targets struct{ Hosts string }
		if common.GetDB().Table(table).Select(column+" AS hosts").Where("id = ?", taskID).Take(&targets).Error != nil {
			denyAuth(c, 404, "任务不存在")
			return false
		}
		if strings.HasPrefix(strings.TrimSpace(targets.Hosts), "[") {
			if json.Unmarshal([]byte(targets.Hosts), &ids) != nil {
				denyAuth(c, 403, "任务主机范围无效")
				return false
			}
		} else {
			for _, id := range strings.Split(targets.Hosts, ",") {
				if !parse(id) {
					denyAuth(c, 403, "任务主机范围无效")
					return false
				}
			}
		}
	}
	if !matched {
		return true
	}
	if len(ids) == 0 || len(ids) > 1000 {
		denyAuth(c, 400, "请选择有效主机")
		return false
	}
	dao := cmdbdao.NewCmdbAssetPermissionDao(common.GetDB().WithContext(c.Request.Context()))
	for _, id := range ids {
		allowed, err := dao.CanUserAccess(identity.UserID, "host", id, action)
		if err != nil || !allowed {
			denyAuth(c, 403, "没有目标主机的访问或连接权限")
			return false
		}
		if action == "connect_ssh" {
			var host struct{ SSHKeyID uint }
			if common.GetDB().Table("cmdb_host").Select("ssh_key_id").Where("id = ?", id).Take(&host).Error != nil {
				denyAuth(c, 403, "目标主机不可用")
				return false
			}
			if host.SSHKeyID == 0 {
				denyAuth(c, 403, "目标主机尚未配置 SSH 连接凭据")
				return false
			}
		}
	}
	return true
}
