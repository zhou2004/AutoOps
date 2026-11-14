package service

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"gorm.io/gorm"
)

type RackDevice struct {
	ID           uint   `json:"id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	SN           string `json:"sn"`
	ManageIP     string `json:"manageIp"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	CabinetID    uint   `json:"cabinetId"`
	UnitPosition int    `json:"unitPosition"`
	UnitHeight   int    `json:"unitHeight"`
	DeviceType   int    `json:"deviceType"`
	Status       int    `json:"status"`
	AssetStatus  int    `json:"assetStatus"`
	Restricted   bool   `json:"restricted"`
}
type RackView struct {
	model.CmdbCabinet
	Devices           []RackDevice `json:"devices"`
	OccupiedUnits     []int        `json:"occupiedUnits"`
	ConflictUnits     []int        `json:"conflictUnits"`
	Unpositioned      int          `json:"unpositioned"`
	InvalidPlacements int          `json:"invalidPlacements"`
}
type RackTopology struct {
	Room     model.CmdbIDC `json:"room"`
	Cabinets []RackView    `json:"cabinets"`
	Unplaced []RackDevice  `json:"unplaced"`
}

// Nil scope means administrator. Empty scopes grant nothing.
func LoadRackTopology(db *gorm.DB, id uint, scopes map[string]map[uint]bool) (*RackTopology, error) {
	out := &RackTopology{Cabinets: []RackView{}, Unplaced: []RackDevice{}}
	if err := db.First(&out.Room, id).Error; err != nil {
		return nil, err
	}
	var cabinets []model.CmdbCabinet
	q := db.Where("idc_id = ?", id).Order("name ASC")
	if scopes != nil {
		ids := scopes["cabinet"]
		if ids == nil {
			ids = map[uint]bool{}
		}
		q = dao.ApplyAssetIDScope(q, ids)
	}
	if err := q.Find(&cabinets).Error; err != nil {
		return nil, err
	}
	byID := map[uint]int{}
	for _, rack := range cabinets {
		byID[rack.ID] = len(out.Cabinets)
		out.Cabinets = append(out.Cabinets, RackView{CmdbCabinet: rack, Devices: []RackDevice{}, OccupiedUnits: []int{}, ConflictUnits: []int{}})
	}
	all := []RackDevice{}
	var physical []model.CmdbPhysicalMachine
	var network []model.CmdbNetworkDevice
	if err := db.Where("idc_id = ?", id).Order("id ASC").Find(&physical).Error; err != nil {
		return nil, err
	}
	if err := db.Where("idc_id = ?", id).Order("id ASC").Find(&network).Error; err != nil {
		return nil, err
	}
	for _, p := range physical {
		all = append(all, RackDevice{ID: p.ID, Kind: "physical", Name: p.HostName, SN: p.SN, ManageIP: p.ManageIP, Brand: p.Brand, Model: p.Model, CabinetID: p.CabinetID, UnitPosition: p.UnitPosition, UnitHeight: max(p.UnitHeight, 1), Status: p.Status, AssetStatus: p.AssetStatus})
	}
	for _, p := range network {
		all = append(all, RackDevice{ID: p.ID, Kind: "network", Name: p.Name, SN: p.SN, ManageIP: p.ManageIP, Brand: p.Brand, Model: p.Model, CabinetID: p.CabinetID, UnitPosition: p.UnitPosition, UnitHeight: max(p.UnitHeight, 1), DeviceType: p.DeviceType, Status: p.Status, AssetStatus: p.AssetStatus})
	}
	for _, p := range all {
		allowed := scopes == nil || scopes[p.Kind][p.ID]
		index, ok := byID[p.CabinetID]
		if !ok {
			if p.CabinetID == 0 && allowed {
				out.Unplaced = append(out.Unplaced, p)
			}
			continue
		}
		if !allowed {
			p = RackDevice{Kind: "restricted", Name: "受限资产", CabinetID: p.CabinetID, UnitPosition: p.UnitPosition, UnitHeight: p.UnitHeight, Restricted: true}
		}
		out.Cabinets[index].Devices = append(out.Cabinets[index].Devices, p)
	}
	for i := range out.Cabinets {
		r := &out.Cabinets[i]
		counts := map[int]int{}
		for _, p := range r.Devices {
			if p.UnitPosition == 0 {
				r.Unpositioned++
				continue
			}
			if p.UnitPosition < 1 || p.UnitPosition+p.UnitHeight-1 > r.UnitNum {
				r.InvalidPlacements++
			}
			for u := max(p.UnitPosition, 1); u <= r.UnitNum && u < p.UnitPosition+p.UnitHeight; u++ {
				counts[u]++
			}
		}
		for u := 1; u <= r.UnitNum; u++ {
			if counts[u] > 0 {
				r.OccupiedUnits = append(r.OccupiedUnits, u)
			}
			if counts[u] > 1 {
				r.ConflictUnits = append(r.ConflictUnits, u)
			}
		}
		r.UsedUnit = len(r.OccupiedUnits)
	}
	return out, nil
}
