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
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := validateCategory(tx, "api_endpoint", m.CategoryID); err != nil {
			return err
		}
		return tx.Create(m).Error
	})
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
	if req.CategoryID != nil {
		query = query.Where("category_id = ?", *req.CategoryID)
	}
	if req.Name != "" {
		query = query.Where("(name LIKE ? OR url LIKE ?)", "%"+req.Name+"%", "%"+req.Name+"%")
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
	if req.PageSize > 100 {
		req.PageSize = 100
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
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := validateCategory(tx, "api_endpoint", m.CategoryID); err != nil {
			return err
		}
		return tx.Model(m).Select("name", "url", "method", "headers", "body", "check_interval", "timeout", "expected_code", "expected_body", "category_id").Updates(m).Error
	})
}

func (d *APIEndpointDao) Delete(id uint) error {
	return d.db.Delete(&model.MonitorAPIEndpoint{}, id).Error
}

func (d *APIEndpointDao) BatchDelete(ids []uint) error {
	return d.db.Delete(&model.MonitorAPIEndpoint{}, ids).Error
}
func (d *APIEndpointDao) UpdateCheck(m *model.MonitorAPIEndpoint) error {
	return d.db.Model(m).Select("last_status_code", "last_response_time", "status", "check_time", "error_msg").Updates(m).Error
}
