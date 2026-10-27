package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common/util"
	database "dodevops-api/pkg/db"
	"encoding/json"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func alertGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "groups.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	old := database.Db
	database.Db = db
	t.Cleanup(func() { database.Db = old; raw, _ := db.DB(); raw.Close() })
	if err = db.AutoMigrate(&cmdb.CmdbGroup{}, &cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}, &model.MonitorAlertGroupRule{}, &model.MonitorAlertRule{}, &model.MonitorDataSource{}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"CREATE TABLE sys_admin(id INTEGER PRIMARY KEY,status INTEGER)", "INSERT INTO sys_admin VALUES(1,1),(2,1)",
		"CREATE TABLE sys_role(id INTEGER PRIMARY KEY,role_key TEXT,status INTEGER)", "INSERT INTO sys_role VALUES(1,'admin',1)",
		"CREATE TABLE sys_admin_role(admin_id INTEGER,role_id INTEGER)", "INSERT INTO sys_admin_role VALUES(1,1)",
	} {
		if err = db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Create(&cmdb.CmdbGroup{CreateTime: util.HTime{Time: time.Now()}, ID: 2, Name: "支付"})
	db.Create(&cmdb.CmdbGroup{CreateTime: util.HTime{Time: time.Now()}, ID: 3, Name: "生产", ParentID: 2})
	db.Create(&cmdb.CmdbGroup{CreateTime: util.HTime{Time: time.Now()}, ID: 4, Name: "其他"})
	db.Create(&model.MonitorDataSource{ID: 1, Name: "prometheus", Type: "Prometheus", DeployMethod: "Host", ApiUrl: "http://example.invalid", Config: "{}"})
	return db
}

func TestAlertGroupSelectionPreservesRulesAndLabels(t *testing.T) {
	db := alertGroupTestDB(t)
	s := &monitorAlertRuleService{groupRuleDao: dao.NewMonitorAlertGroupRuleDao(), ruleDao: dao.NewMonitorAlertRuleDao()}
	group := model.MonitorAlertGroupRule{GroupName: "支付告警", CmdbGroupID: 2, DataSourceID: 1, Labels: `{"owner":"ops"}`}
	if err := s.CreateGroup(&group); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(group.Labels, "支付") {
		t.Fatal("business group label not resolved")
	}
	enabled := 0
	rule := model.MonitorAlertRule{GroupID: group.ID, Alert: "CPU high", Expr: "up == 0", Style: "host", Status: "firing", Enabled: &enabled}
	db.Create(&rule)
	group.GroupName = "新名称"
	group.CmdbGroupID = 3
	group.Labels = ""
	group.RuleContent = "invalid YAML supplied by obsolete client"
	if err := s.UpdateGroup(&group); err != nil {
		t.Fatal(err)
	}
	var saved model.MonitorAlertRule
	if err := db.First(&saved, rule.ID).Error; err != nil {
		t.Fatal("metadata edit deleted child rule", err)
	}
	if saved.Status != "firing" || saved.Enabled == nil || *saved.Enabled != 0 || saved.Style != "host" {
		t.Fatal("rule state changed", saved)
	}
	var stored model.MonitorAlertGroupRule
	db.First(&stored, group.ID)
	labels := map[string]string{}
	json.Unmarshal([]byte(stored.Labels), &labels)
	if labels["owner"] != "ops" || labels["cluster"] != "生产" || stored.CmdbGroupID != 3 {
		t.Fatal("labels or relationship lost", stored)
	}
	for _, bad := range []model.MonitorAlertGroupRule{
		{GroupName: "no group", DataSourceID: 1}, {GroupName: "missing", CmdbGroupID: 999, DataSourceID: 1},
		{GroupName: "bad source", CmdbGroupID: 2, DataSourceID: 999}, {GroupName: "  ", CmdbGroupID: 2, DataSourceID: 1},
	} {
		if err := s.CreateGroup(&bad); err == nil {
			t.Fatal("invalid group accepted", bad)
		}
	}
	var total int64
	db.Model(&model.MonitorAlertGroupRule{}).Count(&total)
	if total != 1 {
		t.Fatal("invalid write persisted")
	}
	// Filtering happens before pagination and count, rather than hiding rows from a mixed page.
	other := model.MonitorAlertGroupRule{GroupName: "other", DataSourceID: 1, CmdbGroupID: 4}
	db.Create(&other)
	db.Create(&model.MonitorAlertRule{GroupID: other.ID, Alert: "other", Expr: "up == 0"})
	list, count, err := s.ruleDao.GetListByQuery(&model.MonitorAlertRuleQuery{Page: 1, PageSize: 1, DataSourceID: 1, CmdbGroupID: 3})
	if err != nil || count != 1 || len(list) != 1 || list[0].ID != rule.ID {
		t.Fatal("business filter or count incorrect", list, count, err)
	}
}

func TestAlertBusinessGroupOptionsRespectResourceGrants(t *testing.T) {
	db := alertGroupTestDB(t)
	options, err := AlertBusinessGroups(context.Background(), db, 2)
	if err != nil || len(options) != 0 {
		t.Fatal("ungranted groups exposed", options, err)
	}
	grant := cmdb.CmdbAssetPermission{Name: "business", UserIDs: "[2]", AssetTypes: `["group"]`, HostGroupIDs: "[2]", PermissionActions: `["get"]`, IsActive: 1}
	db.Create(&grant)
	options, err = AlertBusinessGroups(context.Background(), db, 2)
	if err != nil || len(options) != 2 || options[0].ID != 2 || options[1].ID != 3 {
		t.Fatal("group scope incorrect", options, err)
	}
	db.Model(&grant).Update("is_active", 0)
	options, err = AlertBusinessGroups(context.Background(), db, 2)
	if err != nil || len(options) != 0 {
		t.Fatal("revoked groups remain visible", options, err)
	}
	options, err = AlertBusinessGroups(context.Background(), db, 1)
	if err != nil || len(options) != 3 {
		t.Fatal("admin catalog incomplete", options, err)
	}
}
