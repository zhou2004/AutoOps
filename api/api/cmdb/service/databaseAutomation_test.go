package service

import (
	"context"
	"dodevops-api/common/util"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDatabaseDeployValidation(t *testing.T) {
	valid := DatabaseDeployRequest{Name: "cluster-test", Engine: "mysql", Mode: "replication", GroupID: 1, HostIDs: []uint{1, 2, 3}, ManagerIDs: []uint{1, 2, 3}, Password: "Database_Test123!"}
	if err := validateDatabaseDeploy(&valid); err != nil {
		t.Fatal(err)
	}
	cases := []func(*DatabaseDeployRequest){
		func(r *DatabaseDeployRequest) { r.Name = "../../etc" }, func(r *DatabaseDeployRequest) { r.Engine = "mongodb" },
		func(r *DatabaseDeployRequest) { r.HostIDs = []uint{1, 1} }, func(r *DatabaseDeployRequest) { r.ManagerIDs = []uint{1, 2} },
		func(r *DatabaseDeployRequest) { r.Port = 70000 }, func(r *DatabaseDeployRequest) { r.Password = "x';$(id) #longpassword" },
		func(r *DatabaseDeployRequest) { r.Mode = "standalone" }, func(r *DatabaseDeployRequest) { r.Engine = "redis" },
		func(r *DatabaseDeployRequest) { r.GroupID = 0 }, func(r *DatabaseDeployRequest) { r.ManagerIDs = nil; r.AutoRecovery = true },
	}
	for i, change := range cases {
		r := valid
		change(&r)
		if validateDatabaseDeploy(&r) == nil {
			t.Fatalf("unsafe case %d accepted", i)
		}
	}
}
func TestOrchestratorAuthenticationAndErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "autoops" || p != "secret" {
			t.Error("missing authentication")
		}
		if r.URL.Path == "/api/leader-check" {
			w.Write([]byte(`{"Code":"OK"}`))
			return
		}
		w.Write([]byte(`{"Code":"ERROR","Message":"internal secret"}`))
	}))
	defer server.Close()
	_, err := newOrchestratorClient([]string{server.URL}, "autoops", "secret").call(context.Background(), "recover/host/3306", true)
	if err == nil || strings.Contains(err.Error(), "internal secret") {
		t.Fatalf("error envelope mishandled: %v", err)
	}
}
func TestOrchestratorDoesNotRetryMutation(t *testing.T) {
	attempts := 0
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/leader-check" {
			w.Write([]byte(`{}`))
			return
		}
		attempts++
		http.Error(w, "failure", 503)
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { attempts++; w.Write([]byte(`{}`)) }))
	defer second.Close()
	_, err := newOrchestratorClient([]string{first.URL, second.URL}, "a", "b").call(context.Background(), "recover/host/3306", true)
	if err == nil || attempts != 1 {
		t.Fatalf("ambiguous mutation retried: attempts=%d err=%v", attempts, err)
	}
}
func TestOrchestratorRejectsRedirectAndOversize(t *testing.T) {
	leaked := false
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer dest.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.URL, 302) }))
	defer srv.Close()
	_, err := newOrchestratorClient([]string{srv.URL}, "a", "b").request(context.Background(), srv.URL, "health")
	if err == nil || leaked {
		t.Fatal("redirect followed")
	}
	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`"` + strings.Repeat("x", 4<<20) + `"`)) }))
	defer large.Close()
	if _, err := newOrchestratorClient(nil, "a", "b").request(context.Background(), large.URL, "health"); err == nil {
		t.Fatal("oversized response accepted")
	}
}
func TestDatabaseAutomationLockAndSecrets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.DatabaseAutomationRun{}, &model.DatabaseAutomationLock{}); err != nil {
		t.Fatal(err)
	}
	s := NewDatabaseAutomationService(db)
	first, err := s.acquire(1, 2, "backup")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.acquire(1, 3, "recover"); err == nil {
		t.Fatal("concurrent cluster mutation accepted")
	}
	var count int64
	db.Model(&model.DatabaseAutomationRun{}).Count(&count)
	if count != 1 {
		t.Fatal("failed acquisition left a phantom run")
	}
	s.finish(*first, "success", "done", "")
	if _, err = s.acquire(1, 3, "backup"); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(model.DatabaseDeployment{Secrets: "private-password"})
	if strings.Contains(string(encoded), "private-password") {
		t.Fatal("secret serialized")
	}
}
func TestHAOperationGuards(t *testing.T) {
	if _, err := haActionPath(DatabaseHARequest{Action: "graceful-master-takeover"}, "host/3306", "host/3306", 1); err == nil {
		t.Fatal("self promotion accepted")
	}
	if path, err := haActionPath(DatabaseHARequest{Action: "graceful-master-takeover"}, "old/3306", "new/3306", 1); err != nil || path != "graceful-master-takeover-auto/old/3306/new/3306" {
		t.Fatalf("planned takeover must restart replication on the demoted primary: %s, %v", path, err)
	}
	if _, err := haActionPath(DatabaseHARequest{Action: "arbitrary-command"}, "host/3306", "", 1); err == nil {
		t.Fatal("unknown command accepted")
	}
	if _, err := haActionPath(DatabaseHARequest{Action: "begin-maintenance"}, "host/3306", "", 1); err == nil {
		t.Fatal("empty maintenance reason accepted")
	}
	for _, host := range []string{"a/b", "host?query=x", "host\nInjected", "../health", "host%2fpath"} {
		if _, err := databaseInstancePath(host, 3306); err == nil {
			t.Fatalf("unsafe address accepted %q", host)
		}
	}
	s := NewDatabaseAutomationService(nil)
	if _, err := s.managedInstance(&model.DatabaseDeployment{Nodes: "[1,2]"}, 99); err == nil {
		t.Fatal("cross-cluster target accepted")
	}
}

func TestManagedDatabaseBindingCannotBeForgedOrDropped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.CmdbSQL{}, &model.CmdbGroup{}, &configmodel.AccountAuth{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CmdbGroup{ID: 1, Name: "test", CreateTime: util.HTime{Time: time.Now()}}).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&configmodel.AccountAuth{ID: 1, Type: 1, Name: "root", Host: "127.0.0.1", Port: 3306, Password: "cipher"})
	s := NewCmdbSQLService(dao.NewCmdbSQLDao(db))
	asset := model.CmdbSQL{Name: "test", Type: 1, AccountID: 1, GroupID: 1, DeploymentID: 99, HostID: 88}
	if err := s.CreateDatabase(&asset); err != nil {
		t.Fatal(err)
	}
	if asset.DeploymentID != 0 || asset.HostID != 0 {
		t.Fatal("forged managed binding accepted")
	}
	db.Model(&asset).Updates(map[string]any{"deployment_id": 7, "host_id": 8})
	asset.DeploymentID, asset.HostID = 0, 0
	if err := s.UpdateDatabase(&asset); err != nil {
		t.Fatal(err)
	}
	if asset.DeploymentID != 7 || asset.HostID != 8 {
		t.Fatal("metadata edit dropped managed binding")
	}
}

func TestGeneratedReplicationPasswordFitsMySQLLimit(t *testing.T) {
	first, err := randomDatabaseSecret()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomDatabaseSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 || first == second {
		t.Fatal("replication secret must contain 128 random bits encoded in at most 32 characters")
	}
}

func TestDatabaseLogRedactsSplitSecretBeforePersistence(t *testing.T) {
	var saved string
	log := &databaseRunLog{secrets: []string{"private-password"}, save: func(value string) { saved = value }}
	_, _ = log.Write([]byte("error private-pass"))
	if strings.Contains(saved, "private-pass") {
		t.Fatal("partial secret persisted")
	}
	_, _ = log.Write([]byte("word failed\n"))
	if strings.Contains(log.redacted(), "private-password") || !strings.Contains(log.redacted(), "[REDACTED]") {
		t.Fatal("split secret was not redacted")
	}
}
