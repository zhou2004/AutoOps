package dao

import (
	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestDatabaseSourceFiltersPreserveScopeAndPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.CmdbSQL{}, &model.CmdbGroup{}, &configmodel.AccountAuth{}); err != nil {
		t.Fatal(err)
	}
	for _, g := range []model.CmdbGroup{{ID: 1, Name: "parent"}, {ID: 2, Name: "child", ParentID: 1}, {ID: 3, Name: "other"}} {
		g.CreateTime = util.HTime{Time: time.Now()}
		if e := db.Create(&g).Error; e != nil {
			t.Fatal(e)
		}
	}
	for _, r := range []model.CmdbSQL{{ID: 1, Name: "prod-main", Type: 1, GroupID: 1}, {ID: 2, Name: "prod-replica", Type: 1, GroupID: 2}, {ID: 3, Name: "prod-cache", Type: 3, GroupID: 2}, {ID: 4, Name: "prod-hidden", Type: 1, GroupID: 2}, {ID: 5, Name: "prod-other", Type: 1, GroupID: 3}} {
		if e := db.Create(&r).Error; e != nil {
			t.Fatal(e)
		}
	}
	dao := NewCmdbSQLDao(db)
	scope := map[uint]bool{1: true, 2: true, 3: true, 5: true}
	rows, total, err := dao.Search(1, 1, "prod-", 1, 1, scope)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(rows) != 1 || rows[0].ID != 2 || rows[0].GroupName != "child" {
		t.Fatalf("combined filters or count incorrect: %d %+v", total, rows)
	}
	rows, total, err = dao.Search(2, 1, "prod-", 1, 1, scope)
	if err != nil || total != 2 || len(rows) != 1 || rows[0].ID != 1 {
		t.Fatal("pagination incorrect")
	}
	rows, total, err = dao.Search(1, 10, "", 0, 0, map[uint]bool{})
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatal("empty permission scope leaked assets")
	}
	groups, err := dao.SourceGroups(map[uint]bool{2: true})
	if err != nil || len(groups) != 1 || groups[0].ID != 1 || len(groups[0].Children) != 1 || groups[0].Children[0].ID != 2 {
		t.Fatalf("group scope lost: %+v %v", groups, err)
	}
}
