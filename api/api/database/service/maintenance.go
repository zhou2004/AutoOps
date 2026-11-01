package service

import (
	"bytes"
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"net/http"
	"time"
)

func (s *Service) maintenance(ctx context.Context, c *model.Cluster, op model.Operation) error {
	enter := op.Action == "enter-maintenance"
	if !enter {
		h := object(c.Health)
		if h["database"] != "healthy" || h["replication"] != "healthy" {
			return errors.New("退出维护前须确认复制拓扑健康")
		}
	}
	return s.step(op, "通过原生控制器调整维护状态", func(log func(string)) error {
		if c.Engine == "mysql" {
			for _, m := range c.Members {
				if m.Kind != "data" {
					continue
				}
				path := "end-downtime/" + instancePath(m)
				if enter {
					path = "begin-downtime/" + instancePath(m) + "/autoops/planned-maintenance/1h"
				}
				if _, e := s.orchestrator(ctx, c, path); e != nil {
					return e
				}
			}
		} else if c.Topology == "postgresql-patroni" {
			root, e := primary(c)
			if e != nil {
				return e
			}
			manager, e := patroniFor(c, *root)
			if e != nil {
				return e
			}
			if _, e = s.control(ctx, c, manager, http.MethodPatch, "/config", bytes.NewReader(JSON(map[string]any{"pause": enter}))); e != nil {
				return e
			}
		} else {
			return errors.New("该原生控制器不支持此维护方式")
		}
		policy := object(c.Policy)
		policy["maintenance"] = enter
		policy["maintenanceUpdatedAt"] = time.Now().UTC()
		if enter && c.Engine == "mysql" {
			policy["maintenanceExpiresAt"] = time.Now().UTC().Add(time.Hour)
		} else {
			delete(policy, "maintenanceExpiresAt")
		}
		return s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("policy", JSON(policy)).Error
	})
}
