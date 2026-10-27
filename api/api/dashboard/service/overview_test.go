package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	model "dodevops-api/api/dashboard/model1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func dashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dashboard.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	for _, sql := range []string{
		"CREATE TABLE cmdb_host (id INTEGER PRIMARY KEY, name TEXT, status INTEGER, vendor INTEGER, group_id INTEGER)",
		"CREATE TABLE cmdb_sql (id INTEGER PRIMARY KEY, name TEXT, type INTEGER)",
		"CREATE TABLE cmdb_physical_machine (id INTEGER PRIMARY KEY, name TEXT, status INTEGER)",
		"CREATE TABLE cmdb_network_device (id INTEGER PRIMARY KEY, name TEXT, status INTEGER)",
		"CREATE TABLE k8s_cluster (id INTEGER PRIMARY KEY, name TEXT, status INTEGER, cluster_type INTEGER)",
		"CREATE TABLE cmdb_group (id INTEGER PRIMARY KEY, name TEXT, parent_id INTEGER)",
		"CREATE TABLE app_application (id INTEGER PRIMARY KEY, name TEXT, business_group_id INTEGER, status INTEGER)",
		"CREATE TABLE quick_deployments (id INTEGER PRIMARY KEY, title TEXT, status INTEGER, created_at DATETIME)",
		"CREATE TABLE task_execution_history (id INTEGER PRIMARY KEY, task_name TEXT, status INTEGER, created_at DATETIME)",
		"CREATE TABLE task_ansible_history (id INTEGER PRIMARY KEY, status INTEGER, created_at DATETIME)",
		"CREATE TABLE monitor_agent (id INTEGER PRIMARY KEY, host_id INTEGER, status INTEGER, last_heartbeat DATETIME)",
		"CREATE TABLE monitor_api_endpoint (id INTEGER PRIMARY KEY, name TEXT, status INTEGER, check_time TEXT)",
		"CREATE TABLE monitor_domain_cert (id INTEGER PRIMARY KEY, status INTEGER, check_time TEXT)",
		"CREATE TABLE monitor_incident (id INTEGER PRIMARY KEY, title TEXT, status TEXT, level TEXT, source TEXT, create_time DATETIME)",
		"CREATE TABLE monitor_alert_record (id INTEGER PRIMARY KEY, alert_status TEXT)",
		"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY, status INTEGER)",
		"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)",
		"CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)",
		"INSERT INTO sys_admin VALUES (1,1),(2,1),(3,0)",
		"INSERT INTO sys_role VALUES (1,'admin',1)",
		"INSERT INTO sys_admin_role VALUES (1,1)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}

func dashboardNow() time.Time {
	return time.Date(2026, 10, 20, 12, 0, 0, 0, dashboardLocation)
}

func TestOverviewEmptyAndUnavailableAreDifferent(t *testing.T) {
	db := dashboardTestDB(t)
	data := buildOverview(db, dashboardNow(), 7, nil, nil, true)
	if len(data.Errors) != 0 || data.Hosts.Total != 0 || data.Deployments.SuccessRate != nil || len(data.Deployments.Trend) != 7 {
		t.Fatalf("empty database should be valid empty data: %+v", data)
	}
	if data.Deployments.Trend[0].Date != "2026-10-14" || data.Deployments.Trend[6].Date != "2026-10-20" {
		t.Fatal("range must contain seven calendar days including today")
	}
	mustExec(t, db, "DROP TABLE task_execution_history")
	data = buildOverview(db, dashboardNow(), 7, nil, nil, true)
	if data.Jobs != nil || data.Errors["jobs"] == "" || data.Hosts == nil || data.Deployments == nil {
		t.Fatal("missing optional table should only make its own section unavailable")
	}
	encoded, err := json.Marshal(data)
	if err != nil || !json.Valid(encoded) {
		t.Fatal("response must be valid JSON")
	}
}

func TestExecutionRangeAndStatusSemantics(t *testing.T) {
	db := dashboardTestDB(t)
	now := dashboardNow()
	start := time.Date(2026, 10, 14, 0, 0, 0, 0, dashboardLocation)
	for _, status := range []int{1, 2, 3, 3, 4, 5, 6} {
		mustExec(t, db, "INSERT INTO quick_deployments(title,status,created_at) VALUES (?,?,?)", "release", status, start)
	}
	mustExec(t, db, "INSERT INTO quick_deployments(title,status,created_at) VALUES (?,?,?),(?,?,?),(?,?,?)",
		"before", 4, start.Add(-time.Second), "future", 4, now.Add(time.Second), "latest", 3, now)
	got, err := executionOverview(db.Table("quick_deployments"), "title", start, now, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.AllTimeTotal != 10 || got.Total != 8 || got.Success != 3 || got.Failed != 1 || got.Running != 1 || got.Pending != 1 || got.Other != 2 || *got.SuccessRate != 75 {
		t.Fatalf("incorrect execution semantics: %+v", got)
	}
	if len(got.Recent) != 8 || got.Recent[0].Name != "latest" || got.Trend[0].Total != 7 || got.Trend[1].Total != 0 || got.Trend[6].Success != 1 {
		t.Fatalf("incorrect range/trend/recent records: %+v", got)
	}
}

func TestOverviewInventoryScopesAndMonitoring(t *testing.T) {
	db := dashboardTestDB(t)
	now := dashboardNow()
	mustExec(t, db, "INSERT INTO cmdb_host VALUES (11,'one',1,2,1),(12,'two',1,2,1),(13,'hidden',3,3,1)")
	mustExec(t, db, "INSERT INTO cmdb_sql VALUES (1,'one',1),(2,'two',1),(3,'unknown',99)")
	mustExec(t, db, "INSERT INTO k8s_cluster VALUES (1,'visible',2,1),(2,'hidden',3,2)")
	mustExec(t, db, "INSERT INTO app_application VALUES (1,'one',1,2),(2,'two',1,1),(3,'orphan',99,2)")
	mustExec(t, db, "INSERT INTO cmdb_group VALUES (1,'业务一',0)")
	mustExec(t, db, "INSERT INTO monitor_agent VALUES (1,11,3,?),(2,11,3,?),(3,12,3,?),(4,13,3,?),(5,999,3,?)",
		now.Add(-time.Hour), now, now.Add(-time.Hour), now, now)
	mustExec(t, db, "INSERT INTO monitor_api_endpoint VALUES (1,'unchecked',1,''),(2,'checked',1,'2026-10-20 12:00:00'),(3,'failed',4,'2026-10-20 12:00:00')")
	scopes := map[string]map[uint]bool{"host": {11: true, 12: true}, "database": {}, "physical": {}, "network": {}}
	got := buildOverview(db, now, 7, scopes, []uint{1}, false)
	if len(got.Errors) != 0 {
		t.Fatalf("unexpected section errors: %v", got.Errors)
	}
	if got.Hosts.Total != 2 || len(got.Hosts.Distribution) != 1 || got.Hosts.Distribution[0].Count != 2 || got.Databases.Total != 0 || got.Clusters.Total != 1 {
		t.Fatalf("inventory must group by vendor/status rather than existing name, and apply scopes: %+v", got)
	}
	if got.Agents.Total != 2 || namedCount(got.Agents.Statuses, "心跳正常") != 1 || namedCount(got.Agents.Statuses, "心跳过期 / 未上报") != 1 {
		t.Fatalf("agents must deduplicate, scope and exclude deleted hosts: %+v", got.Agents)
	}
	if namedCount(got.Endpoints.Statuses, "未检查") != 1 || namedCount(got.Endpoints.Statuses, "正常") != 1 {
		t.Fatalf("never checked endpoints cannot be healthy: %+v", got.Endpoints)
	}
	if got.Applications.Total != 3 || namedCount(got.Applications.Distribution, "未分组 / 分组已删除") != 1 {
		t.Fatal("orphan applications must remain in totals and distribution")
	}
}

func namedCount(items []model.CountItem, name string) int64 {
	for _, item := range items {
		if item.Name == name {
			return item.Count
		}
	}
	return 0
}

func TestOverviewAuthorizationAndValidation(t *testing.T) {
	db := dashboardTestDB(t)
	mustExec(t, db, "INSERT INTO cmdb_host VALUES (11,'visible',1,1,1),(12,'hidden',1,1,1)")
	rule := cmdb.CmdbAssetPermission{Name: "test", AssetTypes: `["host"]`, PermissionActions: `["list"]`, HostIDs: `[11]`, UserIDs: `[2]`, IsActive: 1}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	s := &DashboardService{db: db}
	mustExec(t, db, "INSERT INTO k8s_cluster VALUES (1,'private',2,1)")
	denied, err := s.GetOverview(context.Background(), 2, nil, 7)
	if err != nil || denied.Clusters.Total != 0 {
		t.Fatal("missing cluster scope must not grant all clusters to a regular user")
	}
	data, err := s.GetOverview(context.Background(), 2, []uint{}, 7)
	if err != nil || data.Hosts.Total != 1 || data.AssetScope != "authorized" {
		t.Fatalf("regular user scope failed: %v %+v", err, data)
	}
	data, err = s.GetOverview(context.Background(), 1, nil, 7)
	if err != nil || data.Hosts.Total != 2 || data.AssetScope != "all" {
		t.Fatalf("administrator scope failed: %v %+v", err, data)
	}
	for _, user := range []uint{0, 3, 999} {
		if _, err = s.GetOverview(context.Background(), user, nil, 7); err == nil {
			t.Fatalf("inactive/missing user %d must fail", user)
		}
	}
	if _, err = s.GetOverview(context.Background(), 1, nil, 10000); err == nil {
		t.Fatal("unbounded range must fail")
	}
}

func TestIncidentScopeAndPriority(t *testing.T) {
	db := dashboardTestDB(t)
	now := dashboardNow()
	mustExec(t, db, "INSERT INTO monitor_incident VALUES (1,'old critical','firing','critical','prometheus',?),(2,'recent warning','firing','warning','api_endpoint',?),(3,'resolved','resolved','critical','domain_cert',?)", now.AddDate(0, 0, -20), now, now)
	data := buildOverview(db, now, 7, nil, nil, true)
	got := data.Incidents
	if got == nil || got.Firing != 2 || got.Critical != 1 || got.Warning != 1 || got.PeriodTotal != 2 || got.Recent[0].ID != 1 {
		t.Fatalf("current incidents must be independent of trend range, severe first: %+v", got)
	}
}

func TestLegacyIncidentsKeepTheirPriorityAndStatus(t *testing.T) {
	db := dashboardTestDB(t)
	mustExec(t, db, "DROP TABLE monitor_incident")
	mustExec(t, db, "CREATE TABLE monitor_incident (id INTEGER PRIMARY KEY, alert_desc TEXT, alert_level TEXT, status INTEGER, create_time DATETIME)")
	mustExec(t, db, "INSERT INTO monitor_incident VALUES (1,'completed','P1',3,?),(2,'open','P2',1,?),(3,'working','P4',2,?)", dashboardNow(), dashboardNow(), dashboardNow())
	got, err := incidentOverview(db, dashboardNow().AddDate(0, 0, -6), dashboardNow(), 7)
	if err != nil || got.Firing != 2 || got.PeriodTotal != 3 || got.Recent[0].ID != 2 || namedCount(got.Levels, "P2") != 1 {
		t.Fatalf("legacy schema must preserve priority and exclude completed incidents: %v %+v", err, got)
	}
}
