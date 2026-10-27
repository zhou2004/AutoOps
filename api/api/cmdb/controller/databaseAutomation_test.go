package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cmdbmodel "dodevops-api/api/cmdb/model"
	sysmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"dodevops-api/middleware"
	database "dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDatabaseDeploymentAdminBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	db.AutoMigrate(&cmdbmodel.DatabaseDeployment{}, &sysmodel.SysAdmin{})
	db.Exec("CREATE TABLE sys_role (id integer primary key, name text, role_key text, status integer)")
	db.Create(&sysmodel.SysAdmin{ID: 7, Username: "operator", Status: 1})
	db.Exec("CREATE TABLE sys_admin_role (admin_id integer, role_id integer)")
	// No administrator binding: even a valid logged-in user may not inspect
	// cluster-wide topology or mutate deployment state.
	previous := database.Db
	database.Db = db
	defer func() { database.Db = previous }()
	ctrl := NewDatabaseAutomationController(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(constant.ContextKeyUserObj, &sysmodel.JwtAdmin{ID: 7, Username: "operator"})
	})
	group := r.Group("/deployments", middleware.NewCmdbAssetPermissionMiddleware().AdminOnly())
	group.GET("", ctrl.List)
	group.POST("", ctrl.Create)
	group.GET("/:id/orchestrator-access", ctrl.Access)
	group.POST("/:id/ha", ctrl.HAAction)
	for _, item := range []struct{ method, path string }{{"GET", "/deployments"}, {"POST", "/deployments"}, {"GET", "/deployments/1/orchestrator-access"}, {"POST", "/deployments/1/ha"}} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(item.method, item.path, nil))
		var body struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body.Code != http.StatusForbidden {
			t.Fatalf("nonadmin reached %s %s", item.method, item.path)
		}
	}
}
