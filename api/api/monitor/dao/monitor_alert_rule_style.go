package dao

import (
"dodevops-api/api/monitor/model"
"dodevops-api/common"

"gorm.io/gorm"
)

type MonitorAlertRuleStyleDao interface {
Create(data *model.MonitorAlertRuleStyle) error
Delete(id uint) error
Update(data *model.MonitorAlertRuleStyle) error
GetList() ([]*model.MonitorAlertRuleStyle, error)
}

type monitorAlertRuleStyleDao struct {
db *gorm.DB
}

func NewMonitorAlertRuleStyleDao() MonitorAlertRuleStyleDao {
return &monitorAlertRuleStyleDao{db: common.GetDB()}
}

func (d *monitorAlertRuleStyleDao) Create(data *model.MonitorAlertRuleStyle) error {
return d.db.Create(data).Error
}

func (d *monitorAlertRuleStyleDao) Delete(id uint) error {
return d.db.Delete(&model.MonitorAlertRuleStyle{}, id).Error
}

func (d *monitorAlertRuleStyleDao) Update(data *model.MonitorAlertRuleStyle) error {
return d.db.Model(&model.MonitorAlertRuleStyle{}).Where("id = ?", data.ID).Updates(data).Error
}

func (d *monitorAlertRuleStyleDao) GetList() ([]*model.MonitorAlertRuleStyle, error) {
var list []*model.MonitorAlertRuleStyle
err := d.db.Find(&list).Error
return list, err
}
