package service

import (
	"context"
	model "dodevops-api/api/aiops/model"
	cmdb "dodevops-api/api/cmdb/model"
	"encoding/json"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func aiopsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "aiops.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
		"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY, status INTEGER, username TEXT)",
		"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)",
		"CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)",
		"INSERT INTO sys_admin VALUES (1,1,'admin'),(2,1,'operator'),(3,0,'disabled')",
		"INSERT INTO sys_role VALUES (1,'admin',1)",
		"INSERT INTO sys_admin_role VALUES (1,1)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&model.Config{}, &model.Run{}, &cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestConfigurationSecretLifecycleAndReferences(t *testing.T) {
	t.Setenv("AIOPS_KEY_FILE", filepath.Join(t.TempDir(), "key"))
	t.Setenv("AIOPS_ENCRYPTION_KEY", "")
	s := New(aiopsTestDB(t))
	ctx := context.Background()
	config := model.ConfigDTO{Config: model.Config{Kind: "model", Name: "local", Provider: "ollama", BaseURL: "http://localhost:11434", Model: "test", Enabled: true}, APIKey: "TOP_SECRET"}
	saved, err := s.Save(ctx, 0, config)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(saved)
	if strings.Contains(string(raw), "TOP_SECRET") || !saved.HasKey {
		t.Fatal("secret exposed or lost")
	}
	saved.Enabled = false
	updated, err := s.Save(ctx, saved.ID, saved)
	if err != nil || updated.Enabled || !updated.HasKey {
		t.Fatal("update must preserve key and false values")
	}
	updated.ClearKey = true
	cleared, err := s.Save(ctx, updated.ID, updated)
	if err != nil || cleared.HasKey {
		t.Fatal("clear key failed")
	}
	skill, err := s.Save(ctx, 0, model.ConfigDTO{Config: model.Config{Kind: "skill", Name: "inspect", Instructions: "Inspect inventory", Enabled: true}, Modules: []string{"cmdb"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(ctx, 0, model.ConfigDTO{Config: model.Config{Kind: "workflow", Name: "daily", Provider: "internal", Enabled: true}, Steps: []model.Step{{Name: "inspect", SkillID: skill.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Delete(ctx, skill.ID) == nil {
		t.Fatal("referenced skill deleted")
	}
	list, err := s.List(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if !c.Enabled || c.BaseURL != "" || c.Secret != "" {
			t.Fatal("nonadmin configuration leak")
		}
	}
}
func TestEvidenceScopesAndNoCredentials(t *testing.T) {
	db := aiopsTestDB(t)
	for _, sql := range []string{
		"INSERT INTO cmdb_host VALUES(11,'visible',1,1,1),(12,'hidden',3,1,1)",
		"INSERT INTO k8s_cluster VALUES(1,'visible',2,1),(2,'hidden',3,1)",
		"INSERT INTO monitor_incident VALUES(1,'TOP_SECRET','firing','critical','prometheus',CURRENT_TIMESTAMP)",
		"CREATE TABLE tool_service_deploy(id INTEGER,host_id INTEGER,status INTEGER,env_vars TEXT)",
		"INSERT INTO tool_service_deploy VALUES(1,11,1,'TOP_SECRET'),(2,12,3,'TOP_SECRET')",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&cmdb.CmdbAssetPermission{Name: "one", UserIDs: "[2]", AssetTypes: `["host"]`, HostIDs: "[11]", PermissionActions: `["list"]`, IsActive: 1}).Error; err != nil {
		t.Fatal(err)
	}
	s := New(db)
	evidence, err := s.Evidence(context.Background(), 2, []uint{1}, []string{"cmdb", "k8s", "monitor", "tools", "configcenter", "system"}, 7)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(evidence)
	text := string(raw)
	if strings.Contains(text, "TOP_SECRET") || strings.Contains(text, "hidden") {
		t.Fatal("private content leaked")
	}
	var data map[string]interface{}
	_ = json.Unmarshal(raw, &data)
	hosts := data["cmdb"].(map[string]interface{})["hosts"].(map[string]interface{})
	if hosts["total"] != float64(1) {
		t.Fatalf("host scope: %s", text)
	}
	clusters := data["k8s"].(map[string]interface{})["clusters"].(map[string]interface{})
	if clusters["total"] != float64(1) {
		t.Fatal("cluster scope")
	}
	if data["system"].(map[string]interface{})["error"] == nil || data["configcenter"].(map[string]interface{})["error"] == nil {
		t.Fatal("admin summaries exposed")
	}
	if _, err = s.Evidence(context.Background(), 3, nil, []string{"cmdb"}, 7); err == nil {
		t.Fatal("inactive account accepted")
	}
}
func TestAgentStreamsPersistsAndRestrictsHistory(t *testing.T) {
	db := aiopsTestDB(t)
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Messages []chatMessage `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Messages) != 2 || !strings.Contains(body.Messages[0].Content, "English") || !strings.Contains(body.Messages[1].Content, "Authorized evidence") {
			t.Error("missing agent context")
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Evidence-based diagnosis.\"},\"finish_reason\":null}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	config := model.Config{Kind: "model", Name: "test", Provider: "openai", BaseURL: provider.URL, Model: "test", Enabled: true, TimeoutSeconds: 30}
	db.Create(&config)
	workflow := model.Config{Kind: "workflow", Name: "two-step", Provider: "internal", Enabled: true, TimeoutSeconds: 30, Steps: `[{"name":"Inspect","instructions":"Find risks"},{"name":"Plan","instructions":"Prioritize"}]`}
	db.Create(&workflow)
	s := New(db)
	var events []model.Event
	err := s.Diagnose(context.Background(), 1, nil, model.DiagnoseRequest{ModelID: config.ID, WorkflowID: workflow.ID, Prompt: "Inspect", Modules: []string{"cmdb"}, Days: 7, Locale: "en-US"}, func(e model.Event) error { events = append(events, e); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || events[0].Type != "start" || events[len(events)-1].Type != "done" {
		t.Fatal("agent lifecycle")
	}
	id := events[0].RunID
	run, err := s.Run(context.Background(), 1, id)
	if err != nil || run.Status != "succeeded" || run.FinishedAt == nil || !strings.Contains(run.Output, "Evidence-based") || run.Evidence == "" {
		t.Fatalf("run not persisted: %v %+v", err, run)
	}
	if _, err = s.Run(context.Background(), 2, id); err == nil {
		t.Fatal("cross-user history leak")
	}
	rows, total, err := s.History(context.Background(), 2, 0)
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatal("history owner filter")
	}
	old := model.Run{Status: "running", CreatedAt: time.Now().Add(-12 * time.Minute)}
	markInterrupted(&old)
	if old.Status != "interrupted" {
		t.Fatal("stale running status")
	}
}
func TestAgentCancellationAndDisabledModel(t *testing.T) {
	db := aiopsTestDB(t)
	s := New(db)
	c := model.Config{Kind: "model", Name: "disabled", Provider: "openai", BaseURL: "http://localhost:1", Model: "test", Enabled: false}
	db.Create(&c)
	req := model.DiagnoseRequest{ModelID: c.ID, Prompt: "inspect", Modules: []string{"cmdb"}, Days: 7}
	if s.Diagnose(context.Background(), 1, nil, req, func(model.Event) error { return nil }) == nil {
		t.Fatal("disabled model accepted")
	}
	c.Enabled = true
	db.Save(&c)
	ctx, cancel := context.WithCancel(context.Background())
	var id uint
	err := s.Diagnose(ctx, 1, nil, req, func(e model.Event) error {
		if e.Type == "start" {
			id = e.RunID
			cancel()
			return context.Canceled
		}
		return nil
	})
	if err == nil {
		t.Fatal("cancellation lost")
	}
	run, _ := s.Run(context.Background(), 1, id)
	if run.Status != "cancelled" || run.FinishedAt == nil {
		t.Fatal("cancelled run not saved")
	}
	if activeRuns.users[1] {
		t.Fatal("execution slot leaked")
	}
}
