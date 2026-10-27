package db

import (
	"path/filepath"
	"testing"

	"dodevops-api/api/system/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNavigationMigrationPreservesAssignmentsAndIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "menus.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err := db.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	menus := []model.SysMenu{
		{ID: 1, MenuName: "资产管理", MenuType: 1, MenuStatus: 2},
		{ID: 2, MenuName: "容器管理", MenuType: 1, MenuStatus: 2},
		{ID: 3, MenuName: "系统管理", MenuType: 1, MenuStatus: 2},
		{ID: 4, ParentId: 1, MenuName: "Docker管理", Url: "/cmdb/docker", Value: "docker:read", MenuType: 2, MenuStatus: 2},
		{ID: 5, ParentId: 1, MenuName: "资产授权", Url: "cmdb/asset-permission", MenuType: 2, MenuStatus: 2},
		{ID: 6, ParentId: 3, MenuName: "容器授权", Url: "/system/k8s-permission", MenuType: 2, MenuStatus: 2},
		{ID: 8, ParentId: 3, MenuName: "角色权限", Url: "system/role-permission", MenuType: 2, MenuStatus: 2},
		{ID: 9, ParentId: 3, MenuName: "API Token管理", Url: "/system/api-keys", MenuType: 2, MenuStatus: 2},
		{ID: 10, ParentId: 1, MenuName: "数据管理", Url: "cmdb/db", Value: "database:read", MenuType: 2, MenuStatus: 2},
		{ID: 7, ParentId: 1, MenuName: "凭据授权", Url: "cmdb/credential-permission", MenuType: 2, MenuStatus: 2},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatal(err)
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 12, MenuId: 10}, {RoleId: 10, MenuId: 4}, {RoleId: 11, MenuId: 6}} {
		if err := db.Create(&link).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := migrateManagementNavigation(db); err != nil {
			t.Fatal(err)
		}
	}
	var docker, auth, base model.SysMenu
	db.First(&docker, 4)
	db.First(&auth, 5)
	db.First(&base, 3)
	if docker.ParentId != 2 || docker.Url != "k8s/docker" || docker.Value != "docker:read" {
		t.Fatalf("Docker permission or relocation lost: %+v", docker)
	}
	if auth.ParentId != 3 || auth.Url != "system/authorization" || base.MenuName != "基础管理" {
		t.Fatal("authorization not consolidated")
	}
	var key model.SysMenu
	db.First(&key, 9)
	if key.Url != "system/api-keys" || key.MenuName != "API Key" || key.ParentId != 3 {
		t.Fatalf("key menu not migrated: %+v", key)
	}
	var source, parent model.SysMenu
	db.First(&source, 10)
	db.First(&parent, source.ParentId)
	if source.Url != "database/sources" || source.MenuName != "数据源管理" || source.Value != "database:read" || parent.MenuName != "数据库管理" {
		t.Fatal("database navigation or permission lost")
	}
	var links int64
	db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 12, parent.ID).Count(&links)
	if links != 1 {
		t.Fatal("new parent role assignment missing")
	}
	var activeDuplicates int64
	db.Model(&model.SysMenu{}).Where("id IN ? AND menu_status = 2", []uint{6, 7, 8}).Count(&activeDuplicates)
	if activeDuplicates != 0 {
		t.Fatal("obsolete menus remain active")
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 10, MenuId: 4}, {RoleId: 10, MenuId: 2}, {RoleId: 11, MenuId: 5}, {RoleId: 11, MenuId: 3}, {RoleId: 11, MenuId: 6}} {
		var count int64
		db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", link.RoleId, link.MenuId).Count(&count)
		if count != 1 {
			t.Fatalf("role assignment missing or duplicated: %+v: %d", link, count)
		}
	}
}

func TestNavigationMigrationCreatesMissingEntries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fresh.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err := migrateManagementNavigation(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SysMenu{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.SysMenu{ID: 1, MenuName: "基础管理", MenuType: 1})
	db.Create(&model.SysMenu{ID: 2, MenuName: "容器管理", MenuType: 1})
	for i := 0; i < 2; i++ {
		if err := migrateManagementNavigation(db); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&model.SysMenu{}).Count(&count)
	if count != 7 {
		t.Fatalf("duplicate or missing menu: %d", count)
	}
}
