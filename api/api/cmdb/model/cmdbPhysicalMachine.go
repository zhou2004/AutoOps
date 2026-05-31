package model

import "time"

// CmdbPhysicalMachine 物理机表
type CmdbPhysicalMachine struct {
	ID           uint         `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	SN           string       `gorm:"size:128;uniqueIndex;comment:'序列号/SN'" json:"sn"`
	HostName     string       `gorm:"size:100;comment:'主机名'" json:"hostName"`
	ManageIP     string       `gorm:"size:64;comment:'管理IP(BMC/iLO/iDRAC)'" json:"manageIp"`
	BusinessIP   string       `gorm:"size:64;comment:'业务IP'" json:"businessIp"`
	Brand        string       `gorm:"size:50;comment:'品牌(Dell/HP/Inspur等)'" json:"brand"`
	Model        string       `gorm:"size:100;comment:'型号(R750/DL380等)'" json:"model"`
	CPU          string       `gorm:"size:100;comment:'CPU信息'" json:"cpu"`
	Memory       string       `gorm:"size:100;comment:'内存信息(GB)'" json:"memory"`
	Disk         string       `gorm:"size:255;comment:'磁盘信息'" json:"disk"`
	Raid         string       `gorm:"size:100;comment:'RAID类型'" json:"raid"`
	IDCID        uint         `gorm:"column:idc_id;not null;comment:'所属机房ID'" json:"idcId"`
	IDC          *CmdbIDC     `gorm:"foreignKey:IDCID" json:"idc"`
	CabinetID    uint         `gorm:"comment:'所属机柜ID'" json:"cabinetId"`
	Cabinet      *CmdbCabinet `gorm:"foreignKey:CabinetID" json:"cabinet"`
	UnitPosition int          `gorm:"default:0;comment:'机柜U位(起始)'" json:"unitPosition"`
	AssetStatus  int          `gorm:"default:1;comment:'资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废'" json:"assetStatus"`
	PurchaseDate string       `gorm:"size:20;comment:'采购日期'" json:"purchaseDate"`
	WarrantyDate string       `gorm:"size:20;comment:'维保到期'" json:"warrantyDate"`
	Vendor       string       `gorm:"size:50;comment:'供应商'" json:"vendor"`
	Status       int          `gorm:"default:1;comment:'运行状态:1-运行中,2-关机,3-离线'" json:"status"`
	Remark       string       `gorm:"type:text;comment:'备注'" json:"remark"`
	CreatedAt    time.Time    `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbPhysicalMachine) TableName() string {
	return "cmdb_physical_machine"
}

// CmdbPhysicalMachineQuery 物理机查询参数
type CmdbPhysicalMachineQuery struct {
	SN          string `json:"sn"`
	HostName    string `json:"hostName"`
	ManageIP    string `json:"manageIp"`
	IDCID       uint   `json:"idcId"`
	CabinetID   uint   `json:"cabinetId"`
	AssetStatus int    `json:"assetStatus"`
	Brand       string `json:"brand"`
	Keyword     string `json:"keyword"`
	Page        int    `json:"page"`
	Size        int    `json:"size"`
}
