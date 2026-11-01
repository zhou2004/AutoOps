package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"dodevops-api/api/database/model"
)

type nativeRecovery struct {
	ID                   uint `json:"Id"`
	UID                  string
	IsActive             bool
	Acknowledged         bool
	RecoveryEndTimestamp string
	AnalysisEntry        struct {
		AnalyzedInstanceKey struct {
			Hostname string
			Port     int
		}
	}
}

func (s *Service) recoveryAborted(ctx context.Context, c *model.Cluster, r nativeRecovery) bool {
	if r.UID == "" {
		return false
	}
	raw, e := s.orchestrator(ctx, c, "audit-recovery-steps/"+url.PathEscape(r.UID))
	if e != nil {
		return false
	}
	var steps []struct{ Message string }
	if json.Unmarshal(raw, &steps) != nil {
		return false
	}
	failed, stopped := false, false
	for _, step := range steps {
		if strings.Contains(step.Message, "Execution of PreFailoverProcesses hook") && strings.Contains(step.Message, " failed ") {
			failed = true
		}
		if step.Message == "Not running further PreFailoverProcesses hooks" {
			stopped = true
		}
		if strings.Contains(step.Message, "RecoverDeadMaster: will recover") {
			return false
		}
	}
	// v3.2.6 returns early on pre-hook failure without writing end_recovery.
	// Require the explicit native abort trace; an empty timestamp alone proves nothing.
	return failed && stopped
}

func (r nativeRecovery) belongsTo(c *model.Cluster) bool {
	for _, m := range c.Members {
		if m.Kind == "data" && m.Address == r.AnalysisEntry.AnalyzedInstanceKey.Hostname && m.Port == r.AnalysisEntry.AnalyzedInstanceKey.Port {
			return true
		}
	}
	return false
}

func (r nativeRecovery) finished() bool {
	// IsActive denotes the anti-flapping interval, not a running recovery.
	return r.RecoveryEndTimestamp != "" && !strings.HasPrefix(r.RecoveryEndTimestamp, "0001-")
}

func (s *Service) mysqlRecoveries(ctx context.Context, c *model.Cluster) ([]nativeRecovery, error) {
	raw, e := s.orchestrator(ctx, c, "audit-recovery")
	if e != nil {
		return nil, e
	}
	var rows []nativeRecovery
	if e = json.Unmarshal(raw, &rows); e != nil {
		return nil, e
	}
	return rows, nil
}

func (s *Service) waitMySQLRecovery(ctx context.Context, c *model.Cluster) error {
	return waitUntil(ctx, 120*time.Second, func() bool {
		rows, e := s.mysqlRecoveries(ctx, c)
		if e != nil {
			return false
		}
		for _, r := range rows {
			if r.belongsTo(c) && r.IsActive && !r.finished() && !s.recoveryAborted(ctx, c, r) {
				return false
			}
		}
		return true
	})
}

// This is a separate explicit operation. Resuming writes never silently removes
// native anti-flapping protection; only ended recoveries for this cluster are acked.
func (s *Service) acknowledgeMySQLRecovery(ctx context.Context, c *model.Cluster, op model.Operation) error {
	return s.step(op, "确认已结束的原生恢复并解除其防抖阻塞", func(log func(string)) error {
		if _, e := primary(c); e != nil {
			return e
		}
		if object(c.Health)["replication"] != "healthy" || object(c.Health)["database"] != "healthy" {
			return errors.New("须先核验唯一主节点及健康复制")
		}
		rows, e := s.mysqlRecoveries(ctx, c)
		if e != nil {
			return e
		}
		for _, r := range rows {
			if !r.belongsTo(c) || r.Acknowledged {
				continue
			}
			if !r.finished() && !s.recoveryAborted(ctx, c, r) {
				return errors.New("原生恢复尚未结束，不能确认")
			}
			if _, e = s.orchestrator(ctx, c, fmt.Sprintf("ack-recovery/%d?comment=%s", r.ID, url.QueryEscape(fmt.Sprintf("AutoOps operation %d: topology and replication verified", op.ID)))); e != nil {
				return e
			}
			log(fmt.Sprintf("原生恢复 #%d 已确认", r.ID))
		}
		return nil
	})
}
