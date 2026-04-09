package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type MonitorAlertGroupRuleDao interface {
	Create(data *model.MonitorAlertGroupRule) error
	Delete(id uint) error
	Update(data *model.MonitorAlertGroupRule) error
	GetByID(id uint) (*model.MonitorAlertGroupRule, error)
	GetList(page, pageSize int) ([]*model.MonitorAlertGroupRule, int64, error)
	GetAll() ([]*model.MonitorAlertGroupRule, error)
}

type monitorAlertGroupRuleDao struct {
	db *gorm.DB
}

func NewMonitorAlertGroupRuleDao() MonitorAlertGroupRuleDao {
	return &monitorAlertGroupRuleDao{
		db: common.GetDB(),
	}
}

func (d *monitorAlertGroupRuleDao) Create(data *model.MonitorAlertGroupRule) error {
	return d.db.Create(data).Error
}

func (d *monitorAlertGroupRuleDao) Delete(id uint) error {
	return d.db.Unscoped().Delete(&model.MonitorAlertGroupRule{}, id).Error
}

func (d *monitorAlertGroupRuleDao) Update(data *model.MonitorAlertGroupRule) error {
	return d.db.Model(&model.MonitorAlertGroupRule{}).Where("id = ?", data.ID).Updates(data).Error
}

func (d *monitorAlertGroupRuleDao) GetByID(id uint) (*model.MonitorAlertGroupRule, error) {
	var data model.MonitorAlertGroupRule
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *monitorAlertGroupRuleDao) GetList(page, pageSize int) ([]*model.MonitorAlertGroupRule, int64, error) {
	var list []*model.MonitorAlertGroupRule
	var total int64
	query := d.db.Model(&model.MonitorAlertGroupRule{})
	query.Count(&total)
	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, total, err
}

func (d *monitorAlertGroupRuleDao) GetAll() ([]*model.MonitorAlertGroupRule, error) {
	var list []*model.MonitorAlertGroupRule
	err := d.db.Find(&list).Error
	return list, err
}
