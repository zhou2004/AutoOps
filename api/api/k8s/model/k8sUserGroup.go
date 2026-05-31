package model

import "time"

// K8sUserGroup K8s用户组表
type K8sUserGroup struct {
	ID          uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	Name        string    `gorm:"size:255;not null;uniqueIndex;comment:'用户组名称'" json:"name"`
	Code        string    `gorm:"size:128;default:'';index;comment:'用户组编码'" json:"code"`
	Description string    `gorm:"size:512;default:'';comment:'描述'" json:"description"`
	Status      int       `gorm:"default:1;comment:'状态:1-启用,0-禁用'" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (K8sUserGroup) TableName() string {
	return "k8s_user_group"
}

// K8sUserGroupMember 用户组成员关系
type K8sUserGroupMember struct {
	ID        uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	GroupID   uint      `gorm:"not null;uniqueIndex:idx_group_user;comment:'用户组ID'" json:"groupId"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_group_user;comment:'用户ID'" json:"userId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (K8sUserGroupMember) TableName() string {
	return "k8s_user_group_member"
}

// ---------- Request/Response ----------

// CreateUserGroupRequest 创建用户组请求
type CreateUserGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// UpdateUserGroupRequest 更新用户组请求
type UpdateUserGroupRequest struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Status      *int   `json:"status"`
}

// AddGroupMemberRequest 添加组成员请求
type AddGroupMemberRequest struct {
	GroupID uint   `json:"groupId" binding:"required"`
	UserIDs []uint `json:"userIds" binding:"required"`
}

// RemoveGroupMemberRequest 移除组成员请求
type RemoveGroupMemberRequest struct {
	GroupID uint `json:"groupId" binding:"required"`
	UserID  uint `json:"userId" binding:"required"`
}

// UserGroupListResponse 用户组列表响应
type UserGroupListResponse struct {
	List  []UserGroupVo `json:"list"`
	Total int64         `json:"total"`
}

// UserGroupVo 用户组视图
type UserGroupVo struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	MemberCount int       `json:"memberCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UserGroupQuery 用户组查询参数
type UserGroupQuery struct {
	Name   string `form:"name"`
	Code   string `form:"code"`
	Page   int    `form:"page"`
	Size   int    `form:"size"`
	Status int    `form:"status"`
}

// GroupMemberVo 组成员视图
type GroupMemberVo struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"createdAt"`
}