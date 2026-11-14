package datacenterseed

import (
	"path/filepath"
	"testing"

	"dodevops-api/api/cmdb/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "seed.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	t.Cleanup(func() { raw.Close() })
	if err = db.AutoMigrate(&model.CmdbIDC{}, &model.CmdbCabinet{}, &model.CmdbPhysicalMachine{}, &model.CmdbNetworkDevice{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestSeedAddsInventoryWithoutOverwritingOrDuplicating(t *testing.T) {
	db := testDB(t)
	real := model.CmdbPhysicalMachine{SN: "real-sn", HostName: "existing-host", Remark: "user inventory"}
	if err := db.Create(&real).Error; err != nil {
		t.Fatal(err)
	}
	first, err := Seed(db)
	if err != nil {
		t.Fatal(err)
	}
	expected := Counts{Rooms: 5, Cabinets: 50, Servers: 550, Networks: 100}
	if first.Created != expected || len(first.RoomIDs) != 5 {
		t.Fatalf("unexpected receipt: %+v", first)
	}
	var cabinet model.CmdbCabinet
	if err = db.Where("name = ?", "TEST-BJ-A01").First(&cabinet).Error; err != nil {
		t.Fatal(err)
	}
	if cabinet.UsedUnit != 24 {
		t.Fatalf("U occupancy not calculated: %d", cabinet.UsedUnit)
	}
	if err = db.Model(&model.CmdbPhysicalMachine{}).Where("sn = ?", "AUTOOPS-IDC-SAMPLE-TEST-BJ-A01-01").Update("host_name", "user-edited-sample").Error; err != nil {
		t.Fatal(err)
	}
	second, err := Seed(db)
	if err != nil {
		t.Fatal(err)
	}
	if second.Created != (Counts{}) || second.Existing != expected {
		t.Fatalf("seed not idempotent: %+v", second)
	}
	var saved model.CmdbPhysicalMachine
	db.First(&saved, real.ID)
	if saved.HostName != real.HostName || saved.Remark != real.Remark {
		t.Fatal("existing inventory changed")
	}
	var n int64
	db.Model(&model.CmdbPhysicalMachine{}).Where("host_name = ?", "user-edited-sample").Count(&n)
	if n != 1 {
		t.Fatal("user edits to sample were overwritten")
	}
	db.Model(&model.CmdbPhysicalMachine{}).Where("remark = ? AND manage_ip <> ''", note).Count(&n)
	if n != 0 {
		t.Fatal("sample created routable connection targets")
	}
}
func TestSeedRollsBackIfIdentifiersBelongToExistingInventory(t *testing.T) {
	db := testDB(t)
	db.Create(&model.CmdbIDC{ID: 1, Name: "real room"})
	db.Create(&model.CmdbCabinet{Name: "TEST-SH-A01", IDCID: 1, Remark: "belongs to user"})
	receipt, err := Seed(db)
	if err == nil || receipt.Created != (Counts{}) {
		t.Fatal("identifier collision did not abort seed")
	}
	for table, want := range map[string]int64{"cmdb_idc": 1, "cmdb_cabinet": 1, "cmdb_physical_machine": 0, "cmdb_network_device": 0} {
		var n int64
		db.Table(table).Count(&n)
		if n != want {
			t.Fatalf("partial seed persisted in %s: %d", table, n)
		}
	}
}
