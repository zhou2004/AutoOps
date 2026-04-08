package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type MonitorAlertRuleDao interface {
	Create(data *model.MonitorAlertRule) error
	Delete(id uint) error
	Update(data *model.MonitorAlertRule) error
	GetByID(id uint) (*model.MonitorAlertRule, error)
	GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error)
}

type monitorAlertRuleDao struct {
	db *gorm.DB
}

func NewMonitorAlertRuleDao() MonitorAlertRuleDao {
	return &monitorAlertRuleDao{
		db: common.GetDB(),
	}
}

func (d *monitorAlertRuleDao) Create(data *model.MonitorAlertRule) error {
	return d.db.Create(data).Error
}

func (d *monitorAlertRuleDao) Delete(id uint) error {
	return d.db.Delete(&model.MonitorAlertRule{}, id).Error
}

func (d *monitorAlertRuleDao) Update(data *model.MonitorAlertRule) error {
	return d.db.Model(&model.MonitorAlertRule{}).Where("id = ?", data.ID).Updates(data).Error
}

func (d *monitorAlertRuleDao) GetByID(id uint) (*model.MonitorAlertRule, error) {
	var data model.MonitorAlertRule
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *monitorAlertRuleDao) GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error) {
	var list []*model.MonitorAlertRule
	var total int64

	query := d.db.Model(&model.MonitorAlertRule{})
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
