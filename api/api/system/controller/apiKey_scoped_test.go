package controller

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	k8s "dodevops-api/api/k8s/model"
	"dodevops-api/api/system/model"
	"dodevops-api/middleware"
	"dodevops-api/pkg/apikey"
	"dodevops-api/pkg/authorization"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestScopedKeyAllocationAndEnforcement(t *testing.T) {
	db := permissionDB(t)
	gin.SetMode(gin.TestMode)
	adminJWT, _ := jwt.GenerateTokenByAdmin(model.SysAdmin{ID: 1, Username: "admin"})
	userJWT, _ := jwt.GenerateTokenByAdmin(model.SysAdmin{ID: 3, Username: "visitor"})
	if err := db.AutoMigrate(&cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&k8s.K8sPermission{}, &k8s.K8sUserGroup{}, &k8s.K8sUserGroupMember{}, &k8s.K8sGroupPermission{}, &k8s.K8sRbacRole{}, &k8s.K8sRbacBinding{}); err != nil {
		t.Fatal(err)
	}
	namespaceGrant := k8s.K8sPermission{UserID: 3, ClusterID: 7, Namespace: "team", PermissionType: "readonly"}
	if err := db.Create(&namespaceGrant).Error; err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE cmdb_host(id INTEGER, group_id INTEGER)")
	db.Exec("INSERT INTO cmdb_host VALUES(11,1),(12,1)")
	rule := cmdb.CmdbAssetPermission{Name: "one host", UserIDs: "[3]", AssetTypes: `["host"]`, HostIDs: "[11]", PermissionActions: `["get"]`, IsActive: 1}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	g := r.Group("/api/v1", middleware.AuthMiddleware())
	g.GET("/system/api-keys", ListAPIKeys)
	g.GET("/system/api-keys/catalog", APIKeyCatalog)
	g.POST("/system/api-keys", CreateAPIKey)
	g.DELETE("/system/api-keys/:id", RevokeAPIKey)
	g.GET("/system/permissions/me", MyPermissions)
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) }
	for _, path := range []string{"/apps", "/admin/list", "/dashboard/stats", "/monitor/host/:id", "/unknown"} {
		g.GET(path, ok)
	}
	k8sPermissions := middleware.NewK8sPermissionMiddleware()
	g.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/logs", k8sPermissions.CheckNamespacePermission(), ok)
	g.POST("/apps", ok)
	g.POST("/apps/deployment/execute", ok)
	g.PUT("/admin/updatePersonalPassword", ok)
	call := func(method, path, token, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var data struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || data.Code != want {
			t.Fatalf("%s %s: want %d, got %d %s", method, path, want, w.Code, w.Body.String())
		}
		return w
	}
	issue := func(user uint, scopes []string) (model.APIKey, string) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"name": "scoped fixture", "userId": user, "days": 30, "scopes": scopes})
		w := call("POST", "/system/api-keys", adminJWT, string(body), 200)
		var response struct {
			Data struct {
				Key   model.APIKey `json:"key"`
				Token string       `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		key, token := response.Data.Key, response.Data.Token
		if key.GrantMode != apikey.Scoped || key.IssuedBy != 1 || key.UserID != user || len(token) != 69 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("allocation mismatch: %+v", key)
		}
		if strings.Contains(w.Body.String(), `"hash"`) {
			t.Fatal("hash exposed")
		}
		return key, token
	}
	for _, scopes := range []string{`null`, `[]`, `["*"]`, `["user:inherit"]`, `["apps:read"]`, `["app:read","app:read"]`} {
		call("POST", "/system/api-keys", adminJWT, `{"userId":3,"name":"bad","days":30,"scopes":`+scopes+`}`, 400)
	}
	call("POST", "/system/api-keys", adminJWT, `{"userId":3,"name":"missing","days":30}`, 400)
	call("POST", "/system/api-keys", userJWT, `{"userId":3,"name":"bad","days":30,"scopes":["app:read"]}`, 403)
	call("GET", "/system/api-keys/catalog", userJWT, "", 403)
	catalog := call("GET", "/system/api-keys/catalog", adminJWT, "", 200)
	if !strings.Contains(catalog.Body.String(), `"docker"`) {
		t.Fatal("catalog missing")
	}
	key, token := issue(3, []string{"app:read", "monitor:read", "system:read", "k8s:read"})
	// No role grants are needed. The same user's browser login still has no module access.
	call("GET", "/apps", token, "", 200)
	call("GET", "/apps", userJWT, "", 403)
	call("POST", "/apps", token, `{}`, 403)
	call("POST", "/apps/deployment/execute", token, `{}`, 403)
	call("GET", "/dashboard/stats", token, "", 403)
	call("GET", "/admin/list", token, "", 403) // system:read cannot make the owner an administrator.
	call("GET", "/unknown", token, "", 403)
	call("PUT", "/admin/updatePersonalPassword", token, `{}`, 403)
	call("GET", "/k8s/cluster/7/namespaces/team/pods/web/logs", token, "", 200)
	call("GET", "/k8s/cluster/7/namespaces/other/pods/web/logs", token, "", 403)
	call("GET", "/k8s/cluster/8/namespaces/team/pods/web/logs", token, "", 403)
	db.Delete(&namespaceGrant)
	call("GET", "/k8s/cluster/7/namespaces/team/pods/web/logs", token, "", 403)
	call("GET", "/monitor/host/11", token, "", 200)
	call("GET", "/monitor/host/12", token, "", 403)
	db.Model(&rule).Update("is_active", 0)
	call("GET", "/monitor/host/11", token, "", 403)
	db.Model(&rule).Update("is_active", 1)
	identity := call("GET", "/system/permissions/me", token, "", 200).Body.String()
	if !strings.Contains(identity, `"app:read"`) || strings.Contains(identity, `"isAdmin":true`) {
		t.Fatal("incorrect effective identity", identity)
	}
	listed := call("GET", "/system/api-keys", token, "", 200).Body.String()
	if strings.Contains(listed, token) || strings.Contains(listed, `"hash"`) || !strings.Contains(listed, `"grantMode":"scoped"`) {
		t.Fatal("list leaked secret or lost scopes")
	}
	// Even an administrator-owned key has no implicit read/write/execute permission.
	_, adminKey := issue(1, []string{"app:read", "system:read", "system:write"})
	call("GET", "/apps", adminKey, "", 200)
	call("GET", "/admin/list", adminKey, "", 200)
	call("POST", "/apps", adminKey, `{}`, 403)
	call("GET", "/dashboard/stats", adminKey, "", 403)
	call("POST", "/system/api-keys", adminKey, `{"userId":1,"name":"escalate","days":30,"scopes":["app:execute"]}`, 403)
	// All selected permissions survive storage beyond the old varchar(200) limit.
	all := []string{}
	for _, m := range authorization.Catalog {
		for _, a := range m.Actions {
			all = append(all, m.Key+":"+a)
		}
	}
	allKey, _ := issue(1, all)
	if len(allKey.Scopes) <= 200 {
		t.Fatal("fixture must exceed old column limit")
	}
	var stored model.APIKey
	db.First(&stored, allKey.ID)
	if len(strings.Split(stored.Scopes, ",")) != len(all) {
		t.Fatal("permissions truncated")
	}
	db.Model(&key).Update("expires_at", time.Now().Add(-time.Minute))
	call("GET", "/apps", token, "", 401)
	db.Model(&key).Update("expires_at", time.Now().Add(time.Hour))
	db.Exec("UPDATE sys_admin SET status=0 WHERE id=3")
	call("GET", "/apps", token, "", 401)
	call("POST", "/system/api-keys", adminJWT, `{"userId":3,"name":"inactive","days":30,"scopes":["app:read"]}`, 400)
	db.Exec("UPDATE sys_admin SET status=1 WHERE id=3")
	call("DELETE", fmt.Sprintf("/system/api-keys/%d", key.ID), adminJWT, "", 200)
	call("GET", "/apps", token, "", 401)
	// New keys are not affected by subsequent role grant changes.
	independent, independentToken := issue(2, []string{"app:read"})
	db.Create(&model.RolePermission{RoleID: 2, Permissions: `[]`})
	call("GET", "/apps", independentToken, "", 200)
	db.Model(&model.RolePermission{}).Where("role_id = 2").Update("permissions", `["dashboard:read"]`)
	call("GET", "/apps", independentToken, "", 200)
	call("GET", "/dashboard/stats", independentToken, "", 403)
	// A malformed stored grant must fail closed, including an injected inheritance marker.
	db.Model(&independent).Update("scopes", apikey.Inherit)
	call("GET", "/apps", independentToken, "", 403)
	if _, err := apikey.EffectiveIdentity(apikey.WithKey(context.Background(), independent.ID), db, 3); err == nil {
		t.Fatal("cross-owner service context accepted")
	}
}
