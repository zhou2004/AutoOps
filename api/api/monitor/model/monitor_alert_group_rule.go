package model

import "gorm.io/gorm"

type MonitorAlertGroupRule struct {
	gorm.Model
	DataSourceID uint   `json:"data_source_id" gorm:"column:data_source_id;comment:数据源id"`
	GroupName    string `json:"group_name" gorm:"column:group_name;type:varchar(255);comment:规则组名"` // 群组名
	RuleContent  string `json:"rule_content" gorm:"column:rule_content;type:text;comment:原生yaml内容"` // 整个Group的完整Yaml
	Labels       string `json:"labels" gorm:"column:labels;type:text;comment:该组的全局label (JSON格式)"`  // 群组级别的Labels,会下发至Rule
}

func (MonitorAlertGroupRule) TableName() string {
	return "monitor_alert_group_rule"
}
