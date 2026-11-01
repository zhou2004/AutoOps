package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/database/model"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

func (s *Service) execute(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	switch op.Action {
	case "configure-vip":
		return s.configureVIP(ctx, c, op, p.VIP)
	case "verify-vip":
		return s.verifyVIP(ctx, c, op)
	case "update-patroni":
		return s.updatePatroni(ctx, c, op)
	case "balance-primaries":
		return s.balancePrimaries(ctx, c, op)
	case "reconcile":
		return s.reconcileOperation(ctx, c, op, p)
	case "refresh":
		_, e := s.Refresh(ctx, c.ID)
		return e
	case "collect-storage":
		return s.collectStorage(ctx, c, op)
	case "adopt":
		return s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Updates(map[string]any{"management": p.Management, "revision": gorm.Expr("revision + 1")}).Error
	case "provision":
		if e := s.step(op, "Ansible 安装独立实例与控制组件", func(log func(string)) error { _, e := s.ansible(ctx, c, c.Members, "site.yml", nil, log); return e }); e != nil {
			return e
		}
		if c.Topology == "redis-cluster" {
			if e := s.step(op, "建立分片与副本关系", func(log func(string)) error { return s.bootstrapRedis(ctx, c, log) }); e != nil {
				return e
			}
		}
		if c.Topology == "mysql-gtid" {
			if e := s.step(op, "登记 Orchestrator 原生复制拓扑", func(log func(string)) error {
				for _, m := range c.Members {
					if m.Kind == "data" {
						if _, e := s.orchestrator(ctx, c, "discover/"+instancePath(m)); e != nil {
							return e
						}
					}
				}
				return nil
			}); e != nil {
				return e
			}
			if e := s.configureFencing(ctx, c, op); e != nil {
				return e
			}
		}
		if e := s.waitHealthy(ctx, c.ID, 120*time.Second, 0); e != nil {
			return e
		}
		return s.registerSources(c)
	case "enter-maintenance", "exit-maintenance":
		return s.maintenance(ctx, c, op)
	case "repair-replication":
		return s.repairMySQLReplication(ctx, c, op, p)
	case "rejoin":
		return s.rejoinMySQL(ctx, c, op, p)
	case "acknowledge-recovery":
		return s.acknowledgeMySQLRecovery(ctx, c, op)
	case "resume-writes":
		return s.resumeMySQLWrites(ctx, c, op)
	case "configure-semisync":
		return s.configureSemiSync(ctx, c, op, p)
	case "configure-archive":
		return s.configureArchive(ctx, c, op)
	case "configure-fencing":
		return s.configureFencing(ctx, c, op)
	case "configure-endpoint":
		return s.configureEndpoints(ctx, c, op, p)
	case "switchover", "recover":
		return s.switchover(ctx, c, op, p)
	case "restart":
		return s.restart(ctx, c, op, p)
	case "redis-bgsave", "redis-aof-rewrite":
		return s.redisPersistence(ctx, c, op, p)
	case "configure":
		return s.configure(ctx, c, op, p)
	case "expire-backup":
		return s.expireBackup(ctx, c, op, p)
	case "backup":
		return s.backup(ctx, c, op, p)
	case "restore", "verify-backup":
		return s.restore(ctx, c, op, p)
	case "reshard":
		return s.reshard(ctx, c, op, p)
	case "add-node", "remove-node", "rebuild-replica", "assign-replica":
		return s.membership(ctx, c, op, p)
	case "purge-binlogs":
		if !s.binlogPurgeSafe(ctx, c, p.BeforeFile) {
			return errors.New("清理边界仍被复制或恢复链引用")
		}
		primary, e := primary(c)
		if e != nil {
			return e
		}
		db, e := s.sql(ctx, c, *primary, "")
		if e != nil {
			return e
		}
		defer db.Close()
		_, e = db.ExecContext(ctx, "PURGE BINARY LOGS TO ?", p.BeforeFile)
		return e
	}
	return errors.New("未知操作")
}
func primary(c *model.Cluster) (*model.Member, error) {
	var found *model.Member
	for i := range c.Members {
		if c.Members[i].Kind == "data" && c.Members[i].Role == "primary" {
			if c.Topology == "redis-cluster" {
				return &c.Members[i], nil
			}
			if found != nil {
				return nil, errors.New("存在多个复制根节点，需先核验原生拓扑与旧主状态")
			}
			found = &c.Members[i]
		}
	}
	if found != nil {
		return found, nil
	}
	return nil, errors.New("没有已确认的主节点")
}
func (s *Service) waitHealthy(ctx context.Context, id uint, timeout time.Duration, expectedPrimary uint) error {
	deadline := time.Now().Add(timeout)
	for {
		c, e := s.Refresh(ctx, id)
		if e == nil {
			h := object(c.Health)
			ok := h["database"] == "healthy" && h["replication"] != "degraded"
			if expectedPrimary > 0 {
				m, e := member(c, expectedPrimary)
				ok = ok && e == nil && m.Role == "primary"
			}
			if ok {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return errors.New("数据库或复制尚未通过结果核验，请检查成员状态")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
func (s *Service) orchestrator(ctx context.Context, c *model.Cluster, path string) ([]byte, error) {
	for _, m := range c.Members {
		if m.Kind != "orchestrator" {
			continue
		}
		if _, e := s.control(ctx, c, m, http.MethodGet, "/api/leader-check", nil); e != nil {
			continue
		}
		raw, e := s.control(ctx, c, m, http.MethodGet, "/api/"+path, nil)
		if e != nil {
			return nil, e
		}
		response := object(raw)
		if response["Code"] == "ERROR" {
			return nil, errors.New("Orchestrator 拒绝操作，请核对复制与隔离状态")
		}
		return raw, nil
	}
	return nil, errors.New("未找到可用的 Orchestrator Leader")
}
func instancePath(m model.Member) string {
	return url.PathEscape(m.Address) + "/" + strconv.Itoa(m.Port)
}
func patroniFor(c *model.Cluster, data model.Member) (model.Member, error) {
	for _, m := range c.Members {
		if m.Kind == "patroni" && m.DataMemberID == data.ID && data.ID != 0 {
			return m, nil
		}
	}
	return model.Member{}, errors.New("未找到节点对应的 Patroni")
}
func (s *Service) switchover(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	if c.Topology == "redis-sentinel" {
		return s.sentinelSwitchover(ctx, c, op)
	}
	candidate, e := member(c, p.CandidateID)
	if e != nil {
		return e
	}
	old, e := primary(c)
	if e != nil {
		return e
	}
	if c.Engine == "mysql" {
		if op.Action == "switchover" {
			if e = s.step(op, "收敛写入并等待候选追平 GTID", func(log func(string)) error {
				db, e := s.sql(ctx, c, *old, "")
				if e != nil {
					return e
				}
				defer db.Close()
				if _, e = db.ExecContext(ctx, "SET GLOBAL super_read_only=ON"); e != nil {
					return errors.New("原主只读保护未生效")
				}
				var gtid string
				if e = db.QueryRowContext(ctx, "SELECT @@global.gtid_executed").Scan(&gtid); e != nil {
					return e
				}
				replica, e := s.sql(ctx, c, *candidate, "")
				if e != nil {
					return e
				}
				defer replica.Close()
				var timeout int
				if e = replica.QueryRowContext(ctx, "SELECT WAIT_FOR_EXECUTED_GTID_SET(?,30)", gtid).Scan(&timeout); e != nil || timeout != 0 {
					return errors.New("候选未追平，原主保持保护性只读，需人工核验")
				}
				log("原主写入已收敛；候选已持久化指定 GTID 集合")
				return nil
			}); e != nil {
				return e
			}
		}
		// Recovery is only enabled after the independent hook has been installed.
		if op.Action == "recover" && !flag(object(c.Policy)["fencingInstalled"]) {
			return errors.New("实际 Orchestrator 恢复链路尚未安装并验证隔离钩子")
		}
		action := "graceful-master-takeover-auto"
		if op.Action == "recover" {
			action = "recover"
		}
		if e = s.step(op, "Orchestrator 执行角色切换", func(log func(string)) error {
			_, e := s.orchestrator(ctx, c, action+"/"+instancePath(*old)+"/"+instancePath(*candidate))
			return e
		}); e != nil {
			return e
		}
	} else if c.Engine == "postgresql" {
		if op.Action == "recover" {
			if e = s.confirmStopped(ctx, *old); e != nil {
				return e
			}
			if e = s.DB.Model(&model.Member{}).Where("id = ? AND cluster_id = ?", old.ID, c.ID).Update("role", "offline").Error; e != nil {
				return e
			}
		}
		control, e := patroniFor(c, *old)
		if e != nil {
			return e
		}
		body := map[string]any{"leader": old.Name, "candidate": candidate.Name}
		if op.Action == "recover" {
			control, e = patroniFor(c, *candidate)
			if e != nil {
				return e
			}
			delete(body, "leader")
		}
		path := "/switchover"
		if op.Action == "recover" {
			path = "/failover"
		}
		if e = s.step(op, "Patroni 校验候选并执行切换", func(log func(string)) error {
			_, e := s.control(ctx, c, control, http.MethodPost, path, bytes.NewReader(JSON(body)))
			return e
		}); e != nil {
			return e
		}
	} else if c.Topology == "redis-cluster" {
		client, e := s.redis(c, *candidate)
		if e != nil {
			return e
		}
		defer client.Close()
		// Never use FORCE or TAKEOVER; Redis must obtain native majority approval.
		if e = s.step(op, "Redis Cluster 原生计划切换", func(log func(string)) error { return client.Do(ctx, "CLUSTER", "FAILOVER").Err() }); e != nil {
			return e
		}

	}
	if e = s.step(op, "核验角色与复制", func(log func(string)) error {
		if op.Action == "recover" {
			return s.waitRecovered(ctx, c.ID, candidate.ID)
		}
		return s.waitHealthy(ctx, c.ID, 90*time.Second, candidate.ID)
	}); e != nil {
		return e
	}
	return s.step(op, "等待业务入口收敛并核验读写角色", func(log func(string)) error {
		log("最多等待 30 秒，连续核验入口连接与读写角色；仅重试只读探测，不重放切换命令")
		if err := s.waitEndpoints(ctx, c); err != nil {
			return err
		}
		log("业务入口核验通过")
		return nil
	})
}
func (s *Service) restart(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	for _, m := range c.Members {
		if m.Kind != "data" || (p.MemberID > 0 && p.MemberID != m.ID) {
			continue
		}
		if p.MemberID == 0 && m.Role == "primary" && c.Topology != "standalone" {
			continue
		}
		if m.Role == "primary" && c.Topology != "standalone" {
			return errors.New("主节点需要先完成计划切换，再作为副本重启")
		}
		if e := s.step(op, "重启并核验 "+m.Name, func(log func(string)) error {
			var e error
			if c.Topology == "postgresql-patroni" {
				var control model.Member
				control, e = patroniFor(c, m)
				if e == nil {
					_, e = s.control(ctx, c, control, http.MethodPost, "/restart", bytes.NewReader([]byte("{}")))
				}
			} else {
				_, e = s.ansible(ctx, c, []model.Member{m}, "service.yml", map[string]any{"service_action": "restarted"}, log)
			}
			if e != nil {
				return e
			}
			return s.waitHealthy(ctx, c.ID, 90*time.Second, 0)
		}); e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) redisPersistence(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	for _, m := range c.Members {
		if m.Kind != "data" || (p.MemberID > 0 && p.MemberID != m.ID) {
			continue
		}
		client, e := s.redis(c, m)
		if e != nil {
			return e
		}
		e = s.step(op, "持久化 "+m.Name, func(log func(string)) error {
			if e := s.persistencePreflight(ctx, c, m, client); e != nil {
				return e
			}
			field, status := "rdb_bgsave_in_progress", "rdb_last_bgsave_status"
			command := "BGSAVE"
			if op.Action == "redis-aof-rewrite" {
				field, status, command = "aof_rewrite_in_progress", "aof_last_bgrewrite_status", "BGREWRITEAOF"
			}
			if e := client.Do(ctx, command).Err(); e != nil {
				return e
			}
			for {
				raw, e := client.Info(ctx, "persistence").Result()
				if e != nil {
					return e
				}
				metrics := infoMap(raw)
				if metrics[field] == "0" {
					if metrics[status] != "ok" {
						return errors.New("持久化任务返回失败")
					}
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
				}
			}
		})
		client.Close()
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) persistencePreflight(ctx context.Context, c *model.Cluster, m model.Member, client *redis.Client) error {
	raw, e := client.Info(ctx).Result()
	if e != nil {
		return e
	}
	metrics := infoMap(raw)
	if metrics["rdb_bgsave_in_progress"] == "1" || metrics["aof_rewrite_in_progress"] == "1" {
		return errors.New("已有重型持久化任务执行中")
	}
	if m.Role == "replica" && metrics["master_link_status"] != "up" {
		return errors.New("复制链路异常，暂不执行持久化")
	}
	if m.HostID == 0 || m.DataDir == "" {
		return errors.New("持久化预检需要 CMDB 主机和数据目录")
	}
	result, e := s.filesystem(ctx, m)
	if e != nil {
		return e
	}
	free := number(result["availableBytes"])
	memory := number(result["availableMemory"])
	required := number(metrics["used_memory"])*2 + 64*1024*1024
	if free < required || memory < required {
		return errors.New("磁盘或内存余量不足以安全执行后台持久化")
	}
	return nil
}
func (s *Service) bootstrapRedis(ctx context.Context, c *model.Cluster, log func(string)) error {
	nodes := []model.Member{}
	for _, m := range c.Members {
		if m.Kind == "data" {
			nodes = append(nodes, m)
		}
	}
	clients := map[uint]*redis.Client{}
	defer func() {
		for _, v := range clients {
			v.Close()
		}
	}()
	ids := map[uint]string{}
	for _, m := range nodes {
		client, e := s.redis(c, m)
		if e != nil {
			return e
		}
		clients[m.ID] = client
		raw, e := client.ClusterNodes(ctx).Result()
		if e != nil {
			return e
		}
		n := parseClusterNodes(raw)
		if len(n) != 1 || len(n[0]["slots"].([]string)) != 0 {
			return errors.New("拒绝重放集群初始化：节点已有成员或槽位，须先核验远端状态")
		}
		ids[m.ID] = text(n[0]["id"])
	}
	seed := nodes[0]
	for _, m := range nodes[1:] {
		if e := clients[seed.ID].ClusterMeet(ctx, m.Address, strconv.Itoa(m.Port)).Err(); e != nil {
			return e
		}
	}
	if e := waitUntil(ctx, 30*time.Second, func() bool {
		for _, client := range clients {
			raw, e := client.ClusterNodes(ctx).Result()
			if e != nil || len(parseClusterNodes(raw)) != len(nodes) {
				return false
			}
		}
		return true
	}); e != nil {
		return e
	}
	masters := map[string]model.Member{}
	order := []model.Member{}
	for _, m := range nodes {
		if m.Role == "primary" {
			masters[m.Shard] = m
			order = append(order, m)
		}
	}
	if len(order) != 3 {
		return errors.New("预期三个主分片")
	}
	for i, m := range order {
		start, end := i*16384/3, (i+1)*16384/3
		slots := make([]int, 0, end-start)
		for slot := start; slot < end; slot++ {
			slots = append(slots, slot)
		}
		if e := clients[m.ID].ClusterAddSlots(ctx, slots...).Err(); e != nil {
			return e
		}
	}
	for _, m := range nodes {
		if m.Role == "primary" {
			continue
		}
		master, ok := masters[m.Shard]
		if !ok || master.HostID == m.HostID {
			return errors.New("分片副本必须位于其他故障域")
		}
		if e := clients[m.ID].ClusterReplicate(ctx, ids[master.ID]).Err(); e != nil {
			return e
		}
	}
	log("已分配 16384 个槽位；三个复制组各包含一个主节点和两个跨主机副本")
	return nil
}
func waitUntil(ctx context.Context, d time.Duration, predicate func() bool) error {
	end := time.Now().Add(d)
	for {
		if predicate() {
			return nil
		}
		if time.Now().After(end) {
			return errors.New("结果核验超时")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
func cleanError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "password") {
		return errors.New("执行失败，请检查凭据和授权")
	}
	return err
}

// Backward-compatible inference is permitted only for an unambiguous one-to-one
// host mapping; subsequent operations use the instance identity.
func linkPatroniMembers(c *model.Cluster) {
	for i := range c.Members {
		m := &c.Members[i]
		if m.Kind != "patroni" || m.DataMemberID != 0 {
			continue
		}
		var id uint
		count := 0
		for _, data := range c.Members {
			if data.Kind == "data" && data.Address == m.Address {
				id = data.ID
				count++
			}
		}
		if count == 1 {
			m.DataMemberID = id
		}
	}
}
