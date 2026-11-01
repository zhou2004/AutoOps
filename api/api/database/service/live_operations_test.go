package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func exerciseConfiguration(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	key := map[string]string{"mysql": "max_connect_errors", "postgresql": "log_min_duration_statement", "redis": "slowlog-max-len"}[c.Engine]
	report, e := s.Configurations(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	original := ""
	for _, row := range report["instances"].([]map[string]any) {
		v := text(row["values"].(map[string]any)[key])
		if v == "" || (original != "" && original != v) {
			t.Fatal("requires consistent original parameter")
		}
		original = v
	}
	n, e := strconv.Atoi(original)
	if e != nil {
		t.Fatal(e)
	}
	changed := strconv.Itoa(n + 1)
	runLiveOperation(t, ctx, s, c, user, "configure", Parameters{Changes: map[string]string{key: changed}})
	if e = s.waitConfiguration(ctx, c, map[string]string{key: changed}); e != nil {
		t.Fatal(e)
	}
	runLiveOperation(t, ctx, s, c, user, "configure", Parameters{Changes: map[string]string{key: original}})
	s.collectConfigurationStatus(ctx, c)
	fresh, e := s.Cluster(c.ID)
	if e != nil || fresh.ConfigurationStatus != "healthy" {
		t.Fatal("configuration drift after restoring original value", e)
	}
	s.collectNativeEvents(ctx, c)
	var before, after int64
	s.DB.Model(&model.Event{}).Where("cluster_id = ? AND external_key IS NOT NULL", c.ID).Count(&before)
	s.collectNativeEvents(ctx, c)
	s.DB.Model(&model.Event{}).Where("cluster_id = ? AND external_key IS NOT NULL", c.ID).Count(&after)
	if before != after {
		t.Fatal("native event duplicate or unexpected concurrent HA event")
	}
	t.Logf("%s parameter %s changed, verified, restored; native event dedup count=%d", c.Engine, key, after)
	if c.Topology == "postgresql-patroni" {
		runLiveOperation(t, ctx, s, c, user, "restart", Parameters{})
	}
	if c.Engine == "mysql" {
		runLiveOperation(t, ctx, s, c, user, "configure-semisync", Parameters{SemiSyncEnabled: true, SemiSyncTimeoutMS: 1000})
		root, e := primary(c)
		if e != nil {
			t.Fatal(e)
		}
		db, e := s.sql(ctx, c, *root, "autoops_database_lab")
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		if _, e = db.ExecContext(ctx, "INSERT INTO autoops_membership_probe VALUES (?)", fmt.Sprintf("semisync-%d", time.Now().UnixNano())); e != nil {
			t.Fatal(e)
		}
		if e = waitUntil(ctx, 20*time.Second, func() bool {
			var name, status string
			err := db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Rpl_semi_sync_master_status'").Scan(&name, &status)
			return err == nil && status == "ON"
		}); e != nil {
			t.Fatal("semisync acknowledgements not active", e)
		}
		t.Log("semi sync enabled on all members; primary acknowledges semi sync mode")
	}
}

func exerciseRetention(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.Topology != "redis-sentinel" {
		t.Fatal("retention fixture is only for isolated Sentinel")
	}
	var repo model.Repository
	if s.DB.Where("group_id = ? AND kind = ? AND status = ?", c.GroupID, "local", "healthy").First(&repo).Error != nil {
		t.Fatal("independent local fixture repository required")
	}
	op := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	var old model.Backup
	if s.DB.Where("operation_id = ?", op.ID).First(&old).Error != nil {
		t.Fatal("backup absent")
	}
	// Only this newly-created, explicitly isolated test backup receives an aged
	// fixture timestamp. Existing backup records and retention policies are untouched.
	s.DB.Model(&old).Update("created_at", time.Now().AddDate(0, 0, -s.retentionDays(c.ID)-1))
	newOp := runLiveOperation(t, ctx, s, c, user, "backup", Parameters{Method: "rdb", RepositoryID: repo.ID})
	var newest model.Backup
	s.DB.Where("operation_id = ?", newOp.ID).First(&newest)
	runLiveOperation(t, ctx, s, c, user, "verify-backup", Parameters{BackupID: newest.ID, HostID: 531, Port: 19390})
	if _, e := s.expirableBackup(c.ID, newest.ID); e == nil {
		t.Fatal("retention window did not protect new backup")
	}
	runLiveOperation(t, ctx, s, c, user, "expire-backup", Parameters{BackupID: old.ID})
	s.DB.First(&old, old.ID)
	if old.Status != "expired" {
		t.Fatal("retention did not expire the isolated fixture")
	}
	t.Logf("owned aged test backup #%d expired with independent repository; newer restored backup #%d retained", old.ID, newest.ID)
}

func exerciseDiscovery(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	var before, after int64
	s.DB.Model(&model.Cluster{}).Count(&before)
	found, e := s.Discover(ctx, ClusterRequest{Name: c.Name, Engine: c.Engine, Topology: c.Topology, GroupID: c.GroupID, Members: []MemberInput{{SourceID: root.SourceID}}}, user)
	if e != nil {
		t.Fatal(e)
	}
	s.DB.Model(&model.Cluster{}).Count(&after)
	if before != after {
		t.Fatal("discovery persisted an unexpected cluster")
	}
	expected := map[string]bool{}
	for _, m := range c.Members {
		if m.Kind == "data" {
			expected[fmt.Sprintf("%s:%d", m.Address, m.Port)] = true
		}
	}
	for _, m := range found.Members {
		if m.Kind == "data" && m.State == "healthy" {
			delete(expected, fmt.Sprintf("%s:%d", m.Address, m.Port))
		}
	}
	if len(expected) > 0 {
		t.Fatalf("read-only discovery missed registered data instances: %v", expected)
	}
	t.Logf("read-only seed discovery found all %s data members without persisting a cluster", c.Topology)
}

func exerciseRuntimeConfiguration(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	if c.Engine != "mysql" {
		t.Fatal("runtime fixture requires MySQL")
	}
	target, err := primary(c)
	if err != nil {
		t.Fatal(err)
	}
	original := map[uint]string{}
	for _, m := range c.Members {
		if m.Kind != "data" {
			continue
		}
		db, e := s.sql(ctx, c, m, "")
		if e != nil {
			t.Fatal(e)
		}
		var v string
		e = db.QueryRowContext(ctx, "SELECT @@global.wait_timeout").Scan(&v)
		db.Close()
		if e != nil {
			t.Fatal(e)
		}
		original[m.ID] = v
	}
	n, err := strconv.Atoi(original[target.ID])
	if err != nil {
		t.Fatal(err)
	}
	changed := strconv.Itoa(n + 1)
	op := runLiveOperation(t, ctx, s, c, user, "configure", Parameters{MemberID: target.ID, RuntimeOnly: true, Changes: map[string]string{"wait_timeout": changed}})
	var version model.ConfigRevision
	if s.DB.Where("operation_id = ?", op.ID).First(&version).Error != nil || version.MemberID != target.ID || !version.RuntimeOnly {
		t.Fatal("runtime scope/version missing")
	}
	for _, m := range c.Members {
		if m.Kind != "data" {
			continue
		}
		db, e := s.sql(ctx, c, m, "")
		if e != nil {
			t.Fatal(e)
		}
		var v string
		e = db.QueryRowContext(ctx, "SELECT @@global.wait_timeout").Scan(&v)
		db.Close()
		want := original[m.ID]
		if m.ID == target.ID {
			want = changed
		}
		if e != nil || v != want {
			t.Fatal("runtime update escaped member scope", m.ID, e)
		}
	}
	runLiveOperation(t, ctx, s, c, user, "configure", Parameters{MemberID: target.ID, RuntimeOnly: true, Changes: map[string]string{"wait_timeout": original[target.ID]}})
	s.collectConfigurationStatus(ctx, c)
	t.Logf("runtime setting applied only to member %d, verified, restored; version #%d retained", target.ID, version.ID)
}
