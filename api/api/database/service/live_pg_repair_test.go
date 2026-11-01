package service

import (
	"bytes"
	"context"
	"dodevops-api/api/database/model"
	"net/http"
	"strings"
	"testing"
	"time"
)

func finishPGTestRepair(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.ID != 2 {
		t.Fatal("wrong repair fixture")
	}
	root, e := primary(c)
	if e != nil || root.ID != 8 {
		t.Fatal("confirmed branch must be current leader", e)
	}
	for _, m := range c.Members {
		if m.Kind == "data" {
			var logs strings.Builder
			_, e = s.ansible(ctx, c, []model.Member{m}, "patroni-update.yml", nil, func(v string) { logs.WriteString(v) })
			if e != nil {
				t.Log(logs.String())
				t.Fatal(e)
			}
		}
	}
	control, e := patroniFor(c, *root)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.control(ctx, c, control, http.MethodPatch, "/config", bytes.NewReader(JSON(map[string]any{"pause": true}))); e != nil {
		t.Fatal(e)
	}
	defer func() {
		sub, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, e = s.control(sub, c, control, http.MethodPatch, "/config", bytes.NewReader(JSON(map[string]any{"pause": false}))); e != nil {
			t.Error("resume native HA", e)
		}
	}()
	for _, id := range []uint{9, 8} {
		m, _ := member(c, id)
		if _, e = s.ansible(ctx, c, []model.Member{*m}, "service.yml", map[string]any{"service_action": "restarted"}, nil); e != nil {
			t.Fatal(e)
		}
		manager, e := patroniFor(c, *m)
		if e != nil {
			t.Fatal(e)
		}
		if e = waitUntil(ctx, 60*time.Second, func() bool {
			if err := s.probeMember(ctx, c, &manager); err != nil {
				return false
			}
			return manager.Version == "4.1.5"
		}); e != nil {
			t.Fatal("updated controller not ready", e)
		}
	}
	if _, e = s.control(ctx, c, control, http.MethodPatch, "/config", bytes.NewReader(JSON(map[string]any{"pause": false}))); e != nil {
		t.Fatal(e)
	}
	if e = waitUntil(ctx, 60*time.Second, func() bool {
		fresh, err := s.Refresh(ctx, 2)
		if err != nil {
			return false
		}
		leader, err := primary(fresh)
		return err == nil && leader.State == "healthy" && (leader.ID == 8 || leader.ID == 9)
	}); e != nil {
		t.Fatal(e)
	}
	fresh, e := s.Refresh(ctx, 2)
	if e != nil {
		t.Fatal(e)
	}
	runLiveOperation(t, ctx, s, fresh, user, "rebuild-replica", Parameters{MemberID: 7})
	if e = s.waitHealthy(ctx, 2, 90*time.Second, 0); e != nil {
		t.Fatal(e)
	}
	fresh, e = s.Refresh(ctx, 2)
	if e != nil {
		t.Fatal(e)
	}
	for _, m := range fresh.Members {
		if m.Kind == "data" {
			db, e := s.sql(ctx, fresh, m, "postgres")
			if e != nil {
				t.Fatal(e)
			}
			var n int
			e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.autoops_protection_probe WHERE id=$1", "partition-1793514563247651076").Scan(&n)
			db.Close()
			if e != nil || n != 1 {
				t.Fatal("confirmed record missing on repaired member", m.ID, e)
			}
		}
	}
	t.Log("all three data copies retain the acknowledged partition record; former branch preserved in quarantine; Patroni 4.1.5 selected")
}

// Repair only the specifically identified test fixture branch; never a general
// automatic recovery algorithm. Both surviving copies must contain the ACKed row.
func repairPGTestBranch(t *testing.T, ctx context.Context, s *Service, c *model.Cluster) {
	t.Helper()
	if c.ID != 2 || c.Name != "db-lab-postgresql-patroni" {
		t.Fatal("wrong repair fixture")
	}
	old, e := member(c, 7)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.remoteJSON(ctx, old.HostID, `import subprocess,json
s=subprocess.run(['systemctl','show','autoops-db-db-lab-postgresql-patroni-1','-p','MainPID','-p','ActiveState'],check=True,capture_output=True,text=True).stdout
assert 'MainPID=0' in s and 'ActiveState=inactive' in s
print(json.dumps({'fenced':True}))`, nil); e != nil {
		t.Fatal(e)
	}
	for _, id := range []uint{8, 9} {
		m, e := member(c, id)
		if e != nil {
			t.Fatal(e)
		}
		db, e := s.sql(ctx, c, *m, "postgres")
		if e != nil {
			t.Fatal(e)
		}
		var n int
		e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.autoops_protection_probe WHERE id=$1", "partition-1793514563247651076").Scan(&n)
		db.Close()
		if e != nil || n != 1 {
			t.Fatal("confirmed branch fixture absent", e)
		}
	}
	s.DB.Model(&model.Member{}).Where("id = ? AND cluster_id = ?", 7, 2).Update("role", "offline")
	candidate, _ := member(c, 8)
	control, e := patroniFor(c, *candidate)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.control(ctx, c, control, http.MethodPost, "/failover", strings.NewReader(`{"candidate":"data-2"}`)); e != nil {
		t.Fatal(e)
	}
	if e = waitUntil(ctx, 60*time.Second, func() bool {
		fresh, err := s.Refresh(ctx, 2)
		if err != nil {
			return false
		}
		for _, m := range fresh.Members {
			if m.ID == 8 {
				return m.Role == "primary" && m.State == "healthy"
			}
		}
		return false
	}); e != nil {
		t.Fatal(e)
	}
	s.event(2, 0, 0, "integration-repair", "warning", "旧测试主库已通过 systemd 明确停止，原数据目录保留；人工核对 .52/.53 两份含已确认写入的分支后，调用 Patroni 原生 failover 选择 data-2。旧主需隔离重建，不能直接启动。")
}
