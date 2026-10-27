package controller

import (
	"context"
	"dodevops-api/api/system/model"
	"dodevops-api/middleware"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	database "dodevops-api/pkg/db"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func permissionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.Db
	database.Db = db
	t.Cleanup(func() { database.Db = previous; raw, _ := db.DB(); raw.Close() })
	if err = db.AutoMigrate(&model.APIKey{}, &model.RolePermission{}, &model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		"CREATE TABLE sys_admin(id INTEGER PRIMARY KEY, username TEXT, nickname TEXT, status INTEGER)",
		"CREATE TABLE sys_role(id INTEGER PRIMARY KEY, role_key TEXT, role_name TEXT, status INTEGER)",
		"CREATE TABLE sys_admin_role(admin_id INTEGER, role_id INTEGER)",
		"INSERT INTO sys_admin VALUES(1,'admin','管理员',1),(2,'operator','运维',1),(3,'visitor','访客',1)",
		"INSERT INTO sys_role VALUES(1,'admin','管理员',1),(2,'operator','运维',1)",
		"INSERT INTO sys_admin_role VALUES(1,1),(2,2)",
	} {
		if err = db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestUserTokenPermissionsAndAdminAllocation(t *testing.T) {
	db := permissionDB(t)
	gin.SetMode(gin.TestMode)
	grants := []string{}
	for _, m := range authorization.Catalog {
		for _, a := range m.Actions {
			grants = append(grants, m.Key+":"+a)
		}
	}
	encoded, _ := json.Marshal(grants)
	db.Create(&model.RolePermission{RoleID: 2, Permissions: string(encoded)})
	adminJWT, _ := jwt.GenerateTokenByAdmin(model.SysAdmin{ID: 1, Username: "admin"})
	userJWT, _ := jwt.GenerateTokenByAdmin(model.SysAdmin{ID: 2, Username: "operator"})
	visitorJWT, _ := jwt.GenerateTokenByAdmin(model.SysAdmin{ID: 3, Username: "visitor"})
	r := gin.New()
	g := r.Group("/api/v1", middleware.AuthMiddleware())
	g.GET("/system/api-keys", ListAPIKeys)
	g.POST("/system/api-keys", CreateAPIKey)
	g.DELETE("/system/api-keys/:id", RevokeAPIKey)
	g.GET("/system/permissions/me", MyPermissions)
	g.PUT("/system/permissions/roles/:id", SaveRolePermissions)
	paths := []string{"/dashboard/stats", "/cmdb/hostlist", "/cmdb/docker/host/1/info", "/cmdb/databaselist", "/k8s/cluster", "/monitor/incident/list", "/task/list", "/apps", "/config/ecsauthlist", "/tool/list", "/aiops/catalog", "/sysOperationLog/list"}
	for _, p := range paths {
		g.GET(p, func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) })
	}
	g.POST("/apps/deployment/execute", func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) })
	g.GET("/admin/list", func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) })
	call := func(method, path, token, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var data struct {
			Code int `json:"code"`
		}
		json.Unmarshal(w.Body.Bytes(), &data)
		if data.Code != want {
			t.Fatalf("%s %s: want %d, got HTTP %d %s", method, path, want, w.Code, w.Body.String())
		}
		return w
	}
	call("POST", "/system/api-keys", userJWT, `{"userId":2,"name":"self","days":30}`, 403)
	call("POST", "/system/api-keys", visitorJWT, `{"userId":1,"name":"spoof","days":30}`, 403)
	// Previously issued inherited credentials keep their original role-based semantics.
	key, token, err := apikey.Create(context.Background(), db, 2, "Harness", []string{apikey.Inherit}, 30)
	if err != nil {
		t.Fatal(err)
	}
	db.Model(&key).Update("issued_by", 1)
	for _, p := range paths {
		call("GET", p, token, "", 200)
		call("GET", p, userJWT, "", 200)
		call("GET", p, visitorJWT, "", 403)
	}
	call("GET", "/admin/list", token, "", 403)
	call("POST", "/system/api-keys", token, `{"userId":1,"name":"escalate","days":30}`, 403)
	own := call("GET", "/system/api-keys", token, "", 200).Body.String()
	if !strings.Contains(own, "Harness") || strings.Contains(own, token) || strings.Contains(own, "hash") {
		t.Fatal("key list exposed a secret or lost owner data")
	}
	other := call("GET", "/system/api-keys?userId=2", visitorJWT, "", 200).Body.String()
	if strings.Contains(other, "Harness") {
		t.Fatal("owner filter bypass")
	}
	call("DELETE", fmt.Sprintf("/system/api-keys/%d", key.ID), userJWT, "", 403)
	// Empty grants revoke both credentials immediately, without logout or reissuing tokens.
	call("PUT", "/system/permissions/roles/2", userJWT, `{"permissions":[]}`, 403)
	call("PUT", "/system/permissions/roles/2", adminJWT, `{"permissions":[]}`, 200)
	call("GET", "/cmdb/hostlist", token, "", 403)
	call("GET", "/cmdb/hostlist", userJWT, "", 403)
	call("GET", "/system/api-keys", token, "", 200)
	call("PUT", "/system/permissions/roles/2", adminJWT, `{"permissions":["app:read"]}`, 200)
	call("GET", "/apps", token, "", 200)
	call("POST", "/apps/deployment/execute", token, `{}`, 403)
	db.Exec("UPDATE sys_role SET status=2 WHERE id=2")
	call("GET", "/apps", token, "", 403)
	db.Exec("UPDATE sys_role SET status=1 WHERE id=2")
	db.Exec("UPDATE sys_admin SET status=2 WHERE id=2")
	call("GET", "/system/api-keys", token, "", 401)
	call("GET", "/system/api-keys", userJWT, "", 401)
	db.Exec("UPDATE sys_admin SET status=1 WHERE id=2")
	call("DELETE", fmt.Sprintf("/system/api-keys/%d", key.ID), adminJWT, "", 200)
	call("GET", "/system/api-keys", token, "", 401)
	// Existing restricted keys never silently become full user tokens.
	legacy, limited, err := apikey.Create(context.Background(), db, 1, "legacy", []string{"aiops:read"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	call("GET", "/aiops/catalog", limited, "", 200)
	call("GET", "/admin/list", limited, "", 403)
	db.Model(&legacy).Update("expires_at", time.Now().Add(-time.Minute))
	call("GET", "/aiops/catalog", limited, "", 401)
}

func TestLoginPasswordBindsButAccountPasswordNeverSerializes(t *testing.T) {
	var dto model.LoginDto
	if err := json.Unmarshal([]byte(`{"password":"example"}`), &dto); err != nil || dto.Password != "example" {
		t.Fatal("login password binding broken")
	}
	data, _ := json.Marshal(model.SysAdmin{ID: 1, Password: "do-not-return-hashes"})
	if strings.Contains(string(data), "do-not-return") {
		t.Fatal("password leaked")
	}
}
