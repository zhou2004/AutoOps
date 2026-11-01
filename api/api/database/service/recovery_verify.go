package service

import (
	"context"
	"dodevops-api/api/database/model"
	"errors"
	"fmt"
	"time"
)

func (s *Service) confirmStopped(ctx context.Context, m model.Member) error {
	if !m.Managed || m.HostID == 0 || !unitPattern.MatchString(m.Service) {
		return errors.New("缺少可核验的独立隔离通道与服务归属")
	}
	raw, e := s.remoteJSON(ctx, m.HostID, `import json,sys,subprocess
p=json.load(sys.stdin);r=subprocess.run(['systemctl','show',p['service'],'-p','MainPID','-p','ActiveState'],capture_output=True,text=True,check=True)
s=dict(x.split('=',1) for x in r.stdout.splitlines() if '=' in x)
print(json.dumps({'stopped':s.get('MainPID')=='0' and s.get('ActiveState') in ('inactive','failed')}))`, m)
	if e != nil || !flag(object(raw)["stopped"]) {
		return errors.New("独立管理通道未确认旧主进程已经停止，不能强制恢复")
	}
	return nil
}
func (s *Service) waitRecovered(ctx context.Context, id, candidate uint) error {
	return waitUntil(ctx, 90*time.Second, func() bool {
		c, e := s.Refresh(ctx, id)
		if e != nil {
			return false
		}
		root, e := member(c, candidate)
		if e != nil || root.Role != "primary" || root.State != "healthy" {
			return false
		}
		replicas := 0
		for _, m := range c.Members {
			if m.Kind != "data" || m.ID == candidate {
				continue
			}
			if m.State != "healthy" {
				if s.confirmStopped(ctx, m) != nil {
					return false
				}
				continue
			}
			if m.Role != "replica" {
				return false
			}
			v := object(m.Metrics)
			if c.Engine == "postgresql" && v["replicationState"] == "streaming" && m.ParentID == fmt.Sprintf("%s:%d", root.Address, root.Port) {
				replicas++
			}
			if c.Engine == "mysql" && v["replicationIO"] == "Yes" && v["replicationSQL"] == "Yes" && m.ParentID == root.NativeID && flag(v["super_read_only"]) {
				replicas++
			}
		}
		return replicas >= 1 && s.verifyEndpoints(ctx, c) == nil
	})
}
