package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"dodevops-api/api/database/model"
)

func exerciseFencingFailure(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.ID != 4 || c.Name != "autoops-mysql-validation" {
		t.Fatal("fence fault fixture is restricted to the authorized lab")
	}
	old, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	runLiveOperation(t, ctx, s, c, user, "acknowledge-recovery", Parameters{})
	records, e := s.mysqlRecoveries(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	var before uint
	for _, r := range records {
		if r.ID > before {
			before = r.ID
		}
	}
	changed := []uint{}
	stopped := false
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		// Downtime suppresses another attempt while the original unique primary
		// is restored. Keep the independent hooks blocked until replicas are safe.
		for _, m := range c.Members {
			if m.Kind == "data" {
				if _, err := s.orchestrator(cleanup, c, "begin-downtime/"+instancePath(m)+"/autoops/fence-fixture-cleanup/1h"); err != nil {
					t.Error(err)
				}
			}
		}
		for _, m := range c.Members {
			if m.Kind == "data" && m.ID != old.ID {
				db, err := s.sql(cleanup, c, m, "")
				if err != nil {
					t.Error(err)
					continue
				}
				db.ExecContext(cleanup, "SET PERSIST super_read_only=ON")
				db.ExecContext(cleanup, "SET PERSIST read_only=ON")
				db.Close()
			}
		}
		if stopped {
			if _, err := s.ansible(cleanup, c, []model.Member{*old}, "service.yml", map[string]any{"service_action": "started"}, nil); err != nil {
				t.Error(err)
			}
		}
		for _, host := range changed {
			if _, err := s.remoteJSON(cleanup, host, `import os,json
os.chmod('/var/lib/autoops-fencing/4/fencing.json',0o640)
print(json.dumps({'restored':True}))`, map[string]any{}); err != nil {
				t.Error(err)
			}
		}
		if err := s.waitHealthy(cleanup, c.ID, 90*time.Second, old.ID); err != nil {
			t.Error(err)
			return
		}
		fresh, err := s.Refresh(cleanup, c.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if t.Failed() {
			return
		}
		runLiveOperation(t, cleanup, s, fresh, user, "acknowledge-recovery", Parameters{})
		runLiveOperation(t, cleanup, s, fresh, user, "exit-maintenance", Parameters{})
	}()
	for _, m := range c.Members {
		if m.Kind != "orchestrator" {
			continue
		}
		_, e = s.remoteJSON(ctx, m.HostID, `import os,json
p='/var/lib/autoops-fencing/4/fencing.json'
assert os.stat(p).st_uid==0
os.chmod(p,0)
print(json.dumps({'hookConfigBlocked':True}))`, map[string]any{})
		if e != nil {
			t.Fatal(e)
		}
		changed = append(changed, m.HostID)
	}
	if _, e = s.ansible(ctx, c, []model.Member{*old}, "service.yml", map[string]any{"service_action": "stopped"}, nil); e != nil {
		t.Fatal(e)
	}
	stopped = true
	var failedID uint
	e = waitUntil(ctx, 60*time.Second, func() bool {
		raw, err := s.orchestrator(ctx, c, "audit-recovery")
		if err != nil {
			return false
		}
		var rows []struct {
			nativeRecovery
			IsSuccessful bool
			AllErrors    []string
		}
		if json.Unmarshal(raw, &rows) != nil {
			return false
		}
		for _, r := range rows {
			if r.ID > before && r.belongsTo(c) && !r.IsSuccessful && s.recoveryAborted(ctx, c, r.nativeRecovery) {
				failedID = r.ID
				return true
			}
		}
		return false
	})
	if e != nil {
		t.Fatal("native pre-failover rejection not observed", e)
	}
	for _, m := range c.Members {
		if m.Kind != "data" || m.ID == old.ID {
			continue
		}
		db, e := s.sql(ctx, c, m, "")
		if e != nil {
			t.Fatal(e)
		}
		var ro, sro int
		e = db.QueryRowContext(ctx, "SELECT @@global.read_only,@@global.super_read_only").Scan(&ro, &sro)
		db.Close()
		if e != nil || ro != 1 || sro != 1 {
			t.Fatal("fencing failure allowed a writable replica")
		}
	}
	s.event(c.ID, user, 0, "integration-report", "info", string(JSON(map[string]any{"fencingFailureBlocked": true, "nativeRecoveryId": failedID, "autoOpsAPIOnline": false})))
	t.Logf("native recovery #%d rejected by independent pre-failover hook; no replica became writable while AutoOps API was offline", failedID)
}
