package controller

import (
	"dodevops-api/api/database/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func recordController(t *testing.T) (*Controller, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "records.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if e = db.AutoMigrate(model.Models()...); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY,status INTEGER)", "CREATE TABLE sys_role (id INTEGER PRIMARY KEY,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER,role_id INTEGER)", "INSERT INTO sys_admin VALUES (1,1)", "INSERT INTO sys_role VALUES (1,'admin',1)", "INSERT INTO sys_admin_role VALUES (1,1)"} {
		if e = db.Exec(q).Error; e != nil {
			t.Fatal(e)
		}
	}
	ctrl := New(db)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &system.JwtAdmin{ID: 1}); c.Next() })
	r.GET("/backups", ctrl.Records("backups"))
	r.GET("/repositories", ctrl.Repositories)
	return ctrl, r
}
func responseList(t *testing.T, r *gin.Engine, path string) ([]map[string]any, int) {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	var out struct {
		Code int
		Data struct {
			List  []map[string]any
			Total int
		}
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || out.Code != 200 {
		t.Fatalf("bad response: %s %v", w.Body.String(), e)
	}
	return out.Data.List, out.Data.Total
}
func TestBackupViewsFilterBeforePagination(t *testing.T) {
	ctrl, r := recordController(t)
	now := time.Now()
	ctrl.s.DB.Create(&model.Cluster{ID: 1, Name: "one", Engine: "mysql", GroupID: 10})
	ctrl.s.DB.Create(&model.Cluster{ID: 2, Name: "two", Engine: "redis", GroupID: 10})
	for _, b := range []model.Backup{{ID: 1, ClusterID: 1, Status: "complete", VerifiedAt: &now}, {ID: 2, ClusterID: 1, Status: "complete", VerifiedAt: &now}, {ID: 3, ClusterID: 1, Status: "failed", VerifiedAt: &now}, {ID: 4, ClusterID: 1, Status: "complete"}, {ID: 5, ClusterID: 2, Status: "complete", VerifiedAt: &now}, {ID: 6, ClusterID: 1, Status: "expired", VerifiedAt: &now}} {
		ctrl.s.DB.Create(&b)
	}
	rows, total := responseList(t, r, "/backups?view=verified&engine=mysql&size=1&page=2")
	if total != 2 || len(rows) != 1 || rows[0]["id"] != float64(1) {
		t.Fatalf("wrong page: %d %+v", total, rows)
	}
	rows, total = responseList(t, r, "/backups?view=verification&clusterId=1")
	if total != 1 || len(rows) != 1 || rows[0]["id"] != float64(4) {
		t.Fatalf("invalid pending list: %d %+v", total, rows)
	}
	_, total = responseList(t, r, "/backups?view=verified&clusterId=2")
	if total != 1 {
		t.Fatal("cluster filter ignored", total)
	}
	_, total = responseList(t, r, "/backups?clusterId=1")
	if total != 5 {
		t.Fatal("legacy API changed", total)
	}
}
func TestRepositoryLocationSearchBeforePagination(t *testing.T) {
	ctrl, r := recordController(t)
	for _, repo := range []model.Repository{{ID: 1, Name: "local", Kind: "local", Path: "/var/lib/autoops-backup-repository/one"}, {ID: 2, Name: "minio", Kind: "s3", Endpoint: "http://192.168.80.53:9000", Bucket: "database-backups"}, {ID: 3, Name: "other", Kind: "s3", Endpoint: "https://other", Bucket: "other"}} {
		ctrl.s.DB.Create(&repo)
	}
	for _, q := range []string{"192.168.80.53", "database-backups", "autoops-backup-repository"} {
		rows, total := responseList(t, r, "/repositories?location="+q+"&size=1")
		if total != 1 || len(rows) != 1 {
			t.Fatal("location filter failed", q, total)
		}
	}
}

func TestBackupNameSearchPreservesEngineScope(t *testing.T) {
	ctrl, r := recordController(t)
	now := time.Now()
	ctrl.s.DB.Create(&model.Cluster{ID: 1, Name: "mysql-business", Engine: "mysql"})
	ctrl.s.DB.Create(&model.Cluster{ID: 2, Name: "redis-business", Engine: "redis"})
	ctrl.s.DB.Create(&model.Backup{ClusterID: 1, Name: "before-release", Status: "complete", VerifiedAt: &now})
	ctrl.s.DB.Create(&model.Backup{ClusterID: 2, Name: "before-release", Status: "complete", VerifiedAt: &now})
	rows, total := responseList(t, r, "/backups?name=before-release&engine=mysql&view=verified")
	if total != 1 || len(rows) != 1 || rows[0]["clusterId"] != float64(1) {
		t.Fatal("backup name bypassed engine scope", rows, total)
	}
	_, total = responseList(t, r, "/backups?name=mysql-business&view=verified")
	if total != 1 {
		t.Fatal("legacy cluster-name search stopped working", total)
	}
}
