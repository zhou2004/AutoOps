package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCMDBRequestAssetIDUsesControllerSource(t *testing.T) {
	for _, tc := range []struct {
		name, method, url, body, param string
		want                           uint
		present                        bool
	}{
		{"body beats unrelated query", http.MethodPut, "/cmdb/hostupdate?id=1", `{"id":2}`, "id", 2, true},
		{"case folded last key", http.MethodPut, "/cmdb/hostupdate", `{"id":1,"ID":2}`, "id", 2, true},
		{"missing mutation id", http.MethodPut, "/cmdb/hostupdate?id=1", `{}`, "id", 0, false},
		{"file DELETE query", http.MethodDelete, "/cmdb/hostssh/file?hostId=9", "", "hostId", 9, true},
		{"GET detail", http.MethodGet, "/cmdb/hostinfo?id=8", "", "id", 8, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			id, present, err := requestAssetID(c, tc.param)
			if err != nil || id != tc.want || present != tc.present {
				t.Fatalf("id=%d present=%v err=%v", id, present, err)
			}
			body, _ := io.ReadAll(c.Request.Body)
			if string(body) != tc.body {
				t.Fatal("middleware consumed controller body")
			}
		})
	}
}

func TestCMDBMiddlewareRejectsCrossRuleAndMissingObject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "middleware.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err = db.AutoMigrate(&model.CmdbAssetPermission{}, &model.CmdbUserGroup{}, &model.CmdbUserGroupMember{}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"CREATE TABLE sys_admin (id INTEGER PRIMARY KEY,status INTEGER)", "INSERT INTO sys_admin VALUES (1,1)", "CREATE TABLE sys_role (id INTEGER PRIMARY KEY,role_key TEXT,status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER,role_id INTEGER)"} {
		if err = db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []model.CmdbAssetPermission{
		{Name: "view one", UserIDs: `[1]`, AssetTypes: `["host"]`, HostIDs: `[1]`, PermissionActions: `["list","get"]`, IsActive: 1},
		{Name: "operate two", UserIDs: `[1]`, AssetTypes: `["host"]`, HostIDs: `[2]`, PermissionActions: `["connect_ssh","file_delete","update"]`, IsActive: 1},
	} {
		if err = db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	m := &CmdbAssetPermissionMiddleware{permDao: dao.NewCmdbAssetPermissionDao(db)}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: 1}); c.Next() })
	ok := func(c *gin.Context) { c.JSON(200, gin.H{"code": 200}) }
	router.GET("/ssh/:id", m.RequireAssetAction("host", "connect_ssh"), ok)
	router.DELETE("/file", m.RequireAssetActionForID("host", "file_delete", "hostId"), ok)
	router.PUT("/update", m.RequireAssetAction("host", "update"), ok)
	router.GET("/host/:id", m.RequireCmdbPermission("host"), ok)
	router.GET("/list", m.RequireCmdbPermission("host"), ok)
	for _, tc := range []struct {
		method, url, body string
		want              int
	}{
		{"GET", "/ssh/1", "", 403}, {"GET", "/ssh/2", "", 200},
		{"DELETE", "/file?hostId=1", "", 403}, {"DELETE", "/file?hostId=2", "", 200}, {"DELETE", "/file", "", 400},
		{"PUT", "/update?id=2", `{"id":1}`, 403}, {"PUT", "/update", `{"id":2,"ID":1}`, 403}, {"PUT", "/update", `{}`, 400},
		{"GET", "/host/1", "", 200}, {"GET", "/host/2", "", 403},
		{"GET", "/list?id=2", "", 403},
	} {
		t.Run(tc.method+tc.url+tc.body, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, r)
			var response struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != tc.want {
				t.Fatalf("got %s want code %d", w.Body.String(), tc.want)
			}
		})
	}
}
