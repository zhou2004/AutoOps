package model

import "gorm.io/gorm"

// MonitorAlertRuleStyle 规则分类模型定义
type MonitorAlertRuleStyle struct {
	gorm.Model
	Name        string `json:"name" gorm:"column:name;type:varchar(64);unique;not null;comment:分类名称(如CPU,Memory)"`
	Description string `json:"description" gorm:"column:description;type:varchar(255);comment:描述"`
}

func (MonitorAlertRuleStyle) TableName() string {
	return "monitor_alert_rule_style"
}
