package service

import (
	"dodevops-api/api/cmdb/model"
	"encoding/json"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"strings"
	"testing"
)

func TestRackTopologyPrivacyAndHistoricalConflicts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "topology.db")), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	if err = db.AutoMigrate(&model.CmdbIDC{}, &model.CmdbCabinet{}, &model.CmdbPhysicalMachine{}, &model.CmdbNetworkDevice{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.CmdbIDC{ID: 1, Name: "A"})
	db.Create(&model.CmdbCabinet{ID: 1, Name: "A01", IDCID: 1, UnitNum: 4})
	db.Create(&model.CmdbPhysicalMachine{ID: 1, SN: "secret-sn", HostName: "private-host", ManageIP: "10.66.0.1", IDCID: 1, CabinetID: 1, UnitPosition: 1, UnitHeight: 2})
	db.Create(&model.CmdbNetworkDevice{ID: 1, SN: "visible-sn", Name: "Switch", IDCID: 1, CabinetID: 1, UnitPosition: 2, UnitHeight: 1})
	db.Create(&model.CmdbNetworkDevice{ID: 2, SN: "unknown", Name: "Unknown", IDCID: 1, CabinetID: 1, UnitPosition: 0, UnitHeight: 1})
	scopes := map[string]map[uint]bool{"cabinet": {1: true}, "physical": {}, "network": {1: true, 2: true}}
	out, err := LoadRackTopology(db, 1, scopes)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(out)
	for _, secret := range []string{"secret-sn", "private-host", "10.66.0.1"} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("unauthorized details leaked: %s", secret)
		}
	}
	rack := out.Cabinets[0]
	if rack.UsedUnit != 2 || len(rack.ConflictUnits) != 1 || rack.ConflictUnits[0] != 2 || rack.Unpositioned != 1 {
		t.Fatalf("incorrect occupancy: %+v", rack)
	}
	empty, err := LoadRackTopology(db, 1, map[string]map[uint]bool{"cabinet": {}, "physical": {}, "network": {}})
	if err != nil || len(empty.Cabinets) != 0 || len(empty.Unplaced) != 0 {
		t.Fatal("empty scope leaked assets")
	}
	admin, err := LoadRackTopology(db, 1, nil)
	if err != nil || admin.Cabinets[0].Devices[0].Name != "private-host" {
		t.Fatal("admin cannot read asset")
	}
}
