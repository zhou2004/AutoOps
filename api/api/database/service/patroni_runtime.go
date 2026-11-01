package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"time"
)

func (s *Service) updatePatroni(ctx context.Context, c *model.Cluster, op model.Operation) error {
	if c.Topology != "postgresql-patroni" {
		return errors.New("需要 Patroni 集群")
	}
	root, e := primary(c)
	if e != nil {
		return e
	}
	for _, m := range c.Members {
		if m.Kind == "data" {
			if !m.Managed || m.State != "healthy" {
				return errors.New("更新前所有受管数据实例须健康")
			}
			if e = s.step(op, "安装固定控制组件 "+m.Name, func(log func(string)) error {
				_, err := s.ansible(ctx, c, []model.Member{m}, "patroni-update.yml", nil, log)
				return err
			}); e != nil {
				return e
			}
		}
	}
	restart := func(m model.Member) error {
		return s.step(op, "重启控制进程并核验 "+m.Name, func(log func(string)) error {
			if _, err := s.ansible(ctx, c, []model.Member{m}, "service.yml", map[string]any{"service_action": "restarted"}, log); err != nil {
				return err
			}
			return s.waitHealthy(ctx, c.ID, 90*time.Second, 0)
		})
	}
	var candidate uint
	for _, m := range c.Members {
		if m.Kind == "data" && m.ID != root.ID {
			if e = restart(m); e != nil {
				return e
			}
			candidate = m.ID
		}
	}
	if candidate == 0 {
		return errors.New("需要健康的已更新副本")
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	sw := op
	sw.Action = "switchover"
	if e = s.switchover(ctx, fresh, sw, Parameters{CandidateID: candidate}); e != nil {
		return e
	}
	if e = restart(*root); e != nil {
		return e
	}
	fresh, e = s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	for _, m := range fresh.Members {
		if m.Kind == "patroni" && m.Version != "4.1.5" {
			return errors.New("控制组件版本尚未一致")
		}
	}
	return nil
}
