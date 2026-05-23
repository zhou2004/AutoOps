package model

import (
	"time"
)

// K8sPermission K8s权限表
type K8sPermission struct {
	ID             uint      `gorm:"primaryKey;comment:'主键ID'" json:"id"`
	UserID         uint      `gorm:"not null;index:idx_user_cluster_ns,unique;comment:'用户ID(sys_admin.id)'" json:"userId"`
	ClusterID      uint      `gorm:"not null;index:idx_user_cluster_ns,unique;comment:'集群ID(k8s_cluster.id)'" json:"clusterId"`
	Namespace      string    `gorm:"size:255;not null;index:idx_user_cluster_ns,unique;comment:'命名空间名称'" json:"namespace"`
	PermissionType string    `gorm:"size:64;default:'readonly';comment:'权限类型: readonly/write/admin'" json:"permissionType"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (K8sPermission) TableName() string {
	return "k8s_permission"
}

// CreateK8sPermissionRequest 创建K8s权限请求
type CreateK8sPermissionRequest struct {
	UserID         uint   `json:"userId" binding:"required"`
	ClusterID      uint   `json:"clusterId" binding:"required"`
	Namespace      string `json:"namespace" binding:"required"`
	PermissionType string `json:"permissionType" binding:"required,oneof=readonly write admin"`
}

// UpdateK8sPermissionRequest 更新K8s权限请求
type UpdateK8sPermissionRequest struct {
	PermissionType string `json:"permissionType" binding:"required,oneof=readonly write admin"`
}

// K8sPermissionListResponse 权限列表响应
type K8sPermissionListResponse struct {
	List  []K8sPermissionVo `json:"list"`
	Total int64             `json:"total"`
}

// K8sPermissionVo 权限视图
type K8sPermissionVo struct {
	ID             uint      `json:"id"`
	UserID         uint      `json:"userId"`
	Username       string    `json:"username"`
	Nickname       string    `json:"nickname"`
	ClusterID      uint      `json:"clusterId"`
	ClusterName    string    `json:"clusterName"`
	Namespace      string    `json:"namespace"`
	PermissionType string    `json:"permissionType"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// K8sPermissionQuery 权限查询参数
type K8sPermissionQuery struct {
	UserID    uint   `form:"userId"`
	ClusterID uint   `form:"clusterId"`
	Namespace string `form:"namespace"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

// K8sPermissionBatchCreateRequest 批量创建权限请求
type K8sPermissionBatchCreateRequest struct {
	UserID         uint     `json:"userId" binding:"required"`
	ClusterID      uint     `json:"clusterId" binding:"required"`
	Namespaces     []string `json:"namespaces" binding:"required"`
	PermissionType string   `json:"permissionType" binding:"required,oneof=readonly write admin"`
}

// K8sPermissionDeleteRequest 删除权限请求
type K8sPermissionDeleteRequest struct {
	ID uint `json:"id" binding:"required"`
}
