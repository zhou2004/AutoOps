package authorization

import "strings"

type Module struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

var Catalog = []Module{
	{"dashboard", "数据看板", []string{"read"}},
	{"cmdb", "资产管理", []string{"read", "write", "execute"}},
	{"docker", "Docker 管理", []string{"read", "write", "execute"}},
	{"database", "数据库管理", []string{"read", "write", "execute"}},
	{"k8s", "容器管理", []string{"read", "write", "execute"}},
	{"monitor", "监控告警", []string{"read", "write", "execute"}},
	{"task", "任务中心", []string{"read", "write", "execute"}},
	{"app", "应用发布", []string{"read", "write", "execute"}},
	{"config", "配置中心", []string{"read", "write", "execute"}},
	{"tool", "工具与服务部署", []string{"read", "write", "execute"}},
	{"aiops", "AI 运维", []string{"read", "chat", "operate"}},
	{"system", "基础管理", []string{"read", "write"}},
	{"audit", "日志管理", []string{"read", "write"}},
}

func Known(permission string) bool {
	for _, m := range Catalog {
		for _, a := range m.Actions {
			if permission == m.Key+":"+a {
				return true
			}
		}
	}
	return false
}

type Requirement struct {
	Permission string
	AdminOnly  bool
	Self       bool
}

// Route uses Gin's registered route, never the user-controlled raw URL.
// Unknown modules are denied. Every protected HTTP, SSE and WebSocket route uses this policy.
func Route(method, fullPath string) Requirement {
	p := strings.TrimPrefix(fullPath, "/api/v1")
	if p == "/system/permissions/me" || p == "/system/api-keys" && method == "GET" ||
		p == "/admin/info" || p == "/admin/updatePersonal" || p == "/admin/updatePersonalPassword" || p == "/upload" ||
		p == "/cmdb/permission/my" || strings.HasPrefix(p, "/cmdb/permission/check/") ||
		p == "/k8s/permission/my" || p == "/k8s/rbac/my-permissions" {
		return Requirement{Self: true}
	}
	admin := false
	module := ""
	switch {
	case strings.HasPrefix(p, "/system/"), strings.HasPrefix(p, "/admin/"), strings.HasPrefix(p, "/role/"), strings.HasPrefix(p, "/menu/"), strings.HasPrefix(p, "/dept/"), strings.HasPrefix(p, "/post/"):
		module = "system"
		admin = true
	case strings.HasPrefix(p, "/sysLoginInfo/"), strings.HasPrefix(p, "/sysOperationLog/"), strings.HasPrefix(p, "/cmdb/sqlLog/"):
		module = "audit"
	case strings.HasPrefix(p, "/cmdb/"):
		module = "cmdb"
		if strings.Contains(p, "/docker") {
			module = "docker"
		}
		if strings.HasPrefix(p, "/cmdb/database") || strings.HasPrefix(p, "/cmdb/sql") {
			module = "database"
		}
		admin = strings.HasPrefix(p, "/cmdb/permission") || strings.HasPrefix(p, "/cmdb/sql")
	case strings.HasPrefix(p, "/k8s/"):
		module = "k8s"
		admin = strings.HasPrefix(p, "/k8s/permission") || strings.HasPrefix(p, "/k8s/user-group") || strings.HasPrefix(p, "/k8s/group-permission") || strings.HasPrefix(p, "/k8s/rbac/")
		if strings.Contains(p, "/storage/migrate") || strings.Contains(p, "/storage/switch-pvc") {
			admin = true
		}
		if (p == "/k8s/cluster" || p == "/k8s/cluster/:id" || strings.HasSuffix(p, "/sync")) && method != "GET" {
			admin = true
		}
	case strings.HasPrefix(p, "/monitor/"):
		module = "monitor"
		admin = p == "/monitor/agent/list" || p == "/monitor/agent/statistics"
	case strings.HasPrefix(p, "/task"), strings.HasPrefix(p, "/template/"), strings.HasPrefix(p, "/ws/task/"):
		module = "task"
		admin = strings.HasPrefix(p, "/task/kubeasz") || strings.HasPrefix(p, "/task/monitor/")
	case p == "/apps" || strings.HasPrefix(p, "/apps/"), strings.HasPrefix(p, "/jenkins/"):
		module = "app"
	case strings.HasPrefix(p, "/config/"):
		module = "config"
		// These endpoints expose or configure shared infrastructure credentials.
		admin = strings.HasPrefix(p, "/config/keymanage") || strings.HasPrefix(p, "/config/sync-schedule") || strings.HasPrefix(p, "/config/ansible") || method != "GET"
	case p == "/tool" || strings.HasPrefix(p, "/tool/"):
		module = "tool"
	case strings.HasPrefix(p, "/aiops/"):
		module = "aiops"
		admin = strings.HasPrefix(p, "/aiops/configs") || strings.HasPrefix(p, "/aiops/models/")
	case strings.HasPrefix(p, "/dashboard/"):
		module = "dashboard"
	}
	if module == "" {
		return Requirement{}
	}
	action := "write"
	if method == "GET" || method == "HEAD" {
		action = "read"
	}
	if module == "aiops" {
		action = "read"
		if method != "GET" {
			action = "chat"
		}
		if strings.HasSuffix(p, "/grants") || admin {
			action = "operate"
		}
		if strings.HasSuffix(p, "/tools") {
			action = "read"
		} // Per-tool checks apply in the service, including callbacks.
	} else {
		// GET terminal/command/check endpoints have side effects too.
		if strings.Contains(p, "/hostssh/") || strings.Contains(p, "/terminal") || strings.Contains(p, "/files") && module == "k8s" ||
			strings.Contains(p, "/check/") || strings.HasSuffix(p, "/hot-reload") {
			action = "execute"
		}
		if method != "GET" && method != "HEAD" {
			for _, segment := range []string{"start", "stop", "restart", "run", "execute", "query", "deploy", "quick", "schedule", "pause", "resume", "rollback", "drain", "cordon", "sync", "trigger", "check-all", "scan-node-exporter", "uninstall", "connect", "ticket", "transfer", "copy", "migrate", "restore", "action", "backup", "test"} {
				if strings.Contains(p+"/", "/"+segment+"/") {
					action = "execute"
					break
				}
			}
		}
		if p == "/cmdb/databases/:id/query" || strings.HasSuffix(p, "/yaml/validate") || strings.HasSuffix(p, "/preview") {
			action = "read"
		}
	}
	if strings.Contains(p, "/databases/:id/mysql/") && method != "GET" && method != "HEAD" {
		action = "execute"
	}
	// Saving a runnable definition can enable scheduled execution.
	if method != "GET" && (p == "/task/add" || p == "/task/update" || p == "/task/ansible" || p == "/task/ansible/:id" && method == "PUT" || p == "/task/k8s") {
		action = "execute"
	}
	return Requirement{Permission: module + ":" + action, AdminOnly: admin}
}
