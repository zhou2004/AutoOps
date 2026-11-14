package dao

import (
	"dodevops-api/api/cmdb/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"path/filepath"
	"testing"
)

func TestRackPlacementAcrossAssetTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rack.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.CmdbIDC{}, &model.CmdbCabinet{}, &model.CmdbPhysicalMachine{}, &model.CmdbNetworkDevice{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.CmdbIDC{ID: 1, Name: "A"})
	db.Create(&model.CmdbIDC{ID: 2, Name: "B"})
	cabinets := NewCmdbCabinetDao(db)
	physical := NewCmdbPhysicalMachineDao(db)
	network := NewCmdbNetworkDeviceDao(db)
	rack := model.CmdbCabinet{Name: "A01", IDCID: 1, UnitNum: 12}
	if err = cabinets.Create(&rack); err != nil {
		t.Fatal(err)
	}
	x, z := 0.0, 0.0
	rack.LayoutX, rack.LayoutZ = &x, &z
	if err = cabinets.Update(&rack); err != nil {
		t.Fatal(err)
	}
	other := model.CmdbCabinet{Name: "A02", IDCID: 1, UnitNum: 42, LayoutX: &x, LayoutZ: &z}
	if cabinets.Create(&other) == nil {
		t.Fatal("overlapping rack footprints accepted")
	}
	x2 := 1.0
	other.LayoutX = &x2
	if err = cabinets.Create(&other); err != nil {
		t.Fatal(err)
	}
	other.WidthMM = 1200
	other.DepthMM = 1800
	other.Rotation = 90
	if cabinets.Update(&other) == nil {
		t.Fatal("rotated footprint overlap accepted")
	}
	if err = cabinets.Delete(other.ID); err != nil {
		t.Fatal(err)
	}
	host := model.CmdbPhysicalMachine{SN: "host", IDCID: 1, CabinetID: rack.ID, UnitPosition: 1, UnitHeight: 2}
	if err = physical.Create(&host); err != nil {
		t.Fatal(err)
	}
	sw := model.CmdbNetworkDevice{SN: "switch", Name: "Switch", IDCID: 1, CabinetID: rack.ID, UnitPosition: 2, UnitHeight: 1}
	if network.Create(&sw) == nil {
		t.Fatal("cross-type overlap accepted")
	}
	sw.UnitPosition = 3
	if err = network.Create(&sw); err != nil {
		t.Fatal(err)
	}
	stored, _ := cabinets.GetByID(rack.ID)
	if stored.UsedUnit != 3 {
		t.Fatalf("used units: %d", stored.UsedUnit)
	}
	sw.UnitPosition = int(^uint(0) >> 1)
	if network.Update(&sw) == nil {
		t.Fatal("overflow U position accepted")
	}
	sw.UnitPosition = 12
	sw.UnitHeight = 2
	if network.Update(&sw) == nil {
		t.Fatal("out-of-bounds accepted")
	}
	sw.UnitHeight = 1
	sw.IDCID = 2
	if network.Update(&sw) == nil {
		t.Fatal("wrong room accepted")
	}
	sw.IDCID = 1
	sw.UnitPosition = 3
	sw.UnitHeight = 1
	rack.UnitNum = 2
	if cabinets.Update(&rack) == nil {
		t.Fatal("unsafe resize accepted")
	}
	rack.UnitNum = 12
	rack.IDCID = 2
	if cabinets.Update(&rack) == nil {
		t.Fatal("occupied rack moved between rooms")
	}
	rack.IDCID = 1
	if cabinets.Delete(rack.ID) == nil {
		t.Fatal("occupied rack deleted")
	}
	if NewCmdbIDCDao(db).Delete(1) == nil {
		t.Fatal("occupied room deleted")
	}
	host.UnitPosition = 0
	if err = physical.Update(&host); err != nil {
		t.Fatal(err)
	}
	stored, _ = cabinets.GetByID(rack.ID)
	if stored.UsedUnit != 1 {
		t.Fatalf("unknown position fabricated occupancy: %d", stored.UsedUnit)
	}
	if err = network.Delete(sw.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ = cabinets.GetByID(rack.ID)
	if stored.UsedUnit != 0 {
		t.Fatal("delete did not release occupancy")
	}
	if err = physical.Delete(host.ID); err != nil {
		t.Fatal(err)
	}
	if err = cabinets.Delete(rack.ID); err != nil {
		t.Fatal(err)
	}
}
