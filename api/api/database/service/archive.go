package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"dodevops-api/api/database/model"
)

func (s *Service) configureArchive(ctx context.Context, c *model.Cluster, op model.Operation) error {
	if c.Topology != "postgresql-patroni" {
		return errors.New("原生归档配置要求受管 Patroni 集群")
	}
	nodes := []model.Member{}
	for _, m := range c.Members {
		if m.Kind == "data" {
			if !m.Managed {
				return errors.New("观察接管节点不能自动覆盖归档配置")
			}
			nodes = append(nodes, m)
		}
	}
	if e := s.step(op, "配置 pgBackRest 仓库与 Patroni 本地归档参数", func(log func(string)) error { _, e := s.ansible(ctx, c, nodes, "archive.yml", nil, log); return e }); e != nil {
		return e
	}
	for _, m := range c.Members {
		if m.Kind == "patroni" {
			if _, e := s.control(ctx, c, m, http.MethodPost, "/reload", nil); e != nil {
				return e
			}
		}
	}
	if e := s.restart(ctx, c, op, Parameters{}); e != nil {
		return e
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	old, e := primary(fresh)
	if e != nil {
		return e
	}
	var candidate uint
	for _, m := range fresh.Members {
		if m.Kind == "data" && m.Role == "replica" && m.State == "healthy" {
			candidate = m.ID
			break
		}
	}
	if candidate == 0 {
		return errors.New("没有可用于归档参数滚动切换的健康副本")
	}
	switchOp := op
	switchOp.Action = "switchover"
	if e = s.switchover(ctx, fresh, switchOp, Parameters{CandidateID: candidate}); e != nil {
		return e
	}
	fresh, e = s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	if e = s.restart(ctx, fresh, op, Parameters{MemberID: old.ID}); e != nil {
		return e
	}
	if e = s.waitHealthy(ctx, c.ID, 120*time.Second, 0); e != nil {
		return e
	}
	for _, m := range fresh.Members {
		if m.Kind != "data" {
			continue
		}
		db, e := s.sql(ctx, fresh, m, "")
		if e != nil {
			return e
		}
		var mode string
		e = db.QueryRowContext(ctx, "SHOW archive_mode").Scan(&mode)
		db.Close()
		if e != nil || mode != "on" {
			return errors.New("归档参数尚未在所有数据节点生效")
		}
	}
	policy := object(c.Policy)
	policy["archiveConfigured"] = true
	policy["archiveStorage"] = "node-local pgBackRest repository; independent copy required for production"
	return s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("policy", JSON(policy)).Error
}
