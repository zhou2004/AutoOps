package dao

import (
	"dodevops-api/api/cmdb/model"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"sort"
)

var ErrRackPlacement = errors.New("机柜或 U 位配置无效")

func rackValidation(message string) error { return fmt.Errorf("%w: %s", ErrRackPlacement, message) }

type rackPlacement struct {
	ID           uint
	IDCID        uint `gorm:"column:idc_id"`
	CabinetID    uint
	UnitPosition int
	UnitHeight   int
}

// The cabinet row is the common lock for physical and network inventory writes.
// Unknown positions (0) remain explicit and consume no fabricated U slots.
func saveRackAsset(db *gorm.DB, table string, id uint, idc, cabinet uint, start int, height *int, data any) error {
	if *height == 0 {
		*height = 1
	}
	if start < 0 || start > 100 || *height < 1 || *height > 100 {
		return rackValidation("起始 U 位须为 0–100，设备高度须为 1–100 U")
	}
	if cabinet == 0 && start != 0 {
		return rackValidation("请先选择机柜，再设置 U 位")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		ids := []uint{cabinet}
		if id > 0 {
			var old rackPlacement
			if err := tx.Table(table).Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, id).Error; err != nil {
				return err
			}
			ids = append(ids, old.CabinetID)
		}
		if idc > 0 {
			var room model.CmdbIDC
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, idc).Error; err != nil {
				return rackValidation("机房不存在")
			}
		}
		cabinets, err := lockRacks(tx, ids)
		if err != nil {
			return err
		}

		if cabinet > 0 {
			rack, ok := cabinets[cabinet]
			if !ok {
				return rackValidation("机柜不存在")
			}
			if rack.IDCID != idc {
				return rackValidation("机柜不属于所选机房")
			}
			if start > 0 {
				if start+*height-1 > rack.UnitNum {
					return rackValidation("设备超出机柜 U 位范围")
				}
				for _, t := range []string{"cmdb_physical_machine", "cmdb_network_device"} {
					var count int64
					q := tx.Table(t).Where("cabinet_id = ? AND unit_position > 0 AND unit_position <= ? AND unit_position + CASE WHEN unit_height > 0 THEN unit_height ELSE 1 END - 1 >= ?", cabinet, start+*height-1, start)
					if t == table && id > 0 {
						q = q.Where("id <> ?", id)
					}
					if err = q.Count(&count).Error; err != nil {
						return err
					}
					if count > 0 {
						return rackValidation("该 U 位范围已被物理机或网络设备占用")
					}
				}
			}
		}
		if id == 0 {
			err = tx.Omit(clause.Associations).Create(data).Error
		} else {
			err = tx.Omit(clause.Associations).Save(data).Error
		}
		if err != nil {
			return err
		}
		return refreshRackUsage(tx, ids)
	})
}

func lockRacks(tx *gorm.DB, ids []uint) (map[uint]model.CmdbCabinet, error) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := map[uint]model.CmdbCabinet{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := out[id]; ok {
			continue
		}
		var rack model.CmdbCabinet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&rack, id).Error; err != nil {
			return nil, rackValidation("机柜不存在")
		}
		out[id] = rack
	}
	return out, nil
}

func refreshRackUsage(tx *gorm.DB, ids []uint) error {
	for _, id := range ids {
		if id == 0 {
			continue
		}
		var rack model.CmdbCabinet
		if err := tx.First(&rack, id).Error; err != nil {
			return err
		}
		used := map[int]bool{}
		for _, table := range []string{"cmdb_physical_machine", "cmdb_network_device"} {
			var list []rackPlacement
			if err := tx.Table(table).Where("cabinet_id = ? AND unit_position > 0", id).Find(&list).Error; err != nil {
				return err
			}
			for _, p := range list {
				height := p.UnitHeight
				if height < 1 {
					height = 1
				}
				for u := p.UnitPosition; u < p.UnitPosition+height && u <= rack.UnitNum; u++ {
					used[u] = true
				}
			}
		}
		if err := tx.Model(&rack).Update("used_unit", len(used)).Error; err != nil {
			return err
		}
	}
	return nil
}
func deleteRackAsset(db *gorm.DB, table string, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var old rackPlacement
		if err := tx.Table(table).Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, id).Error; err != nil {
			return err
		}
		if _, err := lockRacks(tx, []uint{old.CabinetID}); err != nil {
			return err
		}
		if err := tx.Table(table).Where("id = ?", id).Delete(&rackPlacement{}).Error; err != nil {
			return err
		}
		return refreshRackUsage(tx, []uint{old.CabinetID})
	})
}
func saveRack(db *gorm.DB, data *model.CmdbCabinet) error {
	if data.UnitNum == 0 {
		data.UnitNum = 42
	}
	if data.WidthMM == 0 {
		data.WidthMM = 600
	}
	if data.DepthMM == 0 {
		data.DepthMM = 1000
	}
	if data.UnitNum < 1 || data.UnitNum > 100 || data.WidthMM < 400 || data.WidthMM > 1200 || data.DepthMM < 400 || data.DepthMM > 2000 {
		return rackValidation("机柜容量须为 1–100 U，宽度 400–1200 mm，深度 400–2000 mm")
	}
	if data.Rotation < 0 || data.Rotation >= 360 || data.Rotation%90 != 0 {
		return rackValidation("朝向须为 0、90、180 或 270 度")
	}
	if (data.LayoutX == nil) != (data.LayoutZ == nil) {
		return rackValidation("请同时填写 X、Z 坐标")
	}
	for _, v := range []*float64{data.LayoutX, data.LayoutZ} {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || math.Abs(*v) > 500) {
			return rackValidation("机柜坐标须在 -500 到 500 米之间")
		}
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var room model.CmdbIDC
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, data.IDCID).Error; err != nil {
			return rackValidation("机房不存在")
		}
		if data.ID > 0 {
			racks, err := lockRacks(tx, []uint{data.ID})
			if err != nil {
				return err
			}
			for _, table := range []string{"cmdb_physical_machine", "cmdb_network_device"} {
				var list []rackPlacement
				if err := tx.Table(table).Where("cabinet_id = ?", data.ID).Find(&list).Error; err != nil {
					return err
				}
				if len(list) > 0 && racks[data.ID].IDCID != data.IDCID {
					return rackValidation("机柜仍有关联资产，不能变更所属机房")
				}
				for _, p := range list {
					if p.UnitPosition > 0 && p.UnitPosition+max(p.UnitHeight, 1)-1 > data.UnitNum {
						return rackValidation("缩小容量会使已上架设备超出 U 位范围")
					}
				}
			}
		}
		if data.LayoutX != nil {
			var others []model.CmdbCabinet
			if err := tx.Where("idc_id = ? AND id <> ? AND layout_x IS NOT NULL AND layout_z IS NOT NULL", data.IDCID, data.ID).Find(&others).Error; err != nil {
				return err
			}
			w, d := rackFootprint(*data)
			for _, other := range others {
				ow, od := rackFootprint(other)
				if math.Abs(*data.LayoutX-*other.LayoutX) < (w+ow)/2-0.000001 && math.Abs(*data.LayoutZ-*other.LayoutZ) < (d+od)/2-0.000001 {
					return rackValidation("机柜底面与已有机柜重叠，请调整坐标或朝向")
				}
			}
		}
		data.UsedUnit = 0
		if err := tx.Omit(clause.Associations).Save(data).Error; err != nil {
			return err
		}
		if err := refreshRackUsage(tx, []uint{data.ID}); err != nil {
			return err
		}
		return tx.First(data, data.ID).Error
	})
}
func deleteRack(db *gorm.DB, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if _, err := lockRacks(tx, []uint{id}); err != nil {
			return err
		}
		for _, table := range []string{"cmdb_physical_machine", "cmdb_network_device"} {
			var n int64
			if err := tx.Table(table).Where("cabinet_id = ?", id).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return rackValidation("机柜仍有关联设备，请先移出设备")
			}
		}
		return tx.Delete(&model.CmdbCabinet{}, id).Error
	})
}

func rackFootprint(r model.CmdbCabinet) (float64, float64) {
	w, d := float64(r.WidthMM)/1000, float64(r.DepthMM)/1000
	if w == 0 {
		w = 0.6
	}
	if d == 0 {
		d = 1
	}
	if r.Rotation == 90 || r.Rotation == 270 {
		return d, w
	}
	return w, d
}
