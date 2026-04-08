package model

import "time"

// MonitorAlertRule 监控告警规则表
type MonitorAlertRule struct {
	ID           uint      `gorm:"primaryKey;column:id;autoIncrement;comment:主键ID" json:"id"`
	DataSourceID uint      `gorm:"column:data_source_id;type:bigint unsigned;not null;comment:关联数据源ID" json:"dataSourceId"`
	Name         string    `gorm:"column:name;type:varchar(128);not null;comment:告警名称" json:"name"`
	Labels	   	 string    `gorm:"column:labels;type:text;comment:告警标签，JSON格式字符串" json:"labels"`
	RuleContent  string    `gorm:"column:rule_content;type:text;not null;comment:规则内容(YAML或JSON等)" json:"ruleContent"`
	Status       string    `gorm:"column:status;type:varchar(32);default:'inactive';comment:告警状态: inactive, pending, firing" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP;comment:创建时间" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updatedAt"`
}

// TableName 表名
func (MonitorAlertRule) TableName() string {
	return "monitor_alert_rule"
}
