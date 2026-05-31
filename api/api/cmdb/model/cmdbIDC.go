package model

import "time"

// CmdbIDC 机房表
type CmdbIDC struct {
	ID          uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex;comment:'机房名称'" json:"name"`
	ShortName   string    `gorm:"size:50;comment:'机房简称'" json:"shortName"`
	Address     string    `gorm:"size:255;comment:'机房地址'" json:"address"`
	Contact     string    `gorm:"size:50;comment:'联系人'" json:"contact"`
	Phone       string    `gorm:"size:30;comment:'联系电话'" json:"phone"`
	Level       string    `gorm:"size:20;comment:'机房等级(T1-T4)'" json:"level"`
	Status      int       `gorm:"default:1;comment:'状态:1-启用,2-停用'" json:"status"`
	Description string    `gorm:"type:text;comment:'描述'" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbIDC) TableName() string {
	return "cmdb_idc"
}

// CmdbIDCQuery 机房查询参数
type CmdbIDCQuery struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}
