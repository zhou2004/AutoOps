package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type APIEndpointDao struct {
	db *gorm.DB
}

func NewAPIEndpointDao() *APIEndpointDao {
	return &APIEndpointDao{
		db: common.GetDB(),
	}
}

func (d *APIEndpointDao) Create(m *model.MonitorAPIEndpoint) error {
	return d.db.Create(m).Error
}

func (d *APIEndpointDao) GetByID(id uint) (*model.MonitorAPIEndpoint, error) {
	var m model.MonitorAPIEndpoint
	err := d.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *APIEndpointDao) GetList(req *model.APIEndpointListReq) ([]model.MonitorAPIEndpoint, int64, error) {
	var list []model.MonitorAPIEndpoint
	var total int64
	query := d.db.Model(&model.MonitorAPIEndpoint{})
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status > 0 {
		query = query.Where("status = ?", req.Status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *APIEndpointDao) GetAll() ([]model.MonitorAPIEndpoint, error) {
	var list []model.MonitorAPIEndpoint
	err := d.db.Find(&list).Error
	return list, err
}

func (d *APIEndpointDao) Update(m *model.MonitorAPIEndpoint) error {
	return d.db.Save(m).Error
}

func (d *APIEndpointDao) Delete(id uint) error {
	return d.db.Delete(&model.MonitorAPIEndpoint{}, id).Error
}

func (d *APIEndpointDao) BatchDelete(ids []uint) error {
	return d.db.Delete(&model.MonitorAPIEndpoint{}, ids).Error
}