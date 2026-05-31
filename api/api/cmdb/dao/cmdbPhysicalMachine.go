package dao

import (
	"dodevops-api/api/cmdb/model"
	"fmt"

	"gorm.io/gorm"
)

type CmdbPhysicalMachineDao struct {
	db *gorm.DB
}

func NewCmdbPhysicalMachineDao(db *gorm.DB) *CmdbPhysicalMachineDao {
	return &CmdbPhysicalMachineDao{db: db}
}

func (d *CmdbPhysicalMachineDao) Create(data *model.CmdbPhysicalMachine) error {
	return d.db.Create(data).Error
}

func (d *CmdbPhysicalMachineDao) Update(data *model.CmdbPhysicalMachine) error {
	data.IDC = nil
	data.Cabinet = nil
	return d.db.Save(data).Error
}

func (d *CmdbPhysicalMachineDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbPhysicalMachine{}, id).Error
}

func (d *CmdbPhysicalMachineDao) GetByID(id uint) (*model.CmdbPhysicalMachine, error) {
	var data model.CmdbPhysicalMachine
	err := d.db.Preload("IDC").Preload("Cabinet").First(&data, id).Error
	return &data, err
}

func (d *CmdbPhysicalMachineDao) GetList(query model.CmdbPhysicalMachineQuery) ([]model.CmdbPhysicalMachine, int64, error) {
	var list []model.CmdbPhysicalMachine
	var total int64
	db := d.db.Model(&model.CmdbPhysicalMachine{}).Preload("IDC").Preload("Cabinet")

	if query.SN != "" {
		db = db.Where("sn LIKE ?", "%"+query.SN+"%")
	}
	if query.HostName != "" {
		db = db.Where("host_name LIKE ?", "%"+query.HostName+"%")
	}
	if query.ManageIP != "" {
		db = db.Where("manage_ip LIKE ?", "%"+query.ManageIP+"%")
	}
	if query.IDCID > 0 {
		db = db.Where("idc_id = ?", query.IDCID)
	}
	if query.CabinetID > 0 {
		db = db.Where("cabinet_id = ?", query.CabinetID)
	}
	if query.AssetStatus > 0 {
		db = db.Where("asset_status = ?", query.AssetStatus)
	}
	if query.Brand != "" {
		db = db.Where("brand LIKE ?", "%"+query.Brand+"%")
	}
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("sn LIKE ? OR host_name LIKE ? OR manage_ip LIKE ? OR business_ip LIKE ? OR model LIKE ?",
			kw, kw, kw, kw, kw)
	}

	db.Count(&total)
	page, size := query.Page, query.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// GetListByIDs 根据ID列表获取物理机
func (d *CmdbPhysicalMachineDao) GetListByIDs(ids []uint) ([]model.CmdbPhysicalMachine, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.CmdbPhysicalMachine
	err := d.db.Where("id IN ?", ids).Preload("IDC").Preload("Cabinet").Find(&list).Error
	return list, err
}

// CheckSNExists 检查SN是否存在
func (d *CmdbPhysicalMachineDao) CheckSNExists(sn string, excludeID uint) bool {
	var count int64
	db := d.db.Model(&model.CmdbPhysicalMachine{}).Where("sn = ?", sn)
	if excludeID > 0 {
		db = db.Where("id != ?", excludeID)
	}
	db.Count(&count)
	return count > 0
}

// GetAll 获取所有物理机
func (d *CmdbPhysicalMachineDao) GetAll() ([]model.CmdbPhysicalMachine, error) {
	var list []model.CmdbPhysicalMachine
	err := d.db.Preload("IDC").Preload("Cabinet").Order("id ASC").Find(&list).Error
	return list, err
}

// GetStats 获取统计信息
func (d *CmdbPhysicalMachineDao) GetStats() map[string]int64 {
	result := make(map[string]int64)
	var total int64
	d.db.Model(&model.CmdbPhysicalMachine{}).Count(&total)
	result["total"] = total

	for i := 1; i <= 5; i++ {
		var count int64
		d.db.Model(&model.CmdbPhysicalMachine{}).Where("asset_status = ?", i).Count(&count)
		statusNames := map[int]string{1: "in_stock", 2: "online", 3: "repair", 4: "offline", 5: "scrap"}
		result[fmt.Sprintf("status_%s", statusNames[i])] = count
	}
	return result
}
