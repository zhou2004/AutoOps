package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type MonitorAlertRuleDao interface {
	Create(data *model.MonitorAlertRule) error
	Delete(id uint) error
	DeleteByGroupID(groupId uint) error
	Update(data *model.MonitorAlertRule) error
	GetByID(id uint) (*model.MonitorAlertRule, error)
	GetByGroupID(groupId uint) ([]*model.MonitorAlertRule, error)
	GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error)
	GetListByQuery(req *model.MonitorAlertRuleQuery) ([]*model.MonitorAlertRule, int64, error)
	UpdateStatus(id uint, status string) error
	GetAll() ([]*model.MonitorAlertRule, error)
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
	return d.db.Unscoped().Delete(&model.MonitorAlertRule{}, id).Error
}

func (d *monitorAlertRuleDao) DeleteByGroupID(groupId uint) error {
	return d.db.Unscoped().Where("group_id = ?", groupId).Delete(&model.MonitorAlertRule{}).Error
}

func (d *monitorAlertRuleDao) Update(data *model.MonitorAlertRule) error {
	return d.db.Model(&model.MonitorAlertRule{}).Where("id = ?", data.ID).Updates(data).Error
}

func (d *monitorAlertRuleDao) GetByID(id uint) (*model.MonitorAlertRule, error) {
	var data model.MonitorAlertRule
	err := d.db.First(&data, id).Error
	return &data, err
}

func (d *monitorAlertRuleDao) GetByGroupID(groupId uint) ([]*model.MonitorAlertRule, error) {
	var list []*model.MonitorAlertRule
	err := d.db.Where("group_id = ?", groupId).Find(&list).Error
	return list, err
}

func (d *monitorAlertRuleDao) GetList(page, pageSize int) ([]*model.MonitorAlertRule, int64, error) {
	var list []*model.MonitorAlertRule
	var total int64
	query := d.db.Model(&model.MonitorAlertRule{})
	query.Count(&total)
	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	err := query.Order("id desc").Find(&list).Error
	return list, total, err
}

func (d *monitorAlertRuleDao) UpdateStatus(id uint, status string) error {
	return d.db.Model(&model.MonitorAlertRule{}).Where("id = ?", id).Update("status", status).Error
}

func (d *monitorAlertRuleDao) GetListByQuery(req *model.MonitorAlertRuleQuery) ([]*model.MonitorAlertRule, int64, error) {
	var list []*model.MonitorAlertRule
	var total int64
	query := d.db.Model(&model.MonitorAlertRule{})

	if req.GroupID > 0 {
		query = query.Where("group_id = ?", req.GroupID)
	}
	if req.Alert != "" {
		query = query.Where("alert LIKE ?", "%"+req.Alert+"%")
	}
	if req.Expr != "" {
		query = query.Where("expr LIKE ?", "%"+req.Expr+"%")
	}
	if req.ForDuration != "" {
		query = query.Where("for_duration = ?", req.ForDuration)
	}
	if req.Labels != "" {
		query = query.Where("labels LIKE ?", "%"+req.Labels+"%")
	}
	if req.Constraints != "" {
		query = query.Where("constraints LIKE ?", "%"+req.Constraints+"%")
	}
	if req.Summary != "" {
		query = query.Where("summary LIKE ?", "%"+req.Summary+"%")
	}
	if req.Description != "" {
		query = query.Where("description LIKE ?", "%"+req.Description+"%")
	}
	if req.Style != "" {
		query = query.Where("style = ?", req.Style)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Severity != "" {
		query = query.Where("severity = ?", req.Severity)
	}
	if req.Enabled != nil {
		query = query.Where("enabled = ?", *req.Enabled)
	}

	query.Count(&total)

	if req.Page > 0 && req.PageSize > 0 {
		query = query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize)
	}

	err := query.Order("id desc").Find(&list).Error
	return list, total, err
}

func (d *monitorAlertRuleDao) GetAll() ([]*model.MonitorAlertRule, error) {
	var list []*model.MonitorAlertRule
	err := d.db.Find(&list).Error
	return list, err
}
