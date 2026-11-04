package db

import (
	"dodevops-api/api/system/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestMenuIconDefaultsDoNotChangeAccessOrNavigation(t *testing.T) {
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "icons.db")), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if e = db.AutoMigrate(&model.SysMenu{}, &model.SysRoleMenu{}); e != nil {
		t.Fatal(e)
	}
	rows := []model.SysMenu{
		{ID: 1, MenuType: 1, MenuName: "资产管理", Url: "cmdb", MenuStatus: 2},
		{ID: 2, MenuType: 2, ParentId: 1, MenuName: "主机管理", Url: "/cmdb/ecs", MenuStatus: 2, Value: "cmdb:read"},
		{ID: 3, MenuType: 2, MenuName: "自选图标", Url: "system/admin", Icon: "UserFilled", MenuStatus: 2},
		{ID: 4, MenuType: 3, MenuName: "按钮", Url: "cmdb/group"},
		{ID: 5, MenuType: 1, MenuName: "AI运维", Url: "aiops", Icon: "ri/robot-2-line", MenuStatus: 2},
		{ID: 6, MenuType: 2, MenuName: "停用页面", Url: "k8s/node", MenuStatus: 1, Sort: 99, Value: "k8s:read"},
		{ID: 7, MenuType: 2, MenuName: "扩展页面", Url: "custom/my-feature", MenuStatus: 2},
	}
	if e = db.Create(&rows).Error; e != nil {
		t.Fatal(e)
	}
	link := model.SysRoleMenu{RoleId: 20, MenuId: 2}
	db.Create(&link)
	for i := 0; i < 2; i++ {
		if e = UpdateNavigationIcons(db); e != nil {
			t.Fatal(e)
		}
	}
	var got []model.SysMenu
	db.Order("id").Find(&got)
	want := []string{"Monitor", "Monitor", "UserFilled", "", "ChatDotRound", "Cpu", "Menu"}
	for i, row := range got {
		if row.Icon != want[i] {
			t.Fatalf("wrong icon %d: %s", row.ID, row.Icon)
		}
		original := rows[i]
		if row.ParentId != original.ParentId || row.Url != original.Url || row.Value != original.Value || row.MenuStatus != original.MenuStatus || row.Sort != original.Sort {
			t.Fatalf("icon migration changed navigation/access: %+v", row)
		}
	}
	var n int64
	db.Model(&model.SysRoleMenu{}).Where("role_id=? AND menu_id=?", 20, 2).Count(&n)
	if n != 1 {
		t.Fatal("role link changed")
	}
}

func TestMenuIconRegistryMatchesWeb(t *testing.T) {
	data, e := os.ReadFile("../../../web/src/utils/menuIcons.ts")
	if os.IsNotExist(e) {
		t.Skip("web sources not packaged")
	}
	if e != nil {
		t.Fatal(e)
	}
	matches := regexp.MustCompile(`"(/[^"\n]*)":\s*"([A-Za-z]+)"`).FindAllSubmatch(data, -1)
	if len(matches) != len(navigationIconNames) {
		t.Fatalf("registry count mismatch: web=%d api=%d", len(matches), len(navigationIconNames))
	}
	for _, m := range matches {
		if navigationIconNames[string(m[1])] != string(m[2]) {
			t.Fatalf("registry mismatch for %s", m[1])
		}
	}
}
