package db

import (
	"dodevops-api/api/system/model"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWebNavigationIsIdempotentAndPreservesRoleAssignments(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "web-menus.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := database.DB()
	defer raw.Close()
	if err = database.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)", "INSERT INTO sys_role VALUES (1,'admin',1),(2,'reader',1)"} {
		if err = database.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	menu := model.SysMenu{ID: 1, MenuName: "Existing database page", MenuType: 2, MenuStatus: 2, Url: "database/sources"}
	database.Create(&menu)
	database.Create(&model.SysRoleMenu{RoleId: 2, MenuId: 1})
	for i := 0; i < 2; i++ {
		if err = migrateWebNavigation(database); err != nil {
			t.Fatal(err)
		}
	}
	var menus, adminLinks, readerLinks int64
	database.Model(&model.SysMenu{}).Count(&menus)
	database.Model(&model.SysRoleMenu{}).Where("role_id = 1").Count(&adminLinks)
	database.Model(&model.SysRoleMenu{}).Where("role_id = 2 AND menu_id = 1").Count(&readerLinks)
	if menus != 9 || adminLinks != 8 || readerLinks != 1 {
		t.Fatalf("unexpected migration counts: menus=%d admin=%d existing=%d", menus, adminLinks, readerLinks)
	}
	var original model.SysMenu
	database.First(&original, 1)
	if original.Url != menu.Url || original.MenuName != menu.MenuName {
		t.Fatal("existing navigation was overwritten")
	}
}

func TestWebNavigationRetiresTerminalWithoutDeletingHistory(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "web-retired-menu.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := database.DB()
	defer raw.Close()
	if err = database.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); err != nil {
		t.Fatal(err)
	}
	terminal := model.SysMenu{MenuName: "终端", Url: "web/terminal", MenuType: 2, MenuStatus: 2}
	if err = database.Create(&terminal).Error; err != nil {
		t.Fatal(err)
	}
	link := model.SysRoleMenu{RoleId: 77, MenuId: terminal.ID}
	if err = database.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		if err = migrateWebNavigation(database); err != nil {
			t.Fatal(err)
		}
	}
	var saved model.SysMenu
	if err = database.First(&saved, terminal.ID).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	database.Model(&model.SysRoleMenu{}).Where("role_id = ? AND menu_id = ?", 77, terminal.ID).Count(&count)
	if saved.MenuStatus != 1 || count != 1 {
		t.Fatalf("terminal retirement lost history or left menu active: status=%d links=%d", saved.MenuStatus, count)
	}
}
