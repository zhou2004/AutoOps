package model

import "time"

// 分类按监控类型隔离，0 表示未分类。
type MonitorCategory struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Scope       string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_monitor_category_scope_name" json:"scope"`
	Name        string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_monitor_category_scope_name" json:"name"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	CreatedAt   time.Time `gorm:"column:create_time" json:"createTime"`
	UpdatedAt   time.Time `gorm:"column:update_time" json:"updateTime"`
}

func (MonitorCategory) TableName() string { return "monitor_category" }

type MonitorCategoryReq struct {
	ID          uint   `json:"id"`
	Scope       string `json:"scope" form:"scope" binding:"required,oneof=domain_cert api_endpoint"`
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"max=255"`
}
type MonitorCategoryAssignReq struct {
	Scope      string `json:"scope" binding:"required,oneof=domain_cert api_endpoint"`
	IDs        []uint `json:"ids" binding:"required,min=1,max=1000,dive,gt=0"`
	CategoryID uint   `json:"categoryId"`
}
