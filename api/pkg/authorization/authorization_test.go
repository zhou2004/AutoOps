package authorization

import (
	"context"
	"dodevops-api/api/system/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
)

func TestLegacyRoleGrantAndExplicitEmpty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "roles.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.RolePermission{}, &model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, m := range []model.SysMenu{{ID: 1, Value: "cmdb:host:list", MenuType: 2, MenuStatus: 2}, {ID: 2, Value: "task:job:start", MenuType: 3, MenuStatus: 1}, {ID: 3, Value: "config:ecs:add", MenuType: 2, MenuStatus: 2}} {
		db.Create(&m)
		db.Create(&model.SysRoleMenu{RoleId: 7, MenuId: m.ID})
	}
	grants, legacy, err := ForRole(context.Background(), db, 7)
	if err != nil || !legacy || len(grants) != 1 || grants[0] != "cmdb:read" {
		t.Fatalf("legacy elevated: %v %v", grants, err)
	}
	db.Create(&model.RolePermission{RoleID: 7, Permissions: "[]"})
	grants, legacy, err = ForRole(context.Background(), db, 7)
	if err != nil || legacy || len(grants) != 0 {
		t.Fatalf("empty grant restored legacy permissions: %v", grants)
	}
}
func TestSensitiveRouteRequirements(t *testing.T) {
	for _, tc := range []struct {
		method, path, permission string
		admin                    bool
	}{
		{"POST", "/api/v1/system/api-keys", "system:write", true},
		{"PUT", "/api/v1/role/assignPermissions", "system:write", true},
		{"GET", "/api/v1/config/keymanage/list", "config:read", true},
		{"GET", "/api/v1/cmdb/hostssh/command/:id", "cmdb:execute", false},
		{"GET", "/api/v1/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/terminal", "k8s:execute", false},
		{"GET", "/api/v1/monitor/api-endpoint/check/:id", "monitor:execute", false},
		{"POST", "/api/v1/cmdb/docker/:assetType/:assetId/containers/:container/action", "docker:execute", false},
		{"GET", "/ws/task/ansible/:id/log/:work_id", "task:read", false},
	} {
		got := Route(tc.method, tc.path)
		if got.Permission != tc.permission || got.AdminOnly != tc.admin {
			t.Errorf("%s: %+v", tc.path, got)
		}
	}
	if Route("GET", "/api/v1/new-module/secrets").Permission != "" {
		t.Fatal("unknown module permitted")
	}
}
