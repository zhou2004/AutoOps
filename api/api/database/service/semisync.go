package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"fmt"
	"time"
)

func (s *Service) configureSemiSync(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	if c.Topology != "mysql-gtid" || p.SemiSyncTimeoutMS < 100 || p.SemiSyncTimeoutMS > 60000 {
		return errors.New("半同步拓扑或超时参数无效")
	}
	for _, role := range []string{"replica", "primary"} {
		for _, m := range c.Members {
			if m.Kind != "data" || m.Role != role {
				continue
			}
			if e := s.step(op, "配置并核验半同步 "+m.Name, func(log func(string)) error {
				_, e := s.ansible(ctx, c, []model.Member{m}, "semisync.yml", map[string]any{"semisync_enabled": p.SemiSyncEnabled, "semisync_timeout_ms": p.SemiSyncTimeoutMS}, log)
				if e != nil {
					return e
				}
				if e = s.probeMember(ctx, c, &m); e != nil {
					return e
				}
				return nil
			}); e != nil {
				return e
			}
		}
	}
	// Replica I/O threads must negotiate again after the primary's plugin is
	// installed too; connecting before that point leaves them asynchronous.
	if e := s.step(op, "重新协商并核验半同步应答", func(log func(string)) error {
		for _, m := range c.Members {
			if m.Kind != "data" || m.Role != "replica" {
				continue
			}
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				return e
			}
			_, e = db.ExecContext(ctx, "STOP SLAVE IO_THREAD")
			if e == nil {
				_, e = db.ExecContext(ctx, "START SLAVE IO_THREAD")
			}
			db.Close()
			if e != nil {
				return e
			}
		}
		if e := s.waitHealthy(ctx, c.ID, 45*time.Second, 0); e != nil {
			return e
		}
		if p.SemiSyncEnabled {
			root, e := primary(c)
			if e != nil {
				return e
			}
			db, e := s.sql(ctx, c, *root, "")
			if e != nil {
				return e
			}
			defer db.Close()
			return waitUntil(ctx, 30*time.Second, func() bool {
				var key, clients string
				e := db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Rpl_semi_sync_master_clients'").Scan(&key, &clients)
				return e == nil && number(clients) >= 1
			})
		}
		return nil
	}); e != nil {
		return e
	}
	policy := object(c.Policy)
	policy["semiSyncEnabled"], policy["semiSyncTimeoutMS"] = p.SemiSyncEnabled, p.SemiSyncTimeoutMS
	if e := s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("policy", JSON(policy)).Error; e != nil {
		return e
	}
	s.event(c.ID, op.UserID, op.ID, "configuration", "info", fmt.Sprintf("半同步配置 enabled=%t timeout=%dms；超时可能退化为异步，不代表严格零数据丢失", p.SemiSyncEnabled, p.SemiSyncTimeoutMS))
	return nil
}
