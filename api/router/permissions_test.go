package router

import (
	"dodevops-api/common/config"
	"dodevops-api/pkg/authorization"
	database "dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Audit the actual Gin registry, including routes registered inside controllers.
func TestEveryProtectedRouteHasPermissionAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("server:\n  enableSwagger: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	previousConfig := config.Config
	if err := config.LoadConfig(cfg); err != nil {
		t.Fatal(err)
	}
	defer func() { config.Config = previousConfig }()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "routes.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	previous := database.Db
	database.Db = db
	defer func() { database.Db = previous; raw, _ := db.DB(); raw.Close() }()
	r := gin.New()
	register(r)
	registerWebSocketRoutes(r)
	public := map[string]bool{
		"GET /api/v1/captcha": true, "POST /api/v1/login": true, "POST /api/v1/monitor/agent/heartbeat": true,
		"POST /api/v1/monitor/alert/webhook/gitlab": true, "POST /api/v1/monitor/alert/webhook/zabbix": true, "POST /api/v1/monitor/alert/webhook/prometheus": true,
		"POST /api/v1/aiops/harness/tools": true, "GET /api/v1/cmdb/remote/connect": true,
	}
	total := 0
	for _, route := range r.Routes() {
		if public[route.Method+" "+route.Path] {
			continue
		}
		total++
		required := authorization.Route(route.Method, route.Path)
		if !required.Self && !authorization.Known(required.Permission) {
			t.Errorf("missing catalog mapping: %s %s -> %+v", route.Method, route.Path, required)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(route.Method, route.Path, nil))
		if w.Code != 401 {
			t.Errorf("unauthenticated route: %s %s -> %d", route.Method, route.Path, w.Code)
		}
	}
	if total < 300 {
		t.Fatalf("incomplete route registry: %d", total)
	}
	t.Logf("verified %d protected routes", total)
}
