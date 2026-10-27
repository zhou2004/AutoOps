package middleware

import (
	"dodevops-api/api/cmdb/model"
	"dodevops-api/pkg/authorization"
	database "dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestCrossModuleHostScopeWithoutCredentialGrants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hosts.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.Db
	database.Db = db
	defer func() { database.Db = previous; raw, _ := db.DB(); raw.Close() }()
	if err = db.AutoMigrate(&model.CmdbAssetPermission{}, &model.CmdbCredentialPermission{}, &model.CmdbUserGroup{}, &model.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE sys_admin(id INTEGER,status INTEGER)", "INSERT INTO sys_admin VALUES(1,1)", "CREATE TABLE sys_role(id INTEGER,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role(admin_id INTEGER,role_id INTEGER)", "CREATE TABLE cmdb_host(id INTEGER,ssh_key_id INTEGER, group_id INTEGER)", "INSERT INTO cmdb_host VALUES(1,7,1),(2,8,1)", "CREATE TABLE task_job(id INTEGER,host_ids TEXT)", "INSERT INTO task_job VALUES(1,'1'),(2,'1,2')"} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Create(&model.CmdbAssetPermission{Name: "host one", UserIDs: "[1]", AssetTypes: `["host"]`, HostIDs: "[1]", PermissionActions: `["get","connect_ssh"]`, IsActive: 1})
	for _, tc := range []struct {
		method, route, url, body string
		want                     int
	}{
		{"GET", "/api/v1/monitor/host/:id", "/api/v1/monitor/host/1", "", 200},
		{"GET", "/api/v1/monitor/host/:id", "/api/v1/monitor/host/2", "", 403},
		{"POST", "/api/v1/monitor/agent/deploy", "/api/v1/monitor/agent/deploy", `{"hostIds":[1]}`, 200},
		{"POST", "/api/v1/monitor/agent/deploy", "/api/v1/monitor/agent/deploy", `{"hostIds":[1,2]}`, 403},
		{"POST", "/api/v1/task/:id/run", "/api/v1/task/1/run", "", 200},
		{"POST", "/api/v1/task/:id/run", "/api/v1/task/2/run", "", 403},
	} {
		r := gin.New()
		r.Handle(tc.method, tc.route, func(c *gin.Context) {
			if checkHostTargets(c, authorization.Identity{UserID: 1}) {
				if tc.body != "" {
					data, _ := io.ReadAll(c.Request.Body)
					if string(data) != tc.body {
						t.Error("request body consumed")
					}
				}
				c.Status(200)
			}
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body)))
		if w.Code != tc.want {
			t.Errorf("%s %s: got %d %s", tc.url, tc.body, w.Code, w.Body.String())
		}
	}
	db.Model(&model.CmdbAssetPermission{}).Where("name = ?", "host one").Update("is_active", 0)
	r := gin.New()
	r.POST("/api/v1/monitor/agent/deploy", func(c *gin.Context) {
		if checkHostTargets(c, authorization.Identity{UserID: 1}) {
			t.Error("revoked asset permission allowed")
		}
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/monitor/agent/deploy", strings.NewReader(`{"hostIds":[1]}`)))
	if w.Code != 403 {
		t.Fatal(w.Body.String())
	}
}
