package dao

import (
	"dodevops-api/api/cmdb/model"

	"gorm.io/gorm"
)

type CmdbNetworkDeviceDao struct {
	db *gorm.DB
}

func NewCmdbNetworkDeviceDao(db *gorm.DB) *CmdbNetworkDeviceDao {
	return &CmdbNetworkDeviceDao{db: db}
}

func (d *CmdbNetworkDeviceDao) Create(data *model.CmdbNetworkDevice) error {
	return d.db.Create(data).Error
}

func (d *CmdbNetworkDeviceDao) Update(data *model.CmdbNetworkDevice) error {
	data.IDC = nil
	data.Cabinet = nil
	return d.db.Save(data).Error
}

func (d *CmdbNetworkDeviceDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbNetworkDevice{}, id).Error
}

func (d *CmdbNetworkDeviceDao) GetByID(id uint) (*model.CmdbNetworkDevice, error) {
	var data model.CmdbNetworkDevice
	err := d.db.Preload("IDC").Preload("Cabinet").First(&data, id).Error
	return &data, err
}

func (d *CmdbNetworkDeviceDao) GetList(query model.CmdbNetworkDeviceQuery) ([]model.CmdbNetworkDevice, int64, error) {
	var list []model.CmdbNetworkDevice
	var total int64
	db := d.db.Model(&model.CmdbNetworkDevice{}).Preload("IDC").Preload("Cabinet")

	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.DeviceType > 0 {
		db = db.Where("device_type = ?", query.DeviceType)
	}
	if query.ManageIP != "" {
		db = db.Where("manage_ip LIKE ?", "%"+query.ManageIP+"%")
	}
	if query.IDCID > 0 {
		db = db.Where("idc_id = ?", query.IDCID)
	}
	if query.Brand != "" {
		db = db.Where("brand LIKE ?", "%"+query.Brand+"%")
	}
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		db = db.Where("sn LIKE ? OR name LIKE ? OR manage_ip LIKE ? OR model LIKE ?", kw, kw, kw, kw)
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

// GetListByIDs 根据ID列表获取网络设备
func (d *CmdbNetworkDeviceDao) GetListByIDs(ids []uint) ([]model.CmdbNetworkDevice, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.CmdbNetworkDevice
	err := d.db.Where("id IN ?", ids).Preload("IDC").Preload("Cabinet").Find(&list).Error
	return list, err
}

func (d *CmdbNetworkDeviceDao) GetAll() ([]model.CmdbNetworkDevice, error) {
	var list []model.CmdbNetworkDevice
	err := d.db.Preload("IDC").Preload("Cabinet").Order("id ASC").Find(&list).Error
	return list, err
}
