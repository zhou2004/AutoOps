package dao

import (
	"dodevops-api/api/cmdb/model"

	"gorm.io/gorm"
)

type CmdbCabinetDao struct {
	db *gorm.DB
}

func NewCmdbCabinetDao(db *gorm.DB) *CmdbCabinetDao {
	return &CmdbCabinetDao{db: db}
}

func (d *CmdbCabinetDao) Create(data *model.CmdbCabinet) error {
	return d.db.Create(data).Error
}

func (d *CmdbCabinetDao) Update(data *model.CmdbCabinet) error {
	data.IDC = model.CmdbIDC{}
	return d.db.Save(data).Error
}

func (d *CmdbCabinetDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbCabinet{}, id).Error
}

func (d *CmdbCabinetDao) GetByID(id uint) (*model.CmdbCabinet, error) {
	var data model.CmdbCabinet
	err := d.db.Preload("IDC").First(&data, id).Error
	return &data, err
}

func (d *CmdbCabinetDao) GetList(query model.CmdbCabinetQuery) ([]model.CmdbCabinet, int64, error) {
	var list []model.CmdbCabinet
	var total int64
	db := d.db.Model(&model.CmdbCabinet{}).Preload("IDC")

	if query.Name != "" {
		db = db.Where("cmdb_cabinet.name LIKE ?", "%"+query.Name+"%")
	}
	if query.IDCID > 0 {
		db = db.Where("idc_id = ?", query.IDCID)
	}
	if query.Status > 0 {
		db = db.Where("cmdb_cabinet.status = ?", query.Status)
	}

	db.Count(&total)
	page, size := query.Page, query.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	err := db.Order("cmdb_cabinet.id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

// GetByIDC 根据机房ID获取机柜列表
func (d *CmdbCabinetDao) GetByIDC(idcID uint) ([]model.CmdbCabinet, error) {
	var list []model.CmdbCabinet
	err := d.db.Where("idc_id = ?", idcID).Order("name ASC").Find(&list).Error
	return list, err
}

// GetAll 获取所有机柜
func (d *CmdbCabinetDao) GetAll() ([]model.CmdbCabinet, error) {
	var list []model.CmdbCabinet
	err := d.db.Preload("IDC").Order("name ASC").Find(&list).Error
	return list, err
}
