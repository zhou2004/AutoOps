package model

import "time"

// CmdbCabinet 机柜表
type CmdbCabinet struct {
	ID        uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex;comment:'机柜名称/编号'" json:"name"`
	IDCID     uint      `gorm:"column:idc_id;not null;comment:'所属机房ID'" json:"idcId"`
	IDC       CmdbIDC   `gorm:"foreignKey:IDCID" json:"idc"`
	Position  string    `gorm:"size:100;comment:'位置描述(如A列3排)'" json:"position"`
	UnitNum   int       `gorm:"default:42;comment:'机柜U数(如42U)'" json:"unitNum"`
	UsedUnit  int       `gorm:"default:0;comment:'已用U位数'" json:"usedUnit"`
	PowerKW   float64   `gorm:"default:0;comment:'额定功率(KW)'" json:"powerKw"`
	Status    int       `gorm:"default:1;comment:'状态:1-启用,2-停用'" json:"status"`
	Remark    string    `gorm:"type:text;comment:'备注'" json:"remark"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}


func (CmdbCabinet) TableName() string {
	return "cmdb_cabinet"
}

// CmdbCabinetQuery 机柜查询参数
type CmdbCabinetQuery struct {
	Name   string `json:"name"`
	IDCID  uint   `json:"idcId"`
	Status int    `json:"status"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}
