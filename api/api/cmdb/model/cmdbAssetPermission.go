package model

import "time"

// CmdbAssetPermission 资产授权表 — 类似于 JumpServer 的资产授权规则
type CmdbAssetPermission struct {
	ID          uint   `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name        string `gorm:"size:100;not null;uniqueIndex;comment:'授权规则名称'" json:"name"`
	Description string `gorm:"type:text;comment:'描述'" json:"description"`

	// 授权用户/用户组
	UserIDs  string `gorm:"type:text;comment:'授权用户ID列表(JSON数组)'" json:"userIds"`
	GroupIDs string `gorm:"type:text;comment:'授权用户组ID列表(JSON数组)'" json:"groupIds"`

	// 授权资产范围 (按模型类型)
	AssetTypes string `gorm:"type:text;comment:'授权资产类型(JSON数组: host/physical/network/database)'" json:"assetTypes"`

	// 授权资产(按分组)
	HostGroupIDs string `gorm:"type:text;comment:'授权主机分组ID列表(JSON数组)'" json:"hostGroupIds"`
	HostIDs      string `gorm:"type:text;comment:'授权具体主机ID列表(JSON数组)'" json:"hostIds"`
	PhysicalIDs  string `gorm:"type:text;comment:'授权物理机ID列表(JSON数组)'" json:"physicalIds"`
	NetworkIDs   string `gorm:"type:text;comment:'授权网络设备ID列表(JSON数组)'" json:"networkIds"`
	DatabaseIDs  string `gorm:"type:text;comment:'授权数据库ID列表(JSON数组)'" json:"databaseIds"`
	IDCIDs       string `gorm:"column:idc_ids;type:text;comment:'授权机房ID列表(JSON数组,含其下所有资产)'" json:"idcIds"`

	// 权限级别 (RBAC风格)
	PermissionActions string `gorm:"type:text;comment:'权限操作(JSON数组: get/list/connect/create/update/delete/admin)'" json:"permissionActions"`

	// 有效期
	IsActive    int    `gorm:"default:1;comment:'是否启用:0-禁用,1-启用'" json:"isActive"`
	DateStart   string `gorm:"size:20;comment:'有效期开始(YYYY-MM-DD)'" json:"dateStart"`
	DateExpired string `gorm:"size:20;comment:'有效期结束(YYYY-MM-DD)'" json:"dateExpired"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbAssetPermission) TableName() string {
	return "cmdb_asset_permission"
}

// CmdbAssetPermissionQuery 授权查询参数
type CmdbAssetPermissionQuery struct {
	Name        string `json:"name"`
	IsActive    int    `json:"isActive"`
	AssetTypes  string `json:"assetTypes"`
	SubjectType string `json:"subjectType" form:"subjectType"` // user/group
	SubjectID   uint   `json:"subjectId" form:"subjectId"`     // 用户ID或用户组ID
	Page        int    `json:"page"`
	Size        int    `json:"size"`
}

// CmdbUserAsset 用户已授权资产 — 用于快速查询
type CmdbUserAsset struct {
	UserID              uint     `json:"userId"`
	AllowedHostGroupIDs []uint   `json:"allowedHostGroupIds"` // 允许的主机组ID
	AllowedHostIDs      []uint   `json:"allowedHostIds"`      // 允许的主机ID
	AllowedPhysicalIDs  []uint   `json:"allowedPhysicalIds"`  // 允许的物理机ID
	AllowedNetworkIDs   []uint   `json:"allowedNetworkIds"`   // 允许的网络设备ID
	AllowedDatabaseIDs  []uint   `json:"allowedDatabaseIds"`  // 允许的数据库ID
	PermissionActions   []string `json:"permissionActions"`   // 允许的操作
}
