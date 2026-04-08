package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type MonitorDataSourceDao interface {
	Create(data *model.MonitorDataSource) error
	Delete(id uint) error
	Update(data *model.MonitorDataSource) error
	GetByID(id uint) (*model.MonitorDataSource, error)
	GetList(page, pageSize int) ([]*model.MonitorDataSource, int64, error)
}

type monitorDataSourceDao struct {
	db *gorm.DB
}

func NewMonitorDataSourceDao() MonitorDataSourceDao {
	return &monitorDataSourceDao{
		db: common.GetDB(),
	}
}

func (d *monitorDataSourceDao) Create(data *model.MonitorDataSource) error {
	return d.db.Create(data).Error
}

func (d *monitorDataSourceDao) Delete(id uint) error {
	return d.db.Delete(&model.MonitorDataSource{}, id).Error
}

func (d *monitorDataSourceDao) Update(data *model.MonitorDataSource) error {
	// 使用 Updates 可以更新非零值字段，如果包含零值请使用 Select()
	return d.db.Model(&model.MonitorDataSource{}).Where("id = ?", data.ID).Updates(data).Error
}

func (d *monitorDataSourceDao) GetByID(id uint) (*model.MonitorDataSource, error) {
	var data model.MonitorDataSource
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *monitorDataSourceDao) GetList(page, pageSize int) ([]*model.MonitorDataSource, int64, error) {
	var list []*model.MonitorDataSource
	var total int64

	query := d.db.Model(&model.MonitorDataSource{})
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	err = query.Order("id desc").Find(&list).Error
	return list, total, err
}
