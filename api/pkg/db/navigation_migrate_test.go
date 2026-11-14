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
		{ID: 11, ParentId: 1, MenuName: "监控诊断", Url: "/database/diagnostics", MenuType: 2, MenuStatus: 2},
		{ID: 12, ParentId: 1, MenuName: "参数配置", Url: "/database/configurations", MenuType: 2, MenuStatus: 2},
		{ID: 13, ParentId: 1, MenuName: "操作审计", Url: "database/audit", MenuType: 2, MenuStatus: 2},
		{ID: 14, ParentId: 3, MenuName: "数据日志", Url: "monitor/dblog", MenuType: 2, MenuStatus: 2},
		{ID: 7, ParentId: 1, MenuName: "凭据授权", Url: "cmdb/credential-permission", MenuType: 2, MenuStatus: 2},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatal(err)
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 22, MenuId: 13}, {RoleId: 12, MenuId: 10}, {RoleId: 10, MenuId: 4}, {RoleId: 11, MenuId: 6}, {RoleId: 11, MenuId: 12}} {
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
	var oldAudit, dataLog model.SysMenu
	db.First(&oldAudit, 13)
	db.First(&dataLog, 14)
	if oldAudit.MenuStatus != 1 || dataLog.ParentId != 3 || dataLog.MenuName != "数据日志" {
		t.Fatal("audit not merged into basic data logs")
	}
	for _, id := range []uint{3, 14} {
		var n int64
		db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 22, id).Count(&n)
		if n != 1 {
			t.Fatal("audit role assignment lost", id)
		}
	}
	var databaseChildren []model.SysMenu
	db.Where("parent_id = ? AND menu_status = 2", parent.ID).Order("sort").Find(&databaseChildren)
	if len(databaseChildren) != 5 {
		t.Fatal("unexpected database submenu count", len(databaseChildren))
	}
	for i, item := range databaseChildren {
		if item.Sort != uint(i+1) || item.Icon == "" {
			t.Fatal("database menu order/icon not normalized", item)
		}
	}

	var links int64
	db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 12, parent.ID).Count(&links)
	if links != 1 {
		t.Fatal("new parent role assignment missing")
	}
	var activeDuplicates int64
	db.Model(&model.SysMenu{}).Where("id IN ? AND menu_status = 2", []uint{6, 7, 8, 11, 12}).Count(&activeDuplicates)
	if activeDuplicates != 0 {
		t.Fatal("obsolete menus remain active")
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 10, MenuId: 4}, {RoleId: 10, MenuId: 2}, {RoleId: 11, MenuId: 5}, {RoleId: 11, MenuId: 3}, {RoleId: 11, MenuId: 6}, {RoleId: 11, MenuId: 12}} {
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
	if count != 12 {
		t.Fatalf("duplicate or missing menu: %d", count)
	}
}

func TestDatacenterNavigationPreservesExistingRoomMenu(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "datacenter.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.SysMenu{ID: 1, MenuName: "资产管理", MenuType: 1})
	db.Create(&model.SysMenu{ID: 2, MenuName: "机房信息", MenuType: 2, Url: "system/machine", Value: "cmdb:read", MenuStatus: 2})
	db.Create(&model.SysRoleMenu{RoleId: 7, MenuId: 2})
	for i := 0; i < 2; i++ {
		if err = migrateManagementNavigation(db); err != nil {
			t.Fatal(err)
		}
	}
	var room model.SysMenu
	db.First(&room, 2)
	if room.Url != "cmdb/datacenter" || room.MenuName != "机房管理" || room.ParentId != 1 || room.Value != "cmdb:read" {
		t.Fatalf("lost room menu: %+v", room)
	}
	var n int64
	db.Model(&model.SysMenu{}).Where("url = ?", "cmdb/datacenter").Count(&n)
	if n != 1 {
		t.Fatal("3D menu duplicated or missing")
	}
	db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 7, 2).Count(&n)
	if n != 1 {
		t.Fatal("role link lost")
	}
}

func TestDatacenterNavigationMergesBothMenusAndRoles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "merged-navigation.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err = db.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, menu := range []model.SysMenu{
		{ID: 1, MenuName: "资产管理", MenuType: 1, MenuStatus: 2},
		{ID: 2, MenuName: "机房与机柜", Url: "/cmdb/machine", MenuType: 2, MenuStatus: 2},
		{ID: 3, MenuName: "机房3D", Url: "cmdb/datacenter", MenuType: 2, MenuStatus: 2, Value: "cmdb:read"},
		{ID: 4, MenuName: "修改机柜", ParentId: 2, MenuType: 3, Value: "cmdb:write", MenuStatus: 2},
	} {
		if err = db.Create(&menu).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 7, MenuId: 2}, {RoleId: 8, MenuId: 3}, {RoleId: 7, MenuId: 4}} {
		if err = db.Create(&link).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err = migrateManagementNavigation(db); err != nil {
			t.Fatal(err)
		}
	}
	var menus []model.SysMenu
	db.Where("menu_type = 2 AND menu_status = 2 AND url IN ?", []string{"/cmdb/machine", "cmdb/datacenter"}).Find(&menus)
	if len(menus) != 1 || menus[0].ID != 3 || menus[0].MenuName != "机房管理" || menus[0].Value != "cmdb:read" {
		t.Fatalf("unified menu incorrect: %+v", menus)
	}
	for _, link := range []model.SysRoleMenu{{RoleId: 7, MenuId: 3}, {RoleId: 8, MenuId: 3}, {RoleId: 7, MenuId: 1}, {RoleId: 8, MenuId: 1}, {RoleId: 7, MenuId: 4}} {
		var n int64
		db.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", link.RoleId, link.MenuId).Count(&n)
		if n != 1 {
			t.Fatalf("lost or duplicate role assignment: %+v (%d)", link, n)
		}
	}
	var action model.SysMenu
	db.First(&action, 4)
	if action.ParentId != 3 || action.Value != "cmdb:write" {
		t.Fatalf("inventory action not preserved under unified menu: %+v", action)
	}
}
