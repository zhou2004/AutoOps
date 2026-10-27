package dao

import (
	"dodevops-api/api/cmdb/model"
	monitor "dodevops-api/api/monitor/model"
	"dodevops-api/common/util"
	"testing"
	"time"
)

func TestBusinessGroupHierarchyValidationAndReferences(t *testing.T) {
	db := permissionTestDB(t)
	if err := db.AutoMigrate(&model.CmdbSQL{}, &monitor.MonitorAlertGroupRule{}); err != nil {
		t.Fatal(err)
	}
	d := CmdbGroupDao{db: db}
	for _, g := range []model.CmdbGroup{{CreateTime: util.HTime{Time: time.Now()}, ID: 1, Name: "默认"}, {CreateTime: util.HTime{Time: time.Now()}, ID: 2, Name: "支付"}, {CreateTime: util.HTime{Time: time.Now()}, ID: 3, ParentID: 2, Name: "生产"}} {
		if err := db.Create(&g).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Substring names are distinct; whitespace and exact duplicates are rejected.
	group := model.CmdbGroup{Name: "支付核心", CreateTime: util.HTime{Time: time.Now()}}
	if err := d.CreateCmdbGroup(&group); err != nil {
		t.Fatal(err)
	}
	for _, g := range []model.CmdbGroup{{Name: " 支付 "}, {Name: "   "}, {Name: "bad", ParentID: 999}, {Name: "bad", ParentID: 1}} {
		if err := d.CreateCmdbGroup(&g); err == nil {
			t.Fatal("invalid create accepted", g)
		}
	}
	for _, g := range []model.CmdbGroup{{ID: 2, Name: "支付", ParentID: 2}, {ID: 2, Name: "支付", ParentID: 3}, {ID: 2, Name: "生产"}, {ID: 1, Name: "new"}} {
		if err := d.UpdateCmdbGroup(g.ID, &g); err == nil {
			t.Fatal("invalid move accepted", g)
		}
	}
	if err := d.DeleteCmdbGroup(2); err == nil {
		t.Fatal("parent group deleted")
	}
	if err := d.UpdateCmdbGroup(3, &model.CmdbGroup{Name: "生产", ParentID: 0}); err != nil {
		t.Fatal(err)
	}
	var child model.CmdbGroup
	db.First(&child, 3)
	if child.ParentID != 0 {
		t.Fatal("zero parent ID ignored")
	}
	linked := monitor.MonitorAlertGroupRule{CmdbGroupID: 2, GroupName: "alerts"}
	db.Create(&linked)
	if err := d.DeleteCmdbGroup(2); err == nil {
		t.Fatal("referenced business group deleted")
	}
	db.Unscoped().Delete(&linked)
	if err := d.DeleteCmdbGroup(2); err != nil {
		t.Fatal(err)
	}
	if err := d.DeleteCmdbGroup(1); err == nil {
		t.Fatal("default group deleted")
	}
}
