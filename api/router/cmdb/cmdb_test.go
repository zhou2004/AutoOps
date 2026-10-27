package cmdb

import (
	"path/filepath"
	"strings"
	"testing"

	database "dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCmdbRouteRegistrationHasNoConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "routes.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	previous := database.Db
	database.Db = db
	defer func() { database.Db = previous; _ = sqlDB.Close() }()
	router := gin.New()
	RegisterCmdbRoutes(router.Group("/api/v1"))
	RegisterCmdbRemoteRoutes(router.Group("/api/v1"))
	routes := map[string]bool{}
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/cmdb/permission/credential") {
			t.Errorf("retired credential authorization route is still registered: %s", route.Path)
		}
		routes[route.Method+" "+route.Path] = true
	}
	for _, expected := range []string{
		"GET /api/v1/cmdb/hostlist",
		"POST /api/v1/cmdb/docker/:assetType/:assetId/containers/:container/transfer",
		"GET /api/v1/cmdb/network/:id",
		"GET /api/v1/cmdb/network/:id/config/snapshots",
		"POST /api/v1/cmdb/network/:id/config/restore",
		"GET /api/v1/cmdb/assets/:assetType/:assetId/connections",
		"POST /api/v1/cmdb/assets/:assetType/:assetId/connections/:connectionId/ticket",
		"GET /api/v1/cmdb/remote/connect",
	} {
		if !routes[expected] {
			t.Errorf("missing registered route %s", expected)
		}
	}
}
