package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"time"
)

// Manual repair does not elect a winner or discard divergent transactions. The
// nominated existing root must contain every member's GTID set, with all writes
// fenced and replication paused before that comparison is made.
func (s *Service) repairMySQLReplication(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	candidate, e := member(c, p.CandidateID)
	if e != nil {
		return e
	}
	return s.step(op, "保护性停写、验证 GTID 超集并收敛复制拓扑", func(log func(string)) error {
		for _, m := range c.Members {
			if m.Kind != "data" {
				continue
			}
			if _, e = s.orchestrator(ctx, c, "begin-downtime/"+instancePath(m)+"/autoops/gtid-repair/1h"); e != nil {
				return e
			}
		}
		// Preserve downtime on any failure; the report explains the next action.
		for _, m := range c.Members {
			if m.Kind != "data" {
				continue
			}
			db, err := s.sql(ctx, c, m, "")
			if err != nil {
				return err
			}
			for _, q := range []string{"SET PERSIST super_read_only=ON", "SET PERSIST read_only=ON", "STOP SLAVE"} {
				_, err = db.ExecContext(ctx, q)
				if err != nil {
					db.Close()
					return err
				}
			}
			db.Close()
		}
		if e = s.waitMySQLRecovery(ctx, c); e != nil {
			return errors.New("原生恢复尚未结束，所有节点保持只读与维护状态")
		}

		db, e := s.sql(ctx, c, *candidate, "")
		if e != nil {
			return e
		}
		defer db.Close()
		var chosen, purged, mode string
		if e = db.QueryRowContext(ctx, "SELECT @@global.gtid_executed,@@global.gtid_purged,@@global.gtid_mode").Scan(&chosen, &purged, &mode); e != nil {
			return e
		}
		if mode != "ON" {
			return errors.New("候选未启用 GTID")
		}
		for _, m := range c.Members {
			if m.Kind != "data" || m.ID == candidate.ID {
				continue
			}
			node, err := s.sql(ctx, c, m, "")
			if err != nil {
				return err
			}
			var gtid string
			err = node.QueryRowContext(ctx, "SELECT @@global.gtid_executed").Scan(&gtid)
			node.Close()
			if err != nil {
				return err
			}
			var subset, available int
			if err = db.QueryRowContext(ctx, "SELECT GTID_SUBSET(?,?),GTID_SUBSET(?,?)", gtid, chosen, purged, gtid).Scan(&subset, &available); err != nil {
				return err
			}
			if subset != 1 || available != 1 {
				return errors.New("候选不包含所有 GTID，或缺少所需 binlog；保留全部只读现场，禁止覆盖分叉数据")
			}
		}
		sec, e := s.secrets(c)
		if e != nil {
			return e
		}
		repl := "autoops_replication"
		if c.DeploymentID != nil {
			repl = "autoops_repl"
		}
		if _, e = db.ExecContext(ctx, "RESET SLAVE ALL"); e != nil {
			return e
		}
		for _, m := range c.Members {
			if m.Kind != "data" || m.ID == candidate.ID {
				continue
			}
			node, err := s.sql(ctx, c, m, "")
			if err != nil {
				return err
			}
			if _, err = node.ExecContext(ctx, "RESET SLAVE ALL"); err == nil {
				_, err = node.ExecContext(ctx, "CHANGE MASTER TO MASTER_HOST=?,MASTER_PORT=?,MASTER_USER=?,MASTER_PASSWORD=?,MASTER_AUTO_POSITION=1", candidate.Address, candidate.Port, repl, sec.Replication)
			}
			if err == nil {
				_, err = node.ExecContext(ctx, "START SLAVE")
			}
			node.Close()
			if err != nil {
				return errors.New("复制配置未完成，全部节点仍受只读保护")
			}
		}
		if e = s.waitHealthy(ctx, c.ID, 120*time.Second, candidate.ID); e != nil {
			return e
		}
		fresh, e := s.Refresh(ctx, c.ID)
		if e != nil {
			return e
		}
		for _, m := range fresh.Members {
			if m.Kind == "data" {
				if _, e = s.orchestrator(ctx, fresh, "discover/"+instancePath(m)); e != nil {
					return e
				}
			}
		}
		log("GTID 已收敛，全部实例保持只读；恢复写入与结束维护需独立操作")
		return nil
	})
}
