package service

import (
	"context"
	"errors"
	"fmt"

	"dodevops-api/api/database/model"
)

// Resume writes only on the unique native replication root after verifying all
// replicas and GTID inclusion. A writable replica or active recovery aborts it.
func (s *Service) resumeMySQLWrites(ctx context.Context, c *model.Cluster, op model.Operation) error {
	root, e := primary(c)
	if e != nil {
		return e
	}
	return s.step(op, "核对原生根节点、GTID 与只读保护后恢复写入", func(log func(string)) error {
		raw, e := s.orchestrator(ctx, c, "instance/"+instancePath(*root))
		if e != nil {
			return e
		}
		v := object(raw)
		identity := object(JSON(v["Key"]))
		if text(identity["Hostname"]) != root.Address || int(number(identity["Port"])) != root.Port || !flag(v["IsLastCheckValid"]) {
			return errors.New("Orchestrator 尚未确认当前实例身份与健康状态")
		}
		upstream := object(JSON(v["MasterKey"]))
		if text(upstream["Hostname"]) != "" && number(upstream["Port"]) != 0 {
			return errors.New("Orchestrator 未确认此实例为复制根节点")
		}
		if e = s.waitMySQLRecovery(ctx, c); e != nil {
			return errors.New("原生恢复尚未结束")
		}

		db, e := s.sql(ctx, c, *root, "")
		if e != nil {
			return e
		}
		defer db.Close()
		var rootGTID string
		if e = db.QueryRowContext(ctx, "SELECT @@global.gtid_executed").Scan(&rootGTID); e != nil {
			return e
		}
		for _, m := range c.Members {
			if m.Kind != "data" || m.ID == root.ID {
				continue
			}
			metrics := object(m.Metrics)
			if m.Role != "replica" || m.State != "healthy" || !flag(metrics["read_only"]) || !flag(metrics["super_read_only"]) || metrics["replicationSQL"] != "Yes" || metrics["replicationIO"] != "Yes" {
				return errors.New("其他数据节点必须为健康且受只读保护的副本")
			}
			var subset int
			if e = db.QueryRowContext(ctx, "SELECT GTID_SUBSET(?,?)", text(metrics["gtid_executed"]), rootGTID).Scan(&subset); e != nil || subset != 1 {
				return errors.New("发现 GTID 分叉，不能恢复写入")
			}
		}
		if _, e = db.ExecContext(ctx, "SET PERSIST super_read_only=OFF"); e != nil {
			return e
		}
		if _, e = db.ExecContext(ctx, "SET PERSIST read_only=OFF"); e != nil {
			return e
		}
		var ro, sro int
		if e = db.QueryRowContext(ctx, "SELECT @@global.read_only,@@global.super_read_only").Scan(&ro, &sro); e != nil || ro != 0 || sro != 0 {
			return errors.New("写入状态未生效")
		}
		log(fmt.Sprintf("唯一复制根节点成员 #%d 已恢复写入，所有副本 GTID 与只读保护已核验", root.ID))
		return s.waitEndpoints(ctx, c)
	})
}
