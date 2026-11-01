package service

import (
	"context"
	"errors"
	"time"

	"dodevops-api/api/database/model"
)

// Ask a healthy Sentinel once. Selection and reconfiguration remain entirely
// native; a timeout is an unknown result and must never trigger another request.
func (s *Service) sentinelSwitchover(ctx context.Context, c *model.Cluster, op model.Operation) error {
	if op.Action != "switchover" {
		return errors.New("Sentinel 自主执行故障恢复")
	}
	old, e := primary(c)
	if e != nil {
		return e
	}
	name := text(object(c.Policy)["masterName"])
	if name == "" {
		name = c.Name
	}
	var selected *model.Member
	for i := range c.Members {
		m := &c.Members[i]
		if m.Kind == "sentinel" && m.State == "healthy" && flag(object(m.Metrics)["quorumHealthy"]) {
			selected = m
			break
		}
	}
	if selected == nil {
		return errors.New("没有通过 quorum 检查的 Sentinel")
	}
	if e = s.step(op, "请求 Sentinel 原生计划切换", func(log func(string)) error {
		client, e := s.redis(c, *selected)
		if e != nil {
			return e
		}
		defer client.Close()
		if e = client.Do(ctx, "SENTINEL", "CKQUORUM", name).Err(); e != nil {
			return errors.New("Sentinel 多数派不可用")
		}
		return client.Do(ctx, "SENTINEL", "FAILOVER", name).Err()
	}); e != nil {
		return e
	}
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	for {
		fresh, err := s.Refresh(ctx, c.ID)
		if err == nil {
			next, err := primary(fresh)
			h := object(fresh.Health)
			if err == nil && next.ID != old.ID && h["database"] == "healthy" && h["replication"] == "healthy" {
				return s.verifyEndpoints(ctx, fresh)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("Sentinel 已接受请求，但角色和复制未在期限内完成核验")
		case <-time.After(time.Second):
		}
	}
}
