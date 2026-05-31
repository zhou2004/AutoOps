package dao

import (
	"dodevops-api/api/cmdb/model"

	"gorm.io/gorm"
)

type CmdbIDCDao struct {
	db *gorm.DB
}

func NewCmdbIDCDao(db *gorm.DB) *CmdbIDCDao {
	return &CmdbIDCDao{db: db}
}

// Create 创建机房
func (d *CmdbIDCDao) Create(data *model.CmdbIDC) error {
	return d.db.Create(data).Error
}

// Update 更新机房
func (d *CmdbIDCDao) Update(data *model.CmdbIDC) error {
	return d.db.Save(data).Error
}

// Delete 删除机房
func (d *CmdbIDCDao) Delete(id uint) error {
	return d.db.Delete(&model.CmdbIDC{}, id).Error
}

// GetByID 获取机房
func (d *CmdbIDCDao) GetByID(id uint) (*model.CmdbIDC, error) {
	var data model.CmdbIDC
	err := d.db.First(&data, id).Error
	return &data, err
}

// GetAll 获取所有机房
func (d *CmdbIDCDao) GetAll() ([]model.CmdbIDC, error) {
	var list []model.CmdbIDC
	err := d.db.Order("name ASC").Find(&list).Error
	return list, err
}

// GetList 分页查询机房
func (d *CmdbIDCDao) GetList(query model.CmdbIDCQuery) ([]model.CmdbIDC, int64, error) {
	var list []model.CmdbIDC
	var total int64
	db := d.db.Model(&model.CmdbIDC{})

	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status > 0 {
		db = db.Where("status = ?", query.Status)
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
