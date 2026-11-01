package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"net"
	"testing"
	"time"
)

func exerciseControlQuorum(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if conn, e := net.DialTimeout("tcp", "127.0.0.1:19880", time.Second); e == nil {
		conn.Close()
		t.Fatal("stop the dedicated test API before native offline acceptance")
	}
	kind := "sentinel"
	if c.Engine == "mysql" {
		kind = "orchestrator"
	}
	controls := []model.Member{}
	for _, m := range c.Members {
		if m.Kind == kind {
			controls = append(controls, m)
		}
	}
	if len(controls) != 3 {
		t.Fatal("requires three dedicated control voters")
	}
	stopped := []model.Member{}
	restore := func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		// Restore the old primary first while the majority remains absent. No
		// alternate writable node was observed during the fault window.
		for i := len(stopped) - 1; i >= 0; i-- {
			_, e := s.ansible(cleanup, c, []model.Member{stopped[i]}, "service.yml", map[string]any{"service_action": "started"}, nil)
			if e != nil {
				t.Error(e)
			}
		}
		stopped = nil
	}
	defer restore()
	stop := func(m model.Member) {
		t.Helper()
		if !m.Managed || (m.HostID != 530 && m.HostID != 528 && m.HostID != 531) {
			t.Fatal("outside authorized isolated services")
		}
		stopped = append(stopped, m)
		if _, e := s.ansible(ctx, c, []model.Member{m}, "service.yml", map[string]any{"service_action": "stopped"}, nil); e != nil {
			t.Fatal(e)
		}
	}
	if c.Engine == "mysql" {
		var leader *model.Member
		for i := range controls {
			if controls[i].Role == "leader" {
				leader = &controls[i]
			}
		}
		if leader == nil {
			t.Fatal("native Raft leader not found")
		}
		stop(*leader)
		if e := waitUntil(ctx, 30*time.Second, func() bool {
			fresh, e := s.Refresh(ctx, c.ID)
			if e != nil {
				return false
			}
			for _, m := range fresh.Members {
				if m.Kind == kind && m.ID != leader.ID && m.Role == "leader" && m.State == "healthy" {
					return s.verifyEndpoints(ctx, fresh) == nil
				}
			}
			return false
		}); e != nil {
			t.Fatal("Orchestrator leader did not change", e)
		}
		restore()
		time.Sleep(3 * time.Second)
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	root, e := primary(fresh)
	if e != nil {
		t.Fatal(e)
	}
	c = fresh
	stop(controls[0])
	stop(controls[1])
	stop(*root)
	timer := time.NewTimer(16 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-timer.C:
	}
	for _, m := range c.Members {
		if m.Kind != "data" || m.ID == root.ID {
			continue
		}
		if c.Engine == "mysql" {
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				t.Fatal(e)
			}
			var ro, sro int
			e = db.QueryRowContext(ctx, "SELECT @@read_only,@@super_read_only").Scan(&ro, &sro)
			db.Close()
			if e != nil || ro != 1 || sro != 1 {
				t.Fatal("promotion occurred without control majority")
			}
		} else {
			r, e := s.redis(c, m)
			if e != nil {
				t.Fatal(e)
			}
			info, e := r.Info(ctx, "replication").Result()
			r.Close()
			if e != nil || infoMap(info)["role"] != "slave" {
				t.Fatal("Sentinel promoted without majority")
			}
		}
	}
	t.Logf("%s majority loss plus primary stop: replicas did not promote; test API offline", kind)
	restore()
	if e = s.waitHealthy(ctx, c.ID, 90*time.Second, root.ID); e != nil {
		t.Fatal("control fixture did not recover", e)
	}
	fresh, e = s.Refresh(ctx, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	if c.Engine == "mysql" {
		root, e = primary(fresh)
		if e != nil {
			t.Fatal(e)
		}
		if flag(object(root.Metrics)["read_only"]) {
			runLiveOperation(t, ctx, s, fresh, user, "resume-writes", Parameters{})
		}
	}
	s.event(c.ID, user, 0, "integration-report", "info", fmt.Sprintf("%s 多数派丢失且主进程停止时未提升副本；原生保护在测试 API 不在线时成立，所有专用服务已恢复", kind))
}
