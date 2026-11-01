package service

import (
	"context"
	"database/sql"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"fmt"
	"testing"
	"time"
)

// Process-only faults on the explicitly authorized dedicated laboratory units.
// Host networking, unrelated etcd services and the AutoOps metadata DB are untouched.
func exerciseProtection(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	var source cmdb.CmdbSQL
	if s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source).Error != nil {
		t.Fatal("missing logical source")
	}
	connector := cmdbservice.NewDatabaseOperationService(s.DB)
	stopAndRestore := func(nodes []model.Member, check func()) {
		for _, m := range nodes {
			if !m.Managed || m.HostID != 530 && m.HostID != 528 && m.HostID != 531 {
				t.Fatal("fault target outside authorized lab")
			}
		}
		defer func() {
			_, e := s.ansible(ctx, c, nodes, "service.yml", map[string]any{"service_action": "started"}, nil)
			if e != nil {
				t.Errorf("restore stopped laboratory units: %v", e)
			}
		}()
		if _, e := s.ansible(ctx, c, nodes, "service.yml", map[string]any{"service_action": "stopped"}, nil); e != nil {
			t.Fatal(e)
		}
		check()
	}
	if c.Topology == "postgresql-patroni" {
		root, e := primary(c)
		if e != nil {
			t.Fatal(e)
		}
		db, e := s.sql(ctx, c, *root, "postgres")
		if e != nil {
			t.Fatal(e)
		}
		_, e = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS public.autoops_protection_probe(id text PRIMARY KEY)")
		db.Close()
		if e != nil {
			t.Fatal(e)
		}
		replicas := []model.Member{}
		voters := []model.Member{}
		for _, m := range c.Members {
			if m.Kind == "data" && m.Role == "replica" {
				replicas = append(replicas, m)
			}
			if m.Kind == "etcd" && len(voters) < 2 {
				voters = append(voters, m)
			}
		}
		if len(replicas) != 2 || len(voters) != 2 {
			t.Fatal("expected dedicated three-node topology")
		}
		writeResult := make(chan error, 1)
		var waiting *sql.DB
		stopAndRestore(replicas, func() {
			for _, m := range replicas {
				if node, err := s.sql(ctx, c, m, "postgres"); err == nil {
					node.Close()
					t.Fatalf("stopped replica %d is still reachable", m.ID)
				}
			}
			db, e := connector.OpenSQL(ctx, source.ID, "postgres")
			if e != nil {
				t.Fatal(e)
			}
			waiting = db
			var sync string
			if e = db.QueryRowContext(ctx, "SHOW synchronous_standby_names").Scan(&sync); e != nil || sync == "" {
				t.Fatal("synchronous protection disabled")
			}
			settings, err := queryMaps(ctx, db, "SELECT current_setting('synchronous_commit') AS synchronous_commit,current_setting('synchronous_standby_names') AS synchronous_standby_names,(SELECT count(*) FROM pg_stat_replication) AS active_replicas")
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("stopped replicas protection settings: %s", JSON(settings))
			go func() {
				_, err := db.ExecContext(ctx, "INSERT INTO public.autoops_protection_probe VALUES($1)", fmt.Sprintf("sync-%d", time.Now().UnixNano()))
				writeResult <- err
			}()
			// Do not cancel synchronous-commit waiting: PostgreSQL may finish the
			// local commit when the client explicitly cancels that wait. Observe a
			// pending request, then require acknowledgement after replicas return.
			select {
			case err := <-writeResult:
				t.Fatalf("write returned before a synchronous replica returned: %v", err)
			case <-time.After(3 * time.Second):
			}
		})
		select {
		case err := <-writeResult:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("blocked synchronous write did not finish after replicas returned")
		}
		waiting.Close()
		t.Log("strict synchronous protection: write stayed pending with both replicas stopped and completed only after replicas returned")
		if e = s.waitHealthy(ctx, c.ID, 120*time.Second, 0); e != nil {
			t.Fatal(e)
		}
		c, _ = s.Refresh(ctx, c.ID)
		stopAndRestore(voters, func() {
			e := waitUntil(ctx, 60*time.Second, func() bool {
				writable := 0
				for _, m := range c.Members {
					if m.Kind != "data" {
						continue
					}
					db, e := s.sql(ctx, c, m, "postgres")
					if e != nil {
						continue
					}
					var recovery bool
					e = db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&recovery)
					db.Close()
					if e == nil && !recovery {
						writable++
					}
				}
				return writable == 0
			})
			if e != nil {
				t.Fatal("leader remained writable after DCS majority loss", e)
			}
			t.Log("DCS majority loss: no writable PostgreSQL node after leader lease expired")
		})
		if e = s.waitHealthy(ctx, c.ID, 120*time.Second, 0); e != nil {
			t.Fatal(e)
		}
	}
	c, _ = s.Refresh(ctx, c.ID)
	for _, proxy := range c.Members {
		if proxy.Kind != "proxy" {
			continue
		}
		stopAndRestore([]model.Member{proxy}, func() {
			if e := waitUntil(ctx, 20*time.Second, func() bool { return s.verifyEndpoints(ctx, c) == nil }); e != nil {
				t.Fatal("SQL pair did not survive proxy process failure", e)
			}
			dbname := "postgres"
			q := "INSERT INTO public.autoops_protection_probe VALUES($1)"
			if c.Engine == "mysql" {
				dbname = "autoops_database_lab"
				q = "INSERT INTO autoops_ha_probe(run_key,payload) VALUES(?,'proxy')"
			}
			db, e := connector.OpenSQL(ctx, source.ID, dbname)
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			if _, e = db.ExecContext(ctx, q, fmt.Sprintf("proxy-%d", time.Now().UnixNano())); e != nil {
				t.Fatal("surviving SQL proxy did not accept business write", e)
			}
			t.Logf("proxy member %d stopped: remaining address accepted business write", proxy.ID)
		})
		break
	}
	s.event(c.ID, user, 0, "protection-report", "info", "专用服务进程故障测试完成；不包含硬件 watchdog、VIP 或宿主机故障")
}
