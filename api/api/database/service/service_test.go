package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/apikey"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "platform.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	raw.SetMaxOpenConns(1)
	if e = db.AutoMigrate(model.Models()...); e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&cmdb.CmdbGroup{}, &cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}, &cmdb.DatabaseAutomationLock{}); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY,status INTEGER,username TEXT)", "CREATE TABLE sys_role (id INTEGER PRIMARY KEY,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER,role_id INTEGER)", "INSERT INTO sys_admin VALUES (1,1,'admin'),(2,1,'sql-user'),(3,1,'observer')", "INSERT INTO sys_role VALUES (1,'admin',1)", "INSERT INTO sys_admin_role VALUES (1,1)"} {
		if e = db.Exec(q).Error; e != nil {
			t.Fatal(e)
		}
	}
	db.Create(&cmdb.CmdbGroup{ID: 10, Name: "parent", CreateTime: util.HTime{Time: time.Now()}})
	db.Create(&cmdb.CmdbGroup{ID: 11, Name: "child", ParentID: 10, CreateTime: util.HTime{Time: time.Now()}})
	db.Create(&cmdb.CmdbGroup{ID: 12, Name: "other", CreateTime: util.HTime{Time: time.Now()}})
	for _, c := range []model.Cluster{{ID: 1, Name: "one", GroupID: 10}, {ID: 2, Name: "two", GroupID: 11}, {ID: 3, Name: "three", GroupID: 12}} {
		c.Engine = "mysql"
		c.Topology = "mysql-gtid"
		c.Management = "observe"
		c.Status = "unknown"
		c.Policy = JSON(map[string]any{})
		c.Health = JSON(map[string]any{})
		if e = db.Create(&c).Error; e != nil {
			t.Fatal(e)
		}
		if e = db.Create(&model.Member{ClusterID: c.ID, Kind: "data", Address: c.Name, Port: 3306, Version: "8.0.24", Role: "primary", State: "healthy"}).Error; e != nil {
			t.Fatal(e)
		}
	}
	return New(db)
}
func TestClusterScopesAreIndependentAndPaginatedAfterAuthorization(t *testing.T) {
	s := testService(t)
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "SQL only", UserIDs: `[2]`, AssetTypes: `["database"]`, HostGroupIDs: `[10]`, PermissionActions: `["db_query","db_execute"]`, IsActive: 1})
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "cluster view", UserIDs: `[3]`, AssetTypes: `["database_cluster"]`, HostGroupIDs: `[10]`, PermissionActions: `["db_cluster_view"]`, IsActive: 1})
	if e := s.Authorize(2, 1, "db_cluster_switch"); e != ErrForbidden {
		t.Fatalf("SQL permission escalated: %v", e)
	}
	if e := s.Authorize(3, 2, "db_cluster_view"); e != nil {
		t.Fatal("parent group must include child", e)
	}
	if e := s.Authorize(3, 3, "db_cluster_view"); e != ErrForbidden {
		t.Fatal("cross group access granted")
	}
	rows, total, e := s.List(3, Filter{Page: 1, Size: 1})
	if e != nil || total != 2 || len(rows) != 1 || rows[0].ID != 2 {
		t.Fatalf("incorrect authorized page: %v %d %v", rows, total, e)
	}
}

func TestBackupRecordUsesClusterScopeAndExcludesCredentials(t *testing.T) {
	s := testService(t)
	s.DB.Create(&cmdb.CmdbAssetPermission{Name: "cluster view", UserIDs: `[3]`, AssetTypes: `["database_cluster"]`, HostGroupIDs: `[10]`, PermissionActions: `["db_cluster_view"]`, IsActive: 1})
	b := model.Backup{ID: 777, ClusterID: 1, Status: "complete", Credentials: "private-fixture", Manifest: JSON(map[string]any{"restoreReport": map[string]any{"verified": true}})}
	if err := s.DB.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	got, err := s.BackupRecord(context.Background(), 3, b.ID)
	if err != nil || got.ID != b.ID {
		t.Fatal("authorized record outside the first page unavailable", err)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "private-fixture") || !strings.Contains(string(raw), "restoreReport") {
		t.Fatal("credential leaked or restore report omitted")
	}
	if err := s.DB.Model(&b).Update("cluster_id", 3).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.BackupRecord(context.Background(), 3, b.ID); err != ErrForbidden {
		t.Fatal("cross-group backup was accessible", err)
	}
	if _, err := s.BackupRecord(context.Background(), 3, 0); err == nil {
		t.Fatal("zero ID resolved a record")
	}
}
func TestPlansIdempotencyLocksAndRevision(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	plan, e := s.Plan(ctx, 1, 1, "adopt", Parameters{Management: "operate"})
	if e != nil || !plan.Executable {
		t.Fatal(e)
	}
	if len(plan.IdempotencyKey) < 8 || len(plan.IdempotencyKey) > 80 {
		t.Fatal("plan did not provide a valid idempotency key", plan.IdempotencyKey)
	}
	op, e := s.Submit(ctx, 1, 1, plan.ID, plan.IdempotencyKey, "one")
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Submit(ctx, 1, 1, plan.ID, plan.IdempotencyKey, "one")
	if e != nil || again.ID != op.ID {
		t.Fatal("duplicate submission created another job", e)
	}
	_, e = s.Submit(ctx, 1, 1, plan.ID, "concurrent-request", "one")
	if e != ErrConflict {
		t.Fatal("cluster mutation was not serialized", e)
	}
	claimed, e := s.claim("worker-a")
	if e != nil || claimed == nil {
		t.Fatal(e)
	}
	s.run(ctx, *claimed)
	var done model.Operation
	s.DB.First(&done, op.ID)
	if done.Status != "succeeded" {
		t.Fatal(done.Status, done.Message)
	}
	_, e = s.Submit(ctx, 1, 1, plan.ID, "stale-plan-request", "one")
	if e != ErrConflict {
		t.Fatal("outdated plan was accepted", e)
	}
}

func TestPlanKeysRemainBoundToTheirRequestAndRejectMissingKeys(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	p, e := s.Plan(ctx, 1, 1, "refresh", Parameters{})
	if e != nil {
		t.Fatal(e)
	}
	next, e := s.Plan(ctx, 1, 1, "refresh", Parameters{})
	if e != nil || p.IdempotencyKey == next.IdempotencyKey {
		t.Fatal("different plans need distinct keys", e)
	}
	for _, key := range []string{"", "short", strings.Repeat("x", 81)} {
		if _, e = s.SubmitConfirmed(ctx, 1, 1, p.ID, key, true); e == nil {
			t.Fatal("invalid key accepted", len(key))
		}
	}
	op, e := s.SubmitConfirmed(ctx, 1, 1, p.ID, p.IdempotencyKey, true)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.SubmitConfirmed(ctx, 1, 1, p.ID, p.IdempotencyKey, true)
	if e != nil || again.ID != op.ID {
		t.Fatal("a response-loss retry created a second task", e)
	}
	if _, e = s.SubmitConfirmed(ctx, 1, 1, next.ID, p.IdempotencyKey, true); e == nil {
		t.Fatal("another plan reused the old key")
	}
	var count int64
	s.DB.Model(&model.Operation{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate or invalid submissions created tasks", count)
	}
}
func TestUnknownExecutionRetainsLock(t *testing.T) {
	s := testService(t)
	old := time.Now().Add(-2 * time.Minute)
	op := model.Operation{ClusterID: 1, IdempotencyKey: "abandoned", Status: "running", WorkerID: "dead", HeartbeatAt: &old}
	s.DB.Create(&op)
	s.DB.Create(&model.OperationLock{ClusterID: 1, OperationID: op.ID})
	s.reconcileAbandoned()
	s.DB.First(&op, op.ID)
	if op.Status != "unknown" {
		t.Fatal(op.Status)
	}
	var count int64
	s.DB.Model(&model.OperationLock{}).Count(&count)
	if count != 1 {
		t.Fatal("abandoned operation lost lock")
	}
}
func TestStaleHealthAndCapacityGuard(t *testing.T) {
	old := time.Now().Add(-91 * time.Second)
	c := model.Cluster{CollectedAt: &old, Health: JSON(map[string]any{"database": "healthy", "replication": "healthy"})}
	MarkStale(&c)
	if c.Status != "stale" || object(c.Health)["database"] != "unknown" {
		t.Fatal("stale health remained green")
	}
	s := testService(t)
	if s.binlogPurgeSafe(context.Background(), &model.Cluster{ID: 1}, "mysql-bin.000001") {
		t.Fatal("purged logs without archive evidence")
	}
	if validateChanges("redis", map[string]string{"maxmemory-policy": "allkeys-lru"}) == nil {
		t.Fatal("eviction protection bypassed")
	}
	if validateChanges("postgresql", map[string]string{"synchronous_commit": "off"}) == nil {
		t.Fatal("synchronous protection bypassed")
	}
}
func TestSecretRedactionAcrossWrites(t *testing.T) {
	values := []string{}
	l := safeLog{secrets: []string{"long-secret-token"}, save: func(v string) { values = append(values, v) }}
	l.Write([]byte("before long-sec"))
	l.flush()
	l.Write([]byte("ret-token after"))
	l.flush()
	for _, v := range values {
		if strings.Contains(v, "long-sec") {
			t.Fatalf("secret fragment leaked: %q", v)
		}
	}
}
func TestNativeRedisSlotSummary(t *testing.T) {
	nodes := parseClusterNodes("a 10.0.0.1:7000@17000 myself,master - 0 0 1 connected 0-99 [100->-b]\nb 10.0.0.2:7000@17000 master - 0 0 2 connected 101-16383\n")
	if len(nodes) != 2 {
		t.Fatal(nodes)
	}
	c := model.Cluster{Members: []model.Member{{Kind: "data", Role: "primary", NativeID: "a", Metrics: JSON(map[string]any{"slots": nodes[0]["slots"]})}, {Kind: "data", Role: "primary", NativeID: "b", Metrics: JSON(map[string]any{"slots": nodes[1]["slots"]})}}}
	summary := SlotSummary(&c)
	if summary["covered"] != 16383 || summary["missing"] != 1 {
		t.Fatal(summary)
	}
	var raw map[string]any
	if json.Unmarshal(JSON(summary), &raw) != nil {
		t.Fatal("invalid slot response")
	}
}

func TestReconcileTransfersOwnershipAndRejectsLateWorker(t *testing.T) {
	s := testService(t)
	s.DB.Model(&model.Cluster{}).Where("id = 1").Update("management", "operate")
	old := model.Operation{ClusterID: 1, Action: "backup", Status: "unknown", WorkerID: "old-worker", IdempotencyKey: "old-backup", Parameters: JSON(Parameters{}), UserID: 1}
	if e := s.DB.Create(&old).Error; e != nil {
		t.Fatal(e)
	}
	s.DB.Create(&model.OperationLock{ClusterID: 1, OperationID: old.ID})
	p, e := s.Plan(context.Background(), 1, 1, "reconcile", Parameters{OperationID: old.ID})
	if e != nil || !p.Executable {
		t.Fatal(e, p)
	}
	next, e := s.Submit(context.Background(), 1, 1, p.ID, "new-reconcile", "one")
	if e != nil {
		t.Fatal(e)
	}
	called := false
	if s.step(old, "late mutation", func(func(string)) error { called = true; return nil }) == nil || called {
		t.Fatal("superseded worker executed a new step")
	}
	s.complete(old, "succeeded", "late result")
	var lock model.OperationLock
	s.DB.First(&lock, "cluster_id = 1")
	if lock.OperationID != next.ID {
		t.Fatal("late worker released or changed successor lock")
	}
	var previous model.Operation
	s.DB.First(&previous, old.ID)
	if previous.Status != "reconciling" {
		t.Fatal("late result replaced reconciliation state")
	}
}

func TestQueuedKeyIsRevalidatedAndSQLKeyCannotExecute(t *testing.T) {
	s := testService(t)
	if e := s.DB.Exec("UPDATE sys_admin SET username = username").Error; e != nil {
		t.Fatal(e)
	}
	if e := s.DB.AutoMigrate(&system.APIKey{}); e != nil {
		t.Fatal(e)
	}
	read, _, e := apikey.CreateScoped(context.Background(), s.DB, 1, "read", []string{"database:read"}, 1)
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.Plan(context.Background(), 1, 1, "adopt", Parameters{Management: "operate"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Submit(apikey.WithKey(context.Background(), read.ID), 1, 1, p.ID, "read-key-denied", "one"); e != ErrForbidden {
		t.Fatalf("read key executed mutation: %v", e)
	}
	key, _, e := apikey.CreateScoped(context.Background(), s.DB, 1, "execute", []string{"database:execute"}, 1)
	if e != nil {
		t.Fatal(e)
	}
	op, e := s.Submit(apikey.WithKey(context.Background(), key.ID), 1, 1, p.ID, "key-queued-task", "one")
	if e != nil {
		t.Fatal(e)
	}
	if op.APIKeyID != key.ID {
		t.Fatal("credential identity lost in queue")
	}
	s.DB.Model(&key).Update("revoked_at", time.Now())
	claimed, e := s.claim("worker")
	if e != nil || claimed == nil {
		t.Fatal(e)
	}
	s.run(context.Background(), *claimed)
	var done model.Operation
	s.DB.First(&done, op.ID)
	if done.Status != "failed" {
		t.Fatal("revoked key executed queued mutation", done.Status)
	}
	c, _ := s.Cluster(1)
	if c.Management != "observe" {
		t.Fatal("revoked key changed cluster")
	}
}

func TestExplicitConfirmationKeepsPlanAndLockGuards(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	plan, err := s.Plan(ctx, 1, 1, "adopt", Parameters{Management: "operate"})
	if err != nil || !plan.Executable {
		t.Fatal("plan", err)
	}
	if _, err = s.SubmitConfirmed(ctx, 1, 1, plan.ID, "confirmed-request", false); err == nil {
		t.Fatal("unconfirmed request accepted")
	}
	if _, err = s.SubmitConfirmed(ctx, 1, 2, plan.ID, "another-user-request", true); err == nil {
		t.Fatal("confirmation bypassed plan ownership")
	}
	op, err := s.SubmitConfirmed(ctx, 1, 1, plan.ID, "confirmed-request", true)
	if err != nil || op.Status != "queued" {
		t.Fatal("confirmation without name/reason rejected", err)
	}
	same, err := s.SubmitConfirmed(ctx, 1, 1, plan.ID, "confirmed-request", true)
	if err != nil || same.ID != op.ID {
		t.Fatal("confirmation lost idempotency", err)
	}
	if _, err = s.SubmitConfirmed(ctx, 1, 1, plan.ID, "conflicting-request", true); err != ErrConflict {
		t.Fatal("confirmation bypassed cluster lock", err)
	}
	plan2, err := s.Plan(ctx, 2, 1, "adopt", Parameters{Management: "operate"})
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Model(&model.Cluster{}).Where("id = ?", 2).UpdateColumn("revision", 100)
	if _, err = s.SubmitConfirmed(ctx, 2, 1, plan2.ID, "outdated-confirmation", true); err != ErrConflict {
		t.Fatal("confirmation bypassed revision check", err)
	}
}
