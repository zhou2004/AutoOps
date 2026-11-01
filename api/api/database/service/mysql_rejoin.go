package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"time"
)

func mysqlRejoinDonor(c *model.Cluster, target uint) (*model.Member, error) {
	var root *model.Member
	for i := range c.Members {
		m := &c.Members[i]
		if m.ID == target || m.Kind != "data" || m.Role != "primary" || m.State != "healthy" {
			continue
		}
		if flag(object(m.Metrics)["read_only"]) || root != nil {
			return nil, errors.New("必须存在唯一可写且健康的其他主节点")
		}
		root = m
	}
	if root == nil {
		return nil, errors.New("没有可作为重加入来源的健康主节点")
	}
	return root, nil
}

func (s *Service) rejoinMySQL(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	target, e := member(c, p.MemberID)
	if e != nil {
		return e
	}
	donor, e := mysqlRejoinDonor(c, target.ID)
	if e != nil {
		return e
	}
	if !target.Managed {
		return errors.New("只允许对已确认服务归属的实例执行旧主重加入")
	}
	return s.step(op, "只读启动旧主、核验 GTID 并重新建立复制", func(log func(string)) error {
		// Prevent the native manager from treating this deliberate old-root stop
		// as another primary failure while its discovery state catches up.
		raw, err := s.orchestrator(ctx, c, "instance/"+instancePath(*target))
		if err != nil {
			return err
		}
		alreadyDowntimed := flag(object(raw)["IsDowntimed"])
		if !alreadyDowntimed {
			if _, e = s.orchestrator(ctx, c, "begin-downtime/"+instancePath(*target)+"/autoops/rejoin/1h"); e != nil {
				return e
			}
		}
		if _, e = s.ansible(ctx, c, []model.Member{*target}, "mysql-rejoin.yml", nil, log); e != nil {
			return e
		}
		db, e := s.sql(ctx, c, *target, "")
		if e != nil {
			return e
		}
		defer db.Close()
		source, e := s.sql(ctx, c, *donor, "")
		if e != nil {
			return e
		}
		defer source.Close()
		var oldGTID, currentGTID, purgedGTID string
		var ro, sro, sourceRO int
		if e = db.QueryRowContext(ctx, "SELECT @@global.gtid_executed,@@global.read_only,@@global.super_read_only").Scan(&oldGTID, &ro, &sro); e != nil {
			return e
		}
		if ro != 1 || sro != 1 {
			return errors.New("旧主未处于只读保护状态")
		}
		if e = source.QueryRowContext(ctx, "SELECT @@global.gtid_executed,@@global.gtid_purged,@@global.read_only").Scan(&currentGTID, &purgedGTID, &sourceRO); e != nil {
			return e
		}
		if sourceRO != 0 {
			return errors.New("新主已改变角色，停止重加入")
		}
		var subset, available int
		if e = db.QueryRowContext(ctx, "SELECT GTID_SUBSET(?,?),GTID_SUBSET(?,?)", oldGTID, currentGTID, purgedGTID, oldGTID).Scan(&subset, &available); e != nil {
			return e
		}
		if subset != 1 {
			return errors.New("检测到旧主 GTID 分叉，保留只读现场；必须选择备份重建，不得直接接回")
		}
		if available != 1 {
			return errors.New("新主已清理旧主所需 GTID，保留只读现场；必须从备份重建")
		}
		sec, e := s.secrets(c)
		if e != nil {
			return e
		}
		replUser := "autoops_replication"
		if c.DeploymentID != nil {
			replUser = "autoops_repl"
		}
		for _, q := range []string{"SET PERSIST read_only=ON", "SET PERSIST super_read_only=ON", "STOP SLAVE", "RESET SLAVE ALL"} {
			if _, e = db.ExecContext(ctx, q); e != nil {
				return e
			}
		}
		if _, e = db.ExecContext(ctx, "CHANGE MASTER TO MASTER_HOST=?, MASTER_PORT=?, MASTER_USER=?, MASTER_PASSWORD=?, MASTER_AUTO_POSITION=1", donor.Address, donor.Port, replUser, sec.Replication); e != nil {
			return errors.New("配置 GTID 复制失败，旧主保持只读")
		}
		if _, e = db.ExecContext(ctx, "START SLAVE"); e != nil {
			return e
		}
		if e = s.waitHealthy(ctx, c.ID, 120*time.Second, donor.ID); e != nil {
			return e
		}
		if _, e = s.ansible(ctx, c, []model.Member{*target}, "mysql-rejoin.yml", map[string]any{"rejoin_complete": true}, log); e != nil {
			return e
		}
		if _, e = s.orchestrator(ctx, c, "discover/"+instancePath(*target)); e != nil {
			return e
		}
		if !alreadyDowntimed {
			_, e = s.orchestrator(ctx, c, "end-downtime/"+instancePath(*target))
		}
		return e
	})
}
