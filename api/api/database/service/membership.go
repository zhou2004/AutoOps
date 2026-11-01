package service

import (
	"context"
	"dodevops-api/pkg/rediscompat"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"gorm.io/gorm"
)

func (s *Service) reshard(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	source, e := member(c, p.MemberID)
	if e != nil {
		return e
	}
	target, e := member(c, p.CandidateID)
	if e != nil {
		return e
	}
	src, e := s.redis(c, *source)
	if e != nil {
		return e
	}
	defer src.Close()
	dst, e := s.redis(c, *target)
	if e != nil {
		return e
	}
	defer dst.Close()
	username, password, e := s.memberAccount(c, *target)
	if e != nil {
		return e
	}
	return s.step(op, "迁移槽位并持久记录每批进度", func(log func(string)) error {
		for i, slot := range p.Slots {
			if e = s.executionOwned(ctx); e != nil {
				return e
			}
			// Read the actual owner and migration markers on each iteration. Completed
			// slots can be reconciled after interruption, never reset to the old owner.
			owners, e := rediscompat.Slots(ctx, src, source.Address)
			if e != nil {
				return e
			}
			owner := ""
			for _, r := range owners {
				if slot >= r.Start && slot <= r.End && len(r.Nodes) > 0 {
					owner = r.Nodes[0].ID
				}
			}
			if owner == target.NativeID {
				if e = s.commitSlot(ctx, c, *source, *target, slot); e != nil {
					return e
				}
				log(fmt.Sprintf("%d/%d 槽位 %d 已在目标节点", i+1, len(p.Slots), slot))
				continue
			}
			if owner != source.NativeID {
				return errors.New("槽位所有者已改变，请重新发现拓扑")
			}
			if e = dst.Do(ctx, "CLUSTER", "SETSLOT", slot, "IMPORTING", source.NativeID).Err(); e != nil {
				return e
			}
			if e = src.Do(ctx, "CLUSTER", "SETSLOT", slot, "MIGRATING", target.NativeID).Err(); e != nil {
				return e
			}
			batches := 0
			for {
				if e = s.executionOwned(ctx); e != nil {
					return e
				}
				keys, e := src.ClusterGetKeysInSlot(ctx, slot, 100).Result()
				if e != nil {
					return e
				}
				if len(keys) == 0 {
					break
				}
				args := []any{"MIGRATE", target.Address, target.Port, "", 0, 5000, "AUTH2", username, password, "KEYS"}
				for _, key := range keys {
					args = append(args, key)
				}
				if e = src.Do(ctx, args...).Err(); e != nil {
					return errors.New("迁移批次结果不明，保留 IMPORTING/MIGRATING 状态；重新读取键归属后继续，禁止盲目 REPLACE")
				}
				batches++
				log(fmt.Sprintf("槽位 %d，已完成 %d 批，每批最多 100 个键", slot, batches))
			}
			if e = s.commitSlot(ctx, c, *source, *target, slot); e != nil {
				return e
			}
			if e = s.DB.Create(&model.Event{ClusterID: c.ID, OperationID: op.ID, UserID: op.UserID, Kind: "slot-migrated", Severity: "info", Message: fmt.Sprintf("槽位 %d 已迁移到成员 %d", slot, target.ID)}).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

func (s *Service) commitSlot(ctx context.Context, c *model.Cluster, source, target model.Member, slot int) error {
	// Redis can turn the source into a replica immediately after its last slot
	// is transferred. Notify destination/source first and inspect current roles
	// instead of submitting SETSLOT against a stale list of masters.
	nodes := []model.Member{target, source}
	for _, m := range c.Members {
		if m.Kind == "data" && m.ID != source.ID && m.ID != target.ID {
			nodes = append(nodes, m)
		}
	}
	for _, m := range nodes {
		if e := s.executionOwned(ctx); e != nil {
			return e
		}
		client, e := s.redis(c, m)
		if e != nil {
			return e
		}
		info, e := client.Info(ctx, "replication").Result()
		if e == nil && infoMap(info)["role"] == "master" {
			e = client.Do(ctx, "CLUSTER", "SETSLOT", slot, "NODE", target.NativeID).Err()
			if e != nil && strings.Contains(e.Error(), "only with masters") {
				current, err := client.Info(ctx, "replication").Result()
				if err == nil && infoMap(current)["role"] == "slave" {
					e = nil
				}
			}
		}
		if e == nil {
			owners, err := rediscompat.Slots(ctx, client, m.Address)
			if err != nil {
				e = err
			} else {
				matched := false
				for _, r := range owners {
					if slot >= r.Start && slot <= r.End && len(r.Nodes) > 0 && r.Nodes[0].ID == target.NativeID {
						matched = true
					}
				}
				if !matched {
					e = errors.New("槽位归属尚未在所有节点收敛，保留迁移记录")
				}
			}
		}
		client.Close()
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) membership(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	if op.Action == "assign-replica" {
		node, e := member(c, p.MemberID)
		if e != nil {
			return e
		}
		parent, e := member(c, p.CandidateID)
		if e != nil {
			return e
		}
		client, e := s.redis(c, *node)
		if e != nil {
			return e
		}
		defer client.Close()
		if e = client.ClusterReplicate(ctx, parent.NativeID).Err(); e != nil {
			return e
		}
		return s.waitHealthy(ctx, c.ID, 120*time.Second, 0)
	}
	if op.Action == "add-node" {
		return s.addMember(ctx, c, op, p)
	}
	m, e := member(c, p.MemberID)
	if e != nil {
		return e
	}
	if op.Action == "rebuild-replica" {
		if c.Topology == "postgresql-patroni" {
			if m.Role == "offline" && m.State != "healthy" {
				root, e := primary(c)
				if e != nil || root.State != "healthy" {
					return errors.New("需要已确认的健康新主，不能启动未经核验的旧主")
				}
				if e = s.step(op, "保留旧目录并从原生主库重建离线副本", func(log func(string)) error {
					if _, err := s.ansible(ctx, c, []model.Member{*m}, "patroni-update.yml", nil, log); err != nil {
						return err
					}
					_, err := s.ansible(ctx, c, []model.Member{*m}, "pg-offline-rebuild.yml", map[string]any{"operation_id": op.ID}, log)
					return err
				}); e != nil {
					return e
				}
				return s.waitHealthy(ctx, c.ID, 5*time.Minute, root.ID)
			}
			control, e := patroniFor(c, *m)
			if e != nil {
				return e
			}
			db, e := s.sql(ctx, c, *m, "postgres")
			if e != nil {
				return e
			}
			var before time.Time
			e = db.QueryRowContext(ctx, "SELECT pg_postmaster_start_time()").Scan(&before)
			db.Close()
			if e != nil {
				return e
			}
			if _, e = s.control(ctx, c, control, http.MethodPost, "/reinitialize", strings.NewReader(`{"force":false}`)); e != nil {
				return e
			}
			// Patroni returns 202 before rebuilding. A healthy snapshot of the
			// original process cannot prove that the requested rebuild completed.
			if e = waitUntil(ctx, 5*time.Minute, func() bool {
				db, err := s.sql(ctx, c, *m, "postgres")
				if err != nil {
					return false
				}
				defer db.Close()
				var after time.Time
				var replica bool
				err = db.QueryRowContext(ctx, "SELECT pg_postmaster_start_time(), pg_is_in_recovery()").Scan(&after, &replica)
				return err == nil && replica && after.After(before)
			}); e != nil {
				return e
			}
			return s.waitHealthy(ctx, c.ID, 5*time.Minute, 0)
		}
		if c.Engine == "redis" {
			primary, e := primary(c)
			if e != nil {
				return e
			}
			if c.Topology == "redis-cluster" {
				for i := range c.Members {
					if c.Members[i].NativeID == m.ParentID {
						primary = &c.Members[i]
						break
					}
				}
			}
			client, e := s.redis(c, *m)
			if e != nil {
				return e
			}
			defer client.Close()
			if c.Topology == "redis-cluster" {
				e = client.ClusterReplicate(ctx, primary.NativeID).Err()
			} else {
				e = client.SlaveOf(ctx, primary.Address, strconv.Itoa(primary.Port)).Err()
			}
			if e != nil {
				return e
			}
			return s.waitHealthy(ctx, c.ID, 120*time.Second, 0)
		}
		// MySQL rebuild uses a full physical clone with an explicit owned directory.
		donor, e := mysqlRejoinDonor(c, m.ID)
		if e != nil {
			return e
		}
		recipient := *m
		recipient.Role = "replica"
		return s.step(op, "从一致性备份重建受管 MySQL 副本", func(log func(string)) error {
			_, e := s.ansible(ctx, c, []model.Member{recipient}, "rebuild.yml", map[string]any{"operation_id": op.ID, "donor": donor}, log)
			if e != nil {
				return e
			}
			if e = s.waitHealthy(ctx, c.ID, 5*time.Minute, 0); e != nil {
				return e
			}
			if e = s.DB.Model(&model.Member{}).Where("id = ?", m.ID).Update("baseline_pending", false).Error; e != nil {
				return e
			}
			fresh, e := s.Cluster(c.ID)
			if e != nil {
				return e
			}
			return s.syncMemberEndpoints(ctx, fresh, op)
		})
	}
	if !m.Managed {
		return errors.New("接管节点只移除管理关联可能保留原生复制关系；请先完成独立退役流程")
	}
	if c.Topology == "redis-cluster" {
		client, e := s.redis(c, *m)
		if e != nil {
			return e
		}
		raw, e := client.ClusterNodes(ctx).Result()
		client.Close()
		if e != nil {
			return e
		}
		for _, n := range parseClusterNodes(raw) {
			if text(n["id"]) == m.NativeID && len(n["slots"].([]string)) > 0 {
				return errors.New("节点仍持有槽位或迁移标记，请先迁空")
			}
		}
		for _, node := range c.Members {
			if node.Kind != "data" || node.ID == m.ID {
				continue
			}
			client, e := s.redis(c, node)
			if e != nil {
				return e
			}
			e = client.ClusterForget(ctx, m.NativeID).Err()
			client.Close()
			if e != nil && !strings.Contains(e.Error(), "Unknown node") {
				return e
			}
		}
	}
	if _, e = s.ansible(ctx, c, []model.Member{*m}, "service.yml", map[string]any{"service_action": "stopped", "retire_service": true}, nil); e != nil {
		return e
	}
	if c.Engine == "mysql" && c.Topology == "mysql-gtid" {
		// A node can be retired before Orchestrator's next discovery. Confirm
		// presence through a successful search instead of treating HTTP 500 as
		// proof that it was forgotten.
		known := func() (bool, error) {
			raw, e := s.orchestrator(ctx, c, "search/"+url.PathEscape(m.Address))
			if e != nil {
				return false, e
			}
			var rows []struct {
				Key struct {
					Hostname string
					Port     int
				}
			}
			if e = json.Unmarshal(raw, &rows); e != nil {
				return false, e
			}
			for _, r := range rows {
				if r.Key.Hostname == m.Address && r.Key.Port == m.Port {
					return true, nil
				}
			}
			return false, nil
		}
		present, err := known()
		if err != nil {
			return err
		}
		if present {
			_, requestErr := s.orchestrator(ctx, c, "forget/"+instancePath(*m))
			present, err = known()
			if err != nil {
				return err
			}
			if present {
				if requestErr != nil {
					return requestErr
				}
				return errors.New("原生管理器仍登记目标节点")
			}
		}
	}
	e = s.DB.Transaction(func(tx *gorm.DB) error {
		if m.SourceID > 0 {
			if e := tx.Model(&cmdb.CmdbSQL{}).Where("id = ?", m.SourceID).Update("retired", true).Error; e != nil {
				return e
			}
		}
		if e := tx.Where("cluster_id = ? AND data_member_id = ? AND kind = ?", c.ID, m.ID, "patroni").Delete(&model.Member{}).Error; e != nil {
			return e
		}
		if e := tx.Delete(m).Error; e != nil {
			return e
		}
		return tx.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("revision", gorm.Expr("revision + 1")).Error
	})
	if e != nil {
		return e
	}
	fresh, e := s.Cluster(c.ID)
	if e != nil {
		return e
	}
	return s.syncMemberEndpoints(ctx, fresh, op)
}
func (s *Service) newMemberPlan(c *model.Cluster, p Parameters) ([]model.Member, error) {
	if len(p.Members) != 1 || p.Members[0].HostID == 0 || p.Members[0].SourceID != 0 {
		return nil, errors.New("每个任务新增一个独立受管实例，必须选择 CMDB 主机与空闲端口")
	}
	in := p.Members[0]
	r := ClusterRequest{Name: c.Name + "-node", Engine: c.Engine, Topology: "standalone", GroupID: c.GroupID, Provision: true, Members: []MemberInput{in}, Policy: JSON(map[string]any{})}
	prepared, e := s.prepare(r)
	if e != nil {
		return nil, e
	}
	m := prepared.Members[0]
	m.ClusterID = c.ID
	m.BaselinePending = c.Engine == "mysql"
	m.Role = "replica"
	if in.Role == "primary" && c.Topology == "redis-cluster" {
		m.Role = "primary"
	}
	m.Shard = in.Shard
	m.Name = fmt.Sprintf("data-%d-%d", in.HostID, in.Port)
	m.Service = fmt.Sprintf("autoops-db-%d-%d-%d", c.ID, in.HostID, in.Port)
	m.DataDir = fmt.Sprintf("/var/lib/autoops-database/%s/%d-%d", c.Name, in.HostID, in.Port)
	out := []model.Member{m}
	if c.Topology == "postgresql-patroni" {
		if m.Port+10000 > 65535 {
			return nil, errors.New("请为 Patroni 预留数据端口 +10000")
		}
		out = append(out, model.Member{ClusterID: c.ID, HostID: m.HostID, Name: m.Name + "-patroni", Kind: "patroni", Address: m.Address, Port: m.Port + 10000, DataDir: m.DataDir + "-patroni", Service: m.Service + "-rest", Managed: true, State: "unknown", Metrics: JSON(map[string]any{})})
	}
	return out, nil
}
func (s *Service) addMember(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	nodes, e := s.newMemberPlan(c, p)
	if e != nil {
		return e
	}
	parent, e := member(c, p.CandidateID)
	if e != nil || parent.Kind != "data" || parent.Role != "primary" {
		return errors.New("请选择有效的基线主节点")
	}
	if c.Topology == "redis-cluster" && nodes[0].Role == "replica" && nodes[0].HostID == parent.HostID {
		return errors.New("副本必须位于不同故障域")
	}
	if e = s.DB.Transaction(func(tx *gorm.DB) error {
		for i := range nodes {
			if nodes[i].Kind == "patroni" {
				nodes[i].DataMemberID = nodes[0].ID
			}
			if e := tx.Create(&nodes[i]).Error; e != nil {
				return e
			}
		}
		return tx.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("revision", gorm.Expr("revision + 1")).Error
	}); e != nil {
		return e
	}
	c.Members = append(c.Members, nodes...)
	if e = s.step(op, "部署独立成员与复制基线", func(log func(string)) error {
		_, e := s.ansible(ctx, c, nodes, "site.yml", map[string]any{"skip_replication": true}, log)
		return e
	}); e != nil {
		return e
	}
	m := nodes[0]
	if c.Engine == "mysql" {
		// Update the native recovery hook before making a new replica visible to
		// Orchestrator. Otherwise it could become an unfenceable future primary.
		if flag(object(c.Policy)["fencingInstalled"]) {
			if e = s.configureFencing(ctx, c, op); e != nil {
				return e
			}
		}
		if e = s.step(op, "MySQL 原生 CLONE 建立一致性副本", func(log func(string)) error {
			_, e := s.ansible(ctx, c, []model.Member{m}, "rebuild.yml", map[string]any{"donor": parent, "operation_id": op.ID}, log)
			return e
		}); e != nil {
			return e
		}
	}
	if c.Topology == "redis-cluster" {
		client, e := s.redis(c, m)
		if e != nil {
			return e
		}
		defer client.Close()
		if e = client.ClusterMeet(ctx, parent.Address, strconv.Itoa(parent.Port)).Err(); e != nil {
			return e
		}
		if e = waitUntil(ctx, 30*time.Second, func() bool {
			raw, e := client.ClusterNodes(ctx).Result()
			return e == nil && len(parseClusterNodes(raw)) >= len(c.Members)
		}); e != nil {
			return e
		}
		if m.Role == "replica" {
			if e = client.ClusterReplicate(ctx, parent.NativeID).Err(); e != nil {
				return e
			}
		}
	}
	if e = s.waitHealthy(ctx, c.ID, 5*time.Minute, 0); e != nil {
		return e
	}
	if c.Engine == "mysql" {
		s.DB.Model(&model.Member{}).Where("id = ?", m.ID).Update("baseline_pending", false)
	}
	c, e = s.Cluster(c.ID)
	if e != nil {
		return e
	}
	return s.syncMemberEndpoints(ctx, c, op)
}

// Slot summaries are computed from the native node report, not synthetic layout.
func SlotSummary(c *model.Cluster) map[string]any {
	owners := map[int]string{}
	migrating := []string{}
	conflicts := 0
	for _, m := range c.Members {
		if m.Kind != "data" || m.Role != "primary" {
			continue
		}
		var slots []string
		_ = json.Unmarshal(JSON(object(m.Metrics)["slots"]), &slots)
		for _, v := range slots {
			if strings.HasPrefix(v, "[") {
				migrating = append(migrating, v)
				continue
			}
			parts := strings.SplitN(v, "-", 2)
			lo, e := strconv.Atoi(parts[0])
			if e != nil {
				continue
			}
			hi := lo
			if len(parts) == 2 {
				hi, _ = strconv.Atoi(parts[1])
			}
			if lo < 0 || hi >= 16384 || hi < lo {
				continue
			}
			for slot := lo; slot <= hi; slot++ {
				if _, exists := owners[slot]; exists {
					conflicts++
				}
				owners[slot] = m.NativeID
			}
		}
	}
	return map[string]any{"total": 16384, "covered": len(owners), "missing": 16384 - len(owners), "conflicts": conflicts, "migrating": migrating}
}

func mustPrimary(c *model.Cluster) *model.Member { m, _ := primary(c); return m }

func (s *Service) syncMemberEndpoints(ctx context.Context, c *model.Cluster, op model.Operation) error {
	policy := object(c.Policy)
	if c.Engine == "mysql" && flag(policy["fencingInstalled"]) && policy["fencingMembersHash"] != fencingMembersHash(c) {
		if err := s.configureFencing(ctx, c, op); err != nil {
			return err
		}
	}
	if c.Engine != "redis" {
		proxies := 0
		managed := true
		for _, m := range c.Members {
			if m.Kind == "proxy" {
				proxies++
				managed = managed && m.Managed
			}
		}
		if proxies == 2 && managed {
			return s.configureEndpoints(ctx, c, op, Parameters{})
		}
	}
	return s.registerSources(c)
}
