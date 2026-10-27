package service

import (
	"context"
	cmdbdao "dodevops-api/api/cmdb/dao"
	dashboard "dodevops-api/api/dashboard/service"
	"encoding/json"
	"errors"
	"time"
)

func (s *Service) Evidence(ctx context.Context, user uint, clusters []uint, modules []string, days int) (map[string]interface{}, error) {
	if !validModules(modules) {
		return nil, errors.New("Select valid diagnostic modules / 请选择有效诊断模块")
	}
	overview, err := dashboard.NewDashboardServiceWithDB(s.DB).GetOverview(ctx, user, clusters, days)
	if err != nil {
		return nil, err
	}
	admin, err := s.IsAdmin(ctx, user)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{"generatedAt": overview.GeneratedAt, "days": days, "assetScope": overview.AssetScope, "source": "AutoOps stored business records; not a real-time probe", "limitations": "Counts and recent execution summaries only. No live logs, configuration files or credentials. Missing sections are unavailable, not healthy."}
	available := map[string]interface{}{
		"cmdb":        map[string]interface{}{"hosts": overview.Hosts, "databases": overview.Databases, "physical": overview.Physical, "network": overview.Network},
		"k8s":         map[string]interface{}{"clusters": overview.Clusters},
		"monitor":     map[string]interface{}{"agents": overview.Agents, "endpoints": overview.Endpoints, "certificates": overview.Certificates, "alerts": overview.Alerts, "incidents": overview.Incidents},
		"task":        map[string]interface{}{"jobs": overview.Jobs, "ansible": overview.Ansible},
		"application": map[string]interface{}{"applications": overview.Applications, "deployments": overview.Deployments},
	}
	// Recent titles can contain arbitrary payloads. Only aggregate counts leave AutoOps.
	if overview.Incidents != nil {
		overview.Incidents.Recent = nil
	}
	if overview.Applications != nil {
		overview.Applications.Distribution = nil
	}
	for _, execution := range []string{"jobs", "ansible", "deployments"} {
		switch execution {
		case "jobs":
			if overview.Jobs != nil {
				overview.Jobs.Recent = nil
			}
		case "ansible":
			if overview.Ansible != nil {
				overview.Ansible.Recent = nil
			}
		case "deployments":
			if overview.Deployments != nil {
				overview.Deployments.Recent = nil
			}
		}
	}
	db := s.DB.WithContext(ctx)
	count := func(table string) interface{} {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil {
			return map[string]string{"error": "Data unavailable"}
		}
		return n
	}
	for _, module := range modules {
		if value, ok := available[module]; ok {
			out[module] = value
			continue
		}
		switch module {
		case "configcenter":
			if admin {
				out[module] = map[string]interface{}{"accounts": count("config_account"), "cloudCredentials": count("config_ecsauth"), "keys": count("config_keymanage"), "syncSchedules": count("config_sync_schedule")}
			} else {
				out[module] = map[string]string{"error": "Administrator permission required"}
			}
		case "system":
			if admin {
				out[module] = map[string]interface{}{"users": count("sys_admin"), "roles": count("sys_role"), "menus": count("sys_menu")}
			} else {
				out[module] = map[string]string{"error": "Administrator permission required"}
			}
		case "tools":
			q := db.Table("tool_service_deploy")
			if !admin {
				dao := cmdbdao.NewCmdbAssetPermissionDao(db)
				rules, err := dao.EffectivePermissions(user)
				if err != nil {
					return nil, err
				}
				scope, err := dao.AllowedAssetIDs(rules, "host", "list")
				if err != nil {
					return nil, err
				}
				hosts := cmdbdao.ApplyAssetIDScope(db.Table("cmdb_host"), scope).Select("id")
				q = q.Where("host_id IN (?)", hosts)
			}
			var statuses []struct {
				Status int   `json:"status"`
				Count  int64 `json:"count"`
			}
			if err := q.Select("status, COUNT(*) AS count").Group("status").Scan(&statuses).Error; err != nil {
				out[module] = map[string]string{"error": "Data unavailable"}
			} else {
				out[module] = map[string]interface{}{"services": statuses, "statusLegend": "0=deploying,1=running,2=stopped,3=failed"}
			}
		}
	}
	// Null section values retain module failures without leaking errors from unselected modules.
	raw, _ := json.Marshal(out)
	if len(raw) > 128<<10 {
		return nil, errors.New("Evidence size limit exceeded / 业务快照超出大小限制")
	}
	return out, nil
}
func persistContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
