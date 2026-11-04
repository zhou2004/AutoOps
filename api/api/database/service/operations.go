package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"dodevops-api/pkg/apikey"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Parameters struct {
	BackupName        string            `json:"backupName,omitempty"`
	VIP               VIPInput          `json:"vip"`
	RuntimeOnly       bool              `json:"runtimeOnly"`
	SemiSyncEnabled   bool              `json:"semiSyncEnabled"`
	SemiSyncTimeoutMS int               `json:"semiSyncTimeoutMs"`
	OperationID       uint              `json:"operationId,omitempty"`
	MemberID          uint              `json:"memberId,omitempty"`
	CandidateID       uint              `json:"candidateId,omitempty"`
	Members           []MemberInput     `json:"members,omitempty"`
	Changes           map[string]string `json:"changes,omitempty"`
	Management        string            `json:"management,omitempty"`
	BackupID          uint              `json:"backupId,omitempty"`
	ParentBackupID    uint              `json:"parentBackupId,omitempty"`
	RepositoryID      uint              `json:"repositoryId,omitempty"`
	Method            string            `json:"method,omitempty"`
	HostID            uint              `json:"hostId,omitempty"`
	Port              int               `json:"port,omitempty"`
	TargetTime        string            `json:"targetTime,omitempty"`
	Slots             []int             `json:"slots,omitempty"`
	BeforeFile        string            `json:"beforeFile,omitempty"`
}
type Capability struct {
	Action     string `json:"action"`
	Label      string `json:"label"`
	LabelEn    string `json:"labelEn"`
	Enabled    bool   `json:"enabled"`
	Reason     string `json:"reason"`
	Permission string `json:"permission"`
}

var operationLabels = map[string]string{"configure-vip": "配置虚拟 IP", "verify-vip": "验证 VIP 漂移", "update-patroni": "更新固定 Patroni 控制组件", "balance-primaries": "重新分布 Redis 主节点", "configure-semisync": "配置 MySQL 半同步", "refresh": "刷新状态", "collect-storage": "采集容量", "adopt": "调整管理级别", "provision": "安装集群", "switchover": "计划切换", "recover": "故障恢复", "restart": "滚动重启", "add-node": "新增成员", "rebuild-replica": "重建副本", "remove-node": "移出成员", "reshard": "迁移槽位", "assign-replica": "调整分片副本归属", "expire-backup": "清理备份", "backup": "立即备份", "verify-backup": "恢复验证", "restore": "隔离恢复", "configure": "应用参数", "redis-bgsave": "生成 RDB", "redis-aof-rewrite": "重写 AOF", "purge-binlogs": "清理归档 binlog", "configure-endpoint": "部署访问入口", "configure-fencing": "配置独立隔离", "reconcile": "核验中断任务", "configure-archive": "配置 WAL 归档", "resume-writes": "恢复主库写入", "rejoin": "旧主安全重加入", "repair-replication": "GTID 拓扑修复", "enter-maintenance": "进入维护（MySQL 一小时）", "exit-maintenance": "退出原生维护", "acknowledge-recovery": "确认原生恢复事件"}

func actionPermission(action string) string {
	switch action {
	case "refresh", "collect-storage":
		return "db_cluster_view"
	case "balance-primaries", "switchover", "recover", "resume-writes", "repair-replication", "enter-maintenance", "exit-maintenance", "acknowledge-recovery":
		return "db_cluster_switch"
	case "backup", "verify-backup", "expire-backup":
		return "db_cluster_backup"
	case "restore":
		return "db_cluster_restore"
	case "configure-semisync", "configure", "configure-archive", "redis-bgsave", "redis-aof-rewrite", "purge-binlogs":
		return "db_cluster_config"
	default:
		return "db_cluster_manage"
	}
}
func capability(c *model.Cluster, action string) (bool, string) {
	if _, ok := operationLabels[action]; !ok {
		return false, "不支持的运维操作"
	}
	if action == "refresh" || action == "collect-storage" || action == "adopt" {
		return true, ""
	}
	if action == "reconcile" {
		return true, ""
	}
	if c.Management == "observe" {
		return false, "观察模式不会修改服务，请先完成运维接管"
	}
	if action == "provision" {
		if c.Status == "planned" || c.Status == "install-failed" {
			return true, ""
		}
		return false, "仅待安装的集群可以执行安装"
	}
	if c.Status == "planned" {
		return false, "集群尚未安装"
	}
	if reason := compatibilityReason(c); reason != "" {
		return false, reason
	}
	if action == "recover" && c.Engine == "redis" {
		return false, "故障恢复由 Redis 原生多数派执行；平台不另行强制选主"
	}
	if action == "switchover" || action == "recover" {
		if c.Topology == "standalone" {
			return false, "单机没有切换候选"
		}
		if c.Topology == "mysql-gtid" {
			found := false
			for _, m := range c.Members {
				found = found || m.Kind == "orchestrator"
			}
			if !found {
				return false, "需要配置 Orchestrator 管理成员"
			}
		}
		if c.Topology == "postgresql-patroni" {
			found := false
			for _, m := range c.Members {
				found = found || m.Kind == "patroni"
				if m.Kind == "patroni" && m.Version != "4.1.5" {
					return false, "当前 Patroni 控制版本未通过分区恢复兼容验证，请先更新到固定版本 4.1.5"
				}
			}
			if !found {
				return false, "需要配置 Patroni 管理成员"
			}
		}
	}
	if (action == "enter-maintenance" || action == "exit-maintenance") && c.Topology != "mysql-gtid" && c.Topology != "postgresql-patroni" {
		return false, "仅支持 Orchestrator 与 Patroni 原生维护"
	}
	if action == "configure-archive" && c.Topology != "postgresql-patroni" {
		return false, "原生 WAL 归档配置仅用于受管 Patroni 集群"
	}
	if (action == "resume-writes" || action == "rejoin" || action == "repair-replication" || action == "acknowledge-recovery") && c.Topology != "mysql-gtid" {
		return false, "恢复写入仅适用于已验证复制根节点的 MySQL GTID 集群"
	}
	if action == "update-patroni" && c.Topology != "postgresql-patroni" {
		return false, "需要 Patroni 集群"
	}
	if action == "configure-semisync" && c.Topology != "mysql-gtid" {
		return false, "半同步仅适用于 MySQL GTID 主从"
	}
	if action == "configure-fencing" && c.Engine != "mysql" {
		return false, "此操作仅适用于 MySQL Orchestrator"
	}
	if (action == "reshard" || action == "assign-replica" || action == "balance-primaries") && c.Topology != "redis-cluster" {
		return false, "只有 Redis Cluster 支持槽位迁移"
	}
	if strings.HasPrefix(action, "redis-") && c.Engine != "redis" {
		return false, "此操作仅支持 Redis"
	}
	if action == "purge-binlogs" && c.Engine != "mysql" {
		return false, "此操作仅支持 MySQL"
	}
	if action == "verify-vip" && c.VIP == nil {
		return false, "请先部署 VIP"
	}
	if (action == "configure-endpoint" || action == "configure-vip" || action == "verify-vip") && c.Engine == "redis" {
		return false, "Redis 使用 Sentinel/Cluster 原生客户端入口"
	}
	return true, ""
}
func (s *Service) Capabilities(user uint, c *model.Cluster) []Capability {
	out := []Capability{}
	for _, action := range []string{"configure-vip", "verify-vip", "refresh", "collect-storage", "adopt", "provision", "switchover", "recover", "restart", "add-node", "rebuild-replica", "remove-node", "reshard", "assign-replica", "balance-primaries", "expire-backup", "backup", "verify-backup", "restore", "configure", "redis-bgsave", "redis-aof-rewrite", "purge-binlogs", "configure-endpoint", "configure-fencing", "configure-semisync", "update-patroni", "reconcile", "configure-archive", "resume-writes", "rejoin", "repair-replication", "enter-maintenance", "exit-maintenance", "acknowledge-recovery"} {
		enabled, reason := capability(c, action)
		permission := actionPermission(action)
		if s.Authorize(user, c.ID, permission) != nil {
			enabled = false
			reason = "没有此集群的操作授权"
		}
		out = append(out, Capability{Action: action, Label: operationLabels[action], LabelEn: operationLabelsEnglish[action], Enabled: enabled, Reason: reason, Permission: permission})
	}
	return out
}

type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

func (s *Service) checks(ctx context.Context, c *model.Cluster, action string, p Parameters, user uint) []Check {
	checks := []Check{}
	add := func(name string, ok bool, message string) { checks = append(checks, Check{name, ok, message}) }
	enabled, reason := capability(c, action)
	add("操作能力", enabled, reason)
	if action == "adopt" {
		add("管理级别", p.Management == "observe" || p.Management == "operate" || p.Management == "ha", "请选择观察、运维托管或高可用托管")
		if p.Management != "observe" {
			reason := compatibilityReason(c)
			add("版本兼容", reason == "", reason)
		}
		if p.Management == "ha" {
			add("高可用验收", c.HAValidatedAt != nil, "需要已记录的故障演练与入口验收证据")
		}
		return checks
	}
	if action == "reconcile" {
		var old model.Operation
		e := s.DB.Where("id = ? AND cluster_id = ? AND status = ?", p.OperationID, c.ID, "unknown").First(&old).Error
		add("待核验任务", e == nil, "请选择当前集群结果待核验的任务")
		if e == nil {
			add("原操作权限", s.Authorize(user, c.ID, actionPermission(old.Action)) == nil, "仍需具备原任务的操作权限")
		}
		return checks
	}
	if action == "refresh" || action == "collect-storage" {
		return checks
	}
	if action != "provision" {
		add("状态时效", c.CollectedAt != nil && time.Since(*c.CollectedAt) < 90*time.Second, "状态须在 90 秒内采集，请先刷新拓扑")
	}
	if p.MemberID != 0 {
		_, e := member(c, p.MemberID)
		add("目标成员", e == nil, "目标必须属于当前集群")
	}
	if p.CandidateID != 0 {
		_, e := member(c, p.CandidateID)
		add("候选成员", e == nil, "候选必须属于当前集群")
	}
	switch action {
	case "configure-vip", "verify-vip":
		input := p.VIP
		if action == "verify-vip" && c.VIP != nil {
			input = vipParameters(c.VIP)
		}
		err := s.vipChecks(ctx, c, input, user)
		message := "网卡、双代理归属、ARP 地址与 VRID 冲突检查通过"
		if err != nil {
			message = err.Error()
		}
		add("VIP 部署预检", err == nil, message)
	case "update-patroni":
		add("复制与服务", object(c.Health)["replication"] == "healthy" && object(c.Health)["database"] == "healthy", "依次更新副本、计划切换、更新旧主；固定控制版本 4.1.5，数据库版本不变")
		for _, m := range c.Members {
			if m.Kind == "data" {
				add("受管实例 "+m.Name, m.Managed, "需要受管服务归属")
			}
		}
	case "balance-primaries":
		candidates, e := primaryPlacement(c)
		add("故障域匹配", e == nil, "需要健康的主节点、副本与足够的不同主机；只调用原生 CLUSTER FAILOVER，不强制选主")
		for _, m := range candidates {
			add("目标 "+m.Name, true, fmt.Sprintf("分片主节点分配到 %s:%d", m.Address, m.Port))
		}
	case "configure-semisync":
		add("半同步超时", p.SemiSyncTimeoutMS >= 100 && p.SemiSyncTimeoutMS <= 60000, "超时范围 100–60000ms；超时后可能退化为异步")
		add("复制健康", object(c.Health)["replication"] == "healthy", "全部副本须复制正常")
		for _, m := range c.Members {
			if m.Kind == "data" {
				add("受管插件 "+m.Name, m.Managed && strings.HasPrefix(m.Version, "8.0."), "仅对已确认服务归属的 MySQL 8.0 管理插件")
			}
		}
	case "exit-maintenance", "acknowledge-recovery":
		h := object(c.Health)
		add("退出维护核验", h["database"] == "healthy" && h["replication"] == "healthy", "复制必须恢复健康；此操作会清除当前所有成员的原生 downtime 或 Patroni pause")
	case "repair-replication":
		m, e := member(c, p.CandidateID)
		add("指定既有复制根节点", e == nil && m.Kind == "data" && m.Role == "primary" && m.Managed, "只能明确选择既有根节点；不会丢弃任何分叉 GTID")
		for _, m := range c.Members {
			if m.Kind == "data" {
				add("保护性只读 "+m.Name, m.State == "healthy" && flag(object(m.Metrics)["read_only"]) && flag(object(m.Metrics)["super_read_only"]), "修复前所有节点须可达且只读")
			}
		}
	case "rejoin":
		_, e := mysqlRejoinDonor(c, p.MemberID)
		m, me := member(c, p.MemberID)
		add("旧主身份与新主", e == nil && me == nil && m.Managed && m.Kind == "data", "目标须为受管旧主，并存在唯一可写的新主")
	case "resume-writes":
		root, e := primary(c)
		add("复制根节点", e == nil && root.State == "healthy", "需要已确认且可达的复制根节点")
		add("复制健康", object(c.Health)["replication"] == "healthy", "全部副本须复制正常")
		add("隔离机制", flag(object(c.Policy)["fencingInstalled"]), "需先安装独立恢复前隔离钩子")
	case "configure-endpoint":
		nodes, create, e := s.endpointPlan(c, p)
		add("双代理身份", e == nil, "需要两个不同主机上的受管代理、空闲的写端口与读取端口")
		if e == nil && create {
			preview := *c
			preview.Members = nodes
			checks = append(checks, s.preflightHosts(ctx, &preview)...)
		}
	case "provision":
		_, toolError := automationRoot()
		add("自动化执行器", toolError == nil, "数据库自动化目录不存在")
		for _, m := range c.Members {
			if m.HostID == 0 {
				add("CMDB 主机", false, "安装成员须关联 CMDB 主机")
			}
			if !m.Managed {
				add("目录归属", false, "禁止覆盖接管实例的数据目录")
			}
		}
		for _, check := range s.preflightHosts(ctx, c) {
			checks = append(checks, check)
		}
	case "add-node":
		nodes, e := s.newMemberPlan(c, p)
		add("新增实例", e == nil, "需要一个新 CMDB 实例、独立端口和目录")
		if e == nil {
			preview := *c
			preview.Members = nodes
			checks = append(checks, s.preflightHosts(ctx, &preview)...)
		}
		parent, e := member(c, p.CandidateID)
		add("基线节点", e == nil && parent.Kind == "data" && parent.Role == "primary" && parent.State == "healthy", "请选择健康主节点")
		if c.Engine == "mysql" && e == nil {
			add("MySQL 版本兼容", parent.Version == "8.0.24", "当前固定安装包为 8.0.24，CLONE 要求版本一致；其他版本须使用匹配发行包")
		}
	case "switchover", "recover":
		if action == "recover" && c.Engine == "postgresql" {
			old, e := primary(c)
			verified := e == nil
			if verified {
				verified = s.confirmStopped(ctx, *old) == nil
			}
			add("旧主独立隔离", verified, "必须通过 CMDB 独立 SSH 确认旧主服务与进程已停止；端口不可达不等于已隔离")
		}
		if c.Topology == "redis-sentinel" {
			add("原生候选选择", p.CandidateID == 0, "Sentinel 根据复制偏移、优先级和多数派选择副本，无需指定候选")
			add("Sentinel 多数派", object(c.Health)["control"] == "healthy", "所有 Sentinel 须通过 quorum 与多数派检查")
			_, e := primary(c)
			add("已确认主库", e == nil, "需要已发现的主节点")
			break
		}
		candidate, e := member(c, p.CandidateID)
		add("指定候选", e == nil && candidate.Kind == "data" && candidate.Role == "replica" && candidate.State == "healthy", "必须选择健康副本")
		if e == nil {
			m := object(candidate.Metrics)
			if c.Engine == "mysql" {
				if action == "switchover" {
					add("复制追平", m["replicationIO"] == "Yes" && m["replicationSQL"] == "Yes" && m["lagSeconds"] != nil && number(m["lagSeconds"]) == 0, "候选复制线程须正常且无延迟")
				} else {
					add("候选 SQL 线程", m["replicationSQL"] == "Yes", "主库故障时 IO 线程可能断开，SQL 应用线程仍须正常；GTID 资格由 Orchestrator 复核")
				}
			}
			if c.Engine == "redis" {
				add("复制链路", m["master_link_status"] == "up", "候选副本必须已连接主库")
			}
		}
		if c.Engine == "mysql" {
			add("独立隔离配置", flag(object(c.Policy)["fencingInstalled"]), "故障恢复需要独立隔离执行器，端口不可达不等于已隔离")
		}
		if action == "switchover" {
			primary := 0
			for _, m := range c.Members {
				if m.Kind == "data" && m.Role == "primary" && m.State == "healthy" {
					primary++
				}
			}
			add("主库健康", primary > 0, "计划切换需要当前主节点健康")
		}
	case "restore", "verify-backup":
		var b model.Backup
		e := s.DB.Where("id = ? AND cluster_id = ?", p.BackupID, c.ID).First(&b).Error
		add("有效备份集", e == nil && b.Status == "complete", "必须选择完整且已校验文件的备份集")
		add("隔离恢复端口", p.HostID != 0 && p.Port >= 1024 && p.Port <= 65535, "恢复需要指定 CMDB 主机和独立端口")
		for _, m := range c.Members {
			if m.HostID == p.HostID && m.Port == p.Port {
				add("禁止覆盖现有节点", false, "恢复端口已由当前集群使用")
			}
		}
	case "expire-backup":
		_, e := s.expirableBackup(c.ID, p.BackupID)
		message := "仅清理指定备份的本地文件和仓库副本，不受保留天数限制"
		if e != nil {
			message = e.Error()
		}
		add("受管备份归属", e == nil, message)
	case "backup":
		_, nameError := backupName(c, p, time.Now())
		add("备份名称", nameError == nil, "备份名称最多 100 个字符，不能包含换行或控制字符；留空自动生成")
		if c.Topology != "redis-cluster" {
			root, e := primary(c)
			add("唯一复制根节点", e == nil && root.State == "healthy", "需要唯一且可连接的主节点，拒绝在冲突拓扑上创建备份")
		}
		add("备份方式", supportsBackupMethod(c.Engine, p.Method), "请选择支持的备份方式")
		if p.Method == "binlog-archive" {
			add("binlog 归档引擎", c.Engine == "mysql", "仅 MySQL 支持 binlog 归档")
		}
		if c.Engine == "mysql" && p.Method != "logical" {
			root, e := primary(c)
			add("物理工具兼容", e == nil && (root.Version == "8.0.24" || root.Version == "8.0.25" || root.Version == "8.0.26"), "当前固定工具为 XtraBackup 8.0.26-18；仅开放已选择的 8.0.24–26 兼容配置")
		}
		if p.Method == "wal-archive" || p.Method == "differential" {
			add("归档引擎", c.Engine == "postgresql", "WAL 归档与差异备份仅用于 PostgreSQL")
		}
		if c.Engine == "postgresql" && p.Method != "logical" {
			add("原生归档已配置", flag(object(c.Policy)["archiveConfigured"]), "请先配置 pgBackRest 与 WAL 归档")
		}
		if c.Engine == "redis" {
			add("Redis 备份类型", p.Method == "rdb", "Redis 使用 RDB 分片备份")
		}
		if p.RepositoryID > 0 {
			var r model.Repository
			e := s.DB.First(&r, p.RepositoryID).Error
			add("备份仓库", e == nil && r.GroupID == c.GroupID && r.Status == "healthy", "仓库必须属于当前业务分组且连接检查成功")
		}
		if c.Topology == "redis-cluster" {
			for _, m := range c.Members {
				var slots []string
				_ = json.Unmarshal(JSON(object(m.Metrics)["slots"]), &slots)
				for _, slot := range slots {
					if strings.HasPrefix(slot, "[") {
						add("槽位稳定", false, "迁槽期间不能执行常规分片备份")
					}
				}
			}
		}
	case "configure":
		if p.MemberID > 0 {
			m, e := member(c, p.MemberID)
			add("单实例配置", e == nil && m.Kind == "data" && c.Engine == "mysql", "仅 MySQL 支持工作台单实例参数范围；其他集群须通过原生管理器统一配置")
		}
		if p.RuntimeOnly {
			add("运行时范围", c.Engine == "mysql" && p.MemberID > 0, "运行时变更须绑定实际 MySQL 实例")
		}
		add("参数校验", len(p.Changes) > 0 && validateChanges(c.Engine, p.Changes) == nil, "仅允许支持范围内的运行参数，不能直接改复制或选主配置")
	case "restart", "rebuild-replica":
		if action == "restart" && p.MemberID > 0 {
			m, e := member(c, p.MemberID)
			add("重启角色", e == nil && (c.Topology == "standalone" || m.Role == "replica"), "主节点须先计划切换；不指定节点时仅逐台重启副本")
		}
		for _, m := range c.Members {
			if m.Kind == "data" && (p.MemberID == 0 || p.MemberID == m.ID) {
				add("服务归属 "+m.Name, m.Managed && m.Service != "" && m.HostID != 0, "只能操作有明确归属的受管服务")
			}
		}
		if action == "rebuild-replica" {
			m, e := member(c, p.MemberID)
			allowed := e == nil && (m.Role == "replica" || c.Topology == "postgresql-patroni" && m.Role == "offline" && m.State != "healthy" || c.Engine == "mysql" && m.Role == "initializing" && m.BaselinePending && flag(object(m.Metrics)["read_only"]) && flag(object(m.Metrics)["super_read_only"]))
			add("重建对象", allowed, "从主库重建会覆盖目标受管副本或待建立基线的新实例；不能覆盖可写主库")
		}
	case "remove-node":
		m, e := member(c, p.MemberID)
		allowed := e == nil && m.Kind == "data" && m.Role == "replica"
		if e == nil && c.Topology == "redis-cluster" && m.Role == "primary" {
			var slots []string
			_ = json.Unmarshal(JSON(object(m.Metrics)["slots"]), &slots)
			allowed = len(slots) == 0
		}
		add("移出对象", allowed, "主节点须先完成角色切换；Cluster 主节点须迁空槽位")
		if e == nil {
			add("受管服务", m.Managed && m.Service != "", "退役节点须有明确受管服务身份")
			replicas := 0
			for _, n := range c.Members {
				if n.ID != m.ID && n.Kind == "data" && n.Role == "replica" && n.State == "healthy" && (c.Topology != "redis-cluster" || n.ParentID == m.ParentID) {
					replicas++
				}
			}
			if m.Role == "replica" {
				add("剩余副本", replicas >= 1, "退役后至少保留一个健康副本")
			}
		}
	case "assign-replica":
		node, e := member(c, p.MemberID)
		target, e2 := member(c, p.CandidateID)
		add("副本与主分片", e == nil && e2 == nil && node.Kind == "data" && node.Role == "replica" && target.Kind == "data" && target.Role == "primary" && node.HostID != target.HostID, "请选择副本与不同故障域的主分片")
		if e == nil {
			remaining := 0
			oldEmpty := false
			for _, m := range c.Members {
				if m.NativeID == node.ParentID {
					var slots []string
					json.Unmarshal(JSON(object(m.Metrics)["slots"]), &slots)
					oldEmpty = len(slots) == 0
				}
				if m.ID != node.ID && m.ParentID == node.ParentID && m.Role == "replica" && m.State == "healthy" {
					remaining++
				}
			}
			add("原分片写入保护", oldEmpty || remaining > 0, "有业务槽位的原分片须保留至少一个健康副本")
		}
	case "reshard":
		add("槽位范围", len(p.Slots) > 0 && len(p.Slots) <= 1024, "每次迁移 1–1024 个明确槽位")
		for _, slot := range p.Slots {
			if slot < 0 || slot >= 16384 {
				add("槽位合法性", false, "槽位范围为 0–16383")
				break
			}
		}
		source, e := member(c, p.MemberID)
		target, e2 := member(c, p.CandidateID)
		add("分片目标", e == nil && e2 == nil && source.ID != target.ID && source.Role == "primary" && target.Role == "primary", "源和目标必须是不同的主分片")
		if e2 == nil {
			replicas := 0
			for _, m := range c.Members {
				if m.Kind == "data" && m.ParentID == target.NativeID && m.State == "healthy" && m.Role == "replica" && object(m.Metrics)["master_link_status"] == "up" {
					replicas++
				}
			}
			add("目标分片写入保护", replicas >= 1, "迁槽前必须为目标主分片配置至少一个有效副本")
		}
	case "purge-binlogs":
		add("归档清理边界", namePattern.MatchString(strings.ReplaceAll(p.BeforeFile, ".", "_")), "必须指定精确的 binlog 文件边界")
		add("归档依赖", s.binlogPurgeSafe(ctx, c, p.BeforeFile), "只有已验证归档、所有副本不再依赖且位于恢复窗口之外的日志可清理")
	}
	if p.HostID > 0 && !s.IsAdmin(user) {
		ok, e := cmdbdao.NewCmdbAssetPermissionDao(s.DB).CanUserAccess(user, "host", p.HostID, "command")
		add("目标主机权限", e == nil && ok, "需要目标 CMDB 主机的部署权限")
	}
	for _, m := range p.Members {
		if !s.IsAdmin(user) {
			ok, e := cmdbdao.NewCmdbAssetPermissionDao(s.DB).CanUserAccess(user, "host", m.HostID, "command")
			add("新增节点权限", e == nil && ok, "需要目标主机的部署权限")
		}
	}
	return checks
}

func (s *Service) Plan(ctx context.Context, id, user uint, action string, p Parameters) (*model.OperationPlan, error) {
	if e := s.Authorize(user, id, actionPermission(action)); e != nil {
		return nil, e
	}
	c, e := s.Cluster(id)
	if e != nil {
		return nil, e
	}
	checks := s.checks(ctx, c, action, p, user)
	executable := true
	for _, c := range checks {
		executable = executable && c.Passed
	}
	plan := &model.OperationPlan{ClusterID: id, Revision: c.Revision, Action: action, Parameters: JSON(p), Checks: JSON(checks), Executable: executable, UserID: user, ExpiresAt: time.Now().Add(10 * time.Minute)}
	if e = s.DB.Create(plan).Error; e != nil {
		return nil, e
	}
	// A plan is a unique confirmed intent. Reusing this public identifier on
	// retries does not depend on browser secure-context crypto APIs.
	plan.IdempotencyKey = fmt.Sprintf("db-plan-%d", plan.ID)
	return plan, nil
}

// Submit preserves the name-confirmation contract for existing callers.
func (s *Service) Submit(ctx context.Context, id, user, planID uint, key, confirm string) (*model.Operation, error) {
	c, e := s.Cluster(id)
	if e != nil {
		return nil, e
	}
	if confirm != c.Name {
		return nil, errors.New("集群确认信息不匹配")
	}
	return s.SubmitConfirmed(ctx, id, user, planID, key, true)
}

// SubmitConfirmed records an explicit confirmation; plans, scopes and locking
// still apply even though the UI no longer requires a reason or typed name.
func (s *Service) SubmitConfirmed(ctx context.Context, id, user, planID uint, key string, confirmed bool) (*model.Operation, error) {
	if !confirmed {
		return nil, errors.New("请确认执行此操作")
	}
	if e := s.keyExecutionAllowed(ctx, user); e != nil {
		return nil, e
	}
	if len(key) < 8 || len(key) > 80 {
		return nil, errors.New("必须提供 8–80 字节的幂等键")
	}
	c, e := s.Cluster(id)
	if e != nil {
		return nil, e
	}
	var p model.OperationPlan
	if s.DB.Where("id = ? AND cluster_id = ? AND user_id = ?", planID, id, user).First(&p).Error != nil {
		return nil, errors.New("执行计划不存在")
	}
	if e = s.Authorize(user, id, actionPermission(p.Action)); e != nil {
		return nil, e
	}
	hash := sha256.Sum256(JSON([]any{planID, user, p.Action, p.Parameters}))
	requestHash := hex.EncodeToString(hash[:])
	var op model.Operation
	e = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("cluster_id = ? AND idempotency_key = ?", id, key).First(&op).Error; e == nil {
			if op.RequestHash != requestHash {
				return errors.New("幂等键已被其他请求使用")
			}
			return nil
		} else if e != gorm.ErrRecordNotFound {
			return e
		}
		var locked model.Cluster
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, id).Error; e != nil {
			return e
		}
		if !p.Executable || p.Revision != locked.Revision || p.ExpiresAt.Before(time.Now()) {
			return ErrConflict
		}
		if p.Action == "backup" {
			var params Parameters
			if json.Unmarshal(p.Parameters, &params) != nil {
				return ErrConflict
			}
			if params.RepositoryID > 0 {
				var repo model.Repository
				if tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&repo, params.RepositoryID).Error != nil || repo.GroupID != locked.GroupID || repo.Status != "healthy" {
					return errors.New("备份仓库已删除或状态发生变化，请重新预检")
				}
			}
		}
		op = model.Operation{APIKeyID: apikey.ID(ctx), ClusterID: id, PlanID: planID, Action: p.Action, Parameters: p.Parameters, IdempotencyKey: key, RequestHash: requestHash, Status: "queued", UserID: user}
		if e := tx.Create(&op).Error; e != nil {
			return e
		}
		if p.Action == "reconcile" {
			var params Parameters
			if json.Unmarshal(p.Parameters, &params) != nil {
				return ErrConflict
			}
			r := tx.Model(&model.Operation{}).Where("id = ? AND cluster_id = ? AND status = ?", params.OperationID, id, "unknown").Updates(map[string]any{"status": "reconciling", "worker_id": fmt.Sprintf("superseded-by-%d", op.ID)})
			if r.Error != nil || r.RowsAffected != 1 {
				return ErrConflict
			}
			r = tx.Model(&model.OperationLock{}).Where("cluster_id = ? AND operation_id = ?", id, params.OperationID).Update("operation_id", op.ID)
			if r.Error != nil || r.RowsAffected != 1 {
				return ErrConflict
			}
		} else if e := tx.Create(&model.OperationLock{ClusterID: id, OperationID: op.ID}).Error; e != nil {
			return ErrConflict
		}
		if c.DeploymentID != nil && p.Action != "reconcile" {
			if e := tx.Create(&cmdb.DatabaseAutomationLock{Key: fmt.Sprintf("deployment:%d", *c.DeploymentID), RunID: 0, CreatedAt: time.Now()}).Error; e != nil {
				return ErrConflict
			}
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return &op, nil
}

func (s *Service) RunWorker(ctx context.Context, worker string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	// Collection and scheduling keep running while a long operation is active.
	background, stop := context.WithCancel(ctx)
	defer stop()
	collectorDone := make(chan struct{})
	go func() { defer close(collectorDone); s.RunCollector(background) }()
	defer func() { stop(); <-collectorDone }()
	go func() {
		collect := time.NewTicker(30 * time.Second)
		defer collect.Stop()
		for {
			select {
			case <-background.Done():
				return
			case <-collect.C:
				s.reconcileAbandoned()
				s.scheduleBackups(background)
				s.PruneAudit(background)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			op, e := s.claim(worker)
			if e == nil && op != nil {
				s.run(ctx, *op)
			}
		}
	}
}
func (s *Service) claim(worker string) (*model.Operation, error) {
	var op model.Operation
	result := s.DB.Where("status = ?", "queued").Order("id").Limit(1).Find(&op)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	t := time.Now().UTC()
	r := s.DB.Model(&model.Operation{}).Where("id = ? AND status = ?", op.ID, "queued").Updates(map[string]any{"status": "preflight", "worker_id": worker, "started_at": t, "heartbeat_at": t})
	if r.Error != nil {
		return nil, r.Error
	}
	if r.RowsAffected != 1 {
		return nil, nil
	}
	op.Status = "preflight"
	op.WorkerID = worker
	return &op, nil
}
func (s *Service) run(ctx context.Context, op model.Operation) {
	if op.APIKeyID > 0 {
		ctx = apikey.WithKey(ctx, op.APIKeyID)
	}
	ctx = context.WithValue(ctx, operationContextKey{}, op)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				r := s.DB.Model(&model.Operation{}).Where("id = ? AND worker_id = ? AND status IN ?", op.ID, op.WorkerID, []string{"preflight", "running", "verifying"}).Update("heartbeat_at", time.Now().UTC())
				if r.Error != nil || r.RowsAffected != 1 {
					cancel()
					return
				}
			}
		}
	}()
	mutated := false
	defer func() {
		if recover() != nil {
			status := "failed"
			if mutated {
				status = "unknown"
			}
			s.complete(op, status, "执行器异常，需核对远端实际状态")
		}
	}()
	// Claiming the job moves it to preflight, which blocks new collectors. Drain
	// any snapshot started while it was queued before checking the fresh state.
	for {
		var lease model.CollectionLease
		if s.DB.First(&lease, "cluster_id = ? AND `until` > ?", op.ClusterID, time.Now()).Error != nil {
			break
		}
		select {
		case <-ctx.Done():
			s.complete(op, "failed", "等待状态采集完成时任务取消")
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
	c, e := s.Cluster(op.ClusterID)
	if e != nil {
		s.complete(op, "failed", e.Error())
		return
	}
	var p Parameters
	if json.Unmarshal(op.Parameters, &p) != nil {
		s.complete(op, "failed", "任务参数无效")
		return
	}
	if e = s.Authorize(op.UserID, c.ID, actionPermission(op.Action)); e != nil {
		s.complete(op, "failed", "执行前授权已失效")
		return
	}
	if e = s.keyExecutionAllowed(ctx, op.UserID); e != nil {
		s.complete(op, "failed", "提交任务的账号或 API Key 已失效，或不再具有数据库执行权限")
		return
	}
	var plan model.OperationPlan
	if s.DB.First(&plan, op.PlanID).Error != nil || plan.Revision != c.Revision {
		s.complete(op, "failed", "执行前拓扑版本已变化，请重新预检")
		return
	}
	if op.Action != "provision" && op.Action != "adopt" && op.Action != "reconcile" {
		if _, e = s.Refresh(ctx, c.ID); e != nil {
			s.complete(op, "failed", e.Error())
			return
		}
		fresh, _ := s.Cluster(c.ID)
		if fresh.Revision != c.Revision {
			s.complete(op, "failed", "执行前发现拓扑变化，请重新预检")
			return
		}
		c = fresh
	}
	checks := s.checks(ctx, c, op.Action, p, op.UserID)
	if op.Action == "reconcile" {
		checks = nil
	} // Original job is now atomically marked reconciling; executor validates its ownership.
	for _, check := range checks {
		if !check.Passed {
			s.complete(op, "failed", check.Name+"："+check.Message)
			return
		}
	}
	r := s.DB.Model(&model.Operation{}).Where("id = ? AND worker_id = ? AND status = ?", op.ID, op.WorkerID, "preflight").Update("status", "running")
	if r.Error != nil || r.RowsAffected != 1 {
		return
	}
	mutated = op.Action != "refresh" && op.Action != "collect-storage"
	e = s.execute(ctx, c, op, p)
	if e != nil {
		status := "failed"
		if mutated {
			status = "unknown"
		}
		s.complete(op, status, e.Error())
		return
	}
	r = s.DB.Model(&model.Operation{}).Where("id = ? AND worker_id = ? AND status = ?", op.ID, op.WorkerID, "running").Update("status", "verifying")
	if r.Error != nil || r.RowsAffected != 1 {
		return
	}
	if op.Action != "adopt" {
		if _, e = s.Refresh(ctx, c.ID); e != nil {
			s.complete(op, "unknown", "执行已返回，但状态核验失败")
			return
		}
	}
	s.complete(op, "succeeded", "操作完成，实际结果已核验")
}
func (s *Service) complete(op model.Operation, status, message string) {
	if op.Action == "reconcile" && status == "failed" {
		status = "unknown"
	}
	if len(message) > 1000 {
		message = message[:1000]
	}
	_ = s.DB.Transaction(func(tx *gorm.DB) error {
		r := tx.Model(&model.Operation{}).Where("id = ? AND worker_id = ?", op.ID, op.WorkerID).Updates(map[string]any{"status": status, "message": message, "finished_at": time.Now().UTC()})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected != 1 {
			return nil
		}
		if status != "unknown" {
			var c model.Cluster
			if e := tx.First(&c, op.ClusterID).Error; e != nil {
				return e
			}
			if c.DeploymentID != nil {
				if e := tx.Where("`key` = ? AND run_id = 0", fmt.Sprintf("deployment:%d", *c.DeploymentID)).Delete(&cmdb.DatabaseAutomationLock{}).Error; e != nil {
					return e
				}
			}
			if e := tx.Where("cluster_id = ? AND operation_id = ?", op.ClusterID, op.ID).Delete(&model.OperationLock{}).Error; e != nil {
				return e
			}
		}
		severity := "info"
		if status != "succeeded" {
			severity = "warning"
		}
		return tx.Create(&model.Event{ClusterID: op.ClusterID, OperationID: op.ID, UserID: op.UserID, Kind: op.Action, Severity: severity, Message: message}).Error
	})
}
func (s *Service) reconcileAbandoned() {
	s.DB.Model(&model.Operation{}).Where("status IN ? AND heartbeat_at < ?", []string{"preflight", "running", "verifying"}, time.Now().Add(-90*time.Second)).Updates(map[string]any{"status": "unknown", "message": "Worker 心跳已过期，保留互斥锁；核对远端任务后才能继续"})
}
func (s *Service) step(op model.Operation, name string, fn func(func(string)) error) error {
	if e := s.executionOwned(context.WithValue(context.Background(), operationContextKey{}, op)); e != nil {
		return e
	}
	st := model.OperationStep{OperationID: op.ID, Name: name, Status: "running", StartedAt: time.Now().UTC()}
	if e := s.DB.Create(&st).Error; e != nil {
		return e
	}
	save := func(v string) {
		if len(v) > 2<<20 {
			v = v[len(v)-(2<<20):]
		}
		s.DB.Model(&st).Update("log", v)
	}
	e := fn(save)
	status := "succeeded"
	if e != nil {
		status = "failed"
	}
	s.DB.Model(&st).Updates(map[string]any{"status": status, "finished_at": time.Now().UTC()})
	return e
}
