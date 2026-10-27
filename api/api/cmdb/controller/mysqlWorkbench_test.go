package controller

import (
	cmdb "dodevops-api/api/cmdb/model"
	system "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"dodevops-api/common/util"
	database "dodevops-api/pkg/db"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMySQLWorkbenchEnforcesAssetReadWriteAndAdminBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&cmdb.CmdbSQL{}, &cmdb.CmdbAssetPermission{}, &cmdb.CmdbUserGroup{}, &cmdb.CmdbUserGroupMember{}, &system.SysAdmin{}); err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE sys_role(id integer primary key,role_key text,status integer)")
	db.Exec("CREATE TABLE sys_admin_role(admin_id integer,role_id integer)")
	if e := db.Create(&system.SysAdmin{ID: 7, Username: "reader", Status: 1, CreateTime: util.HTime{Time: time.Now()}}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&cmdb.CmdbSQL{ID: 1, Name: "granted", Type: 1}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&cmdb.CmdbSQL{ID: 2, Name: "hidden", Type: 1}).Error; e != nil {
		t.Fatal(e)
	}
	if e := db.Create(&cmdb.CmdbAssetPermission{Name: "read only", UserIDs: `[7]`, AssetTypes: `["database"]`, DatabaseIDs: `[1]`, PermissionActions: `["db_query"]`, IsActive: 1}).Error; e != nil {
		t.Fatal(e)
	}
	old := database.Db
	database.Db = db
	defer func() { database.Db = old }()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(constant.ContextKeyUserObj, &system.JwtAdmin{ID: 7, Username: "reader"})
		c.Next()
	})
	NewDatabaseOperationController(db).RegisterMySQL(router.Group("/cmdb"))
	for _, test := range []struct{ method, path string }{{"GET", "/2/mysql/table"}, {"GET", "/2/mysql/data"}, {"GET", "/2/mysql/export"}, {"POST", "/1/mysql/rows"}, {"POST", "/1/mysql/import"}, {"POST", "/1/mysql/schema"}, {"PUT", "/1/mysql/variables"}} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(test.method, "/cmdb/databases"+test.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		var body struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Code != 403 {
			t.Fatalf("unauthorized %s %s reached controller: %d %s", test.method, test.path, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/cmdb/databases/1/mysql/capabilities", nil))
	var result struct {
		Code int `json:"code"`
		Data struct {
			Write bool `json:"write"`
		}
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &result)
	if result.Code != 200 || result.Data.Write {
		t.Fatalf("read-only UI capabilities incorrect: %s", rec.Body.String())
	}
}
