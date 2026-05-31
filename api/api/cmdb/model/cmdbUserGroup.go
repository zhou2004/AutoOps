package model

import "time"

// CmdbUserGroup CMDB用户组表（独立于K8s的用户组）
type CmdbUserGroup struct {
	ID          uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name        string    `gorm:"size:255;not null;uniqueIndex;comment:'用户组名称'" json:"name"`
	Code        string    `gorm:"size:128;default:'';index;comment:'用户组编码'" json:"code"`
	Description string    `gorm:"size:512;default:'';comment:'描述'" json:"description"`
	Status      int       `gorm:"default:1;comment:'状态:1-启用,0-禁用'" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbUserGroup) TableName() string {
	return "cmdb_user_group"
}

// CmdbUserGroupMember CMDB用户组成员关系
type CmdbUserGroupMember struct {
	ID        uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	GroupID   uint      `gorm:"not null;uniqueIndex:idx_cmdb_ugm_group_user;comment:'用户组ID'" json:"groupId"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_cmdb_ugm_group_user;comment:'用户ID'" json:"userId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (CmdbUserGroupMember) TableName() string {
	return "cmdb_user_group_member"
}

// ---------- Request/Response ----------

type CmdbCreateUserGroupReq struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

type CmdbUpdateUserGroupReq struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      *int   `json:"status"`
}

type CmdbAddGroupMemberReq struct {
	GroupID uint   `json:"groupId" binding:"required"`
	UserIDs []uint `json:"userIds" binding:"required"`
}

type CmdbRemoveGroupMemberReq struct {
	GroupID uint `json:"groupId" binding:"required"`
	UserID  uint `json:"userId" binding:"required"`
}

type CmdbUserGroupQuery struct {
	Name string `json:"name" form:"name"`
	Code string `json:"code" form:"code"`
	Page int    `json:"page" form:"page"`
	Size int    `json:"size" form:"size"`
}

type CmdbUserGroupVo struct {
	CmdbUserGroup
	MemberCount int64 `json:"memberCount"`
}

type CmdbGroupMemberVo struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
}

// CmdbCredentialPermission 凭据授权表 — 控制用户/用户组对SSH凭据的使用权限
type CmdbCredentialPermission struct {
	ID           uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name         string    `gorm:"size:100;not null;comment:'授权规则名称'" json:"name"`
	CredentialID uint      `gorm:"not null;comment:'凭据ID'" json:"credentialId"`
	UserIDs      string    `gorm:"type:text;comment:'授权用户ID列表(JSON数组)'" json:"userIds"`
	GroupIDs     string    `gorm:"type:text;comment:'授权用户组ID列表(JSON数组)'" json:"groupIds"`
	IsActive     int       `gorm:"default:1;comment:'是否启用:0-禁用,1-启用'" json:"isActive"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbCredentialPermission) TableName() string {
	return "cmdb_credential_permission"
}

type CmdbCredentialPermissionQuery struct {
	Name         string `json:"name" form:"name"`
	CredentialID uint   `json:"credentialId" form:"credentialId"`
	IsActive     int    `json:"isActive" form:"isActive"`
	Page         int    `json:"page" form:"page"`
	Size         int    `json:"size" form:"size"`
}

type CmdbCredentialPermissionReq struct {
	Name         string `json:"name" binding:"required"`
	CredentialID uint   `json:"credentialId" binding:"required"`
	UserIDs      []uint `json:"userIds"`
	GroupIDs     []uint `json:"groupIds"`
	IsActive     int    `json:"isActive"`
}
