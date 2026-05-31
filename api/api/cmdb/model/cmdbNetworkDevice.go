package model

import "time"

// CmdbNetworkDevice 网络设备表
type CmdbNetworkDevice struct {
	ID           uint         `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	SN           string       `gorm:"size:128;uniqueIndex;comment:'序列号/SN'" json:"sn"`
	Name         string       `gorm:"size:100;not null;comment:'设备名称'" json:"name"`
	DeviceType   int          `gorm:"not null;comment:'设备类型:1-路由器,2-交换机,3-防火墙,4-负载均衡,5-其他'" json:"deviceType"`
	Brand        string       `gorm:"size:50;comment:'品牌(Cisco/Huawei/H3C等)'" json:"brand"`
	Model        string       `gorm:"size:100;comment:'型号'" json:"model"`
	ManageIP     string       `gorm:"size:64;comment:'管理IP'" json:"manageIp"`
	Version      string       `gorm:"size:100;comment:'固件/系统版本'" json:"version"`
	PortNum      int          `gorm:"default:24;comment:'端口数量'" json:"portNum"`
	IDCID        uint         `gorm:"column:idc_id;not null;comment:'所属机房ID'" json:"idcId"`
	IDC          *CmdbIDC     `gorm:"foreignKey:IDCID" json:"idc"`
	CabinetID    uint         `gorm:"comment:'所属机柜ID'" json:"cabinetId"`
	Cabinet      *CmdbCabinet `gorm:"foreignKey:CabinetID" json:"cabinet"`
	AssetStatus  int          `gorm:"default:1;comment:'资产状态:1-在库,2-已上架,3-维修中,4-已下架,5-报废'" json:"assetStatus"`
	PurchaseDate string       `gorm:"size:20;comment:'采购日期'" json:"purchaseDate"`
	WarrantyDate string       `gorm:"size:20;comment:'维保到期'" json:"warrantyDate"`
	Status       int          `gorm:"default:1;comment:'运行状态:1-运行中,2-关机,3-离线'" json:"status"`
	Remark       string       `gorm:"type:text;comment:'备注'" json:"remark"`
	CreatedAt    time.Time    `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbNetworkDevice) TableName() string {
	return "cmdb_network_device"
}

// CmdbNetworkDeviceQuery 网络设备查询参数
type CmdbNetworkDeviceQuery struct {
	Name       string `json:"name"`
	DeviceType int    `json:"deviceType"`
	ManageIP   string `json:"manageIp"`
	IDCID      uint   `json:"idcId"`
	Brand      string `json:"brand"`
	Keyword    string `json:"keyword"`
	Page       int    `json:"page"`
	Size       int    `json:"size"`
}
