package model

import "time"

type K8sRule struct {
	APIGroups []string `json:"apiGroups"`
	Resources []string `json:"resources"`
	Verbs     []string `json:"verbs"`
}

type K8sRbacRole struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClusterID uint      `gorm:"not null" json:"clusterId"`
	Namespace string    `gorm:"size:255;default:''" json:"namespace"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Rules     string    `gorm:"type:json;not null" json:"rules"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (K8sRbacRole) TableName() string {
	return "k8s_rbac_role"
}

type K8sRbacBinding struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ClusterID   uint      `gorm:"not null" json:"clusterId"`
	Namespace   string    `gorm:"size:255;default:''" json:"namespace"`
	RoleID      uint      `gorm:"not null" json:"roleId"`
	SubjectType string    `gorm:"size:32;not null" json:"subjectType"`
	SubjectID   uint      `gorm:"not null" json:"subjectId"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (K8sRbacBinding) TableName() string {
	return "k8s_rbac_binding"
}

// Request and Vo structures
type CreateRbacRoleReq struct {
	ClusterID uint      `json:"clusterId" binding:"required"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name" binding:"required"`
	Rules     []K8sRule `json:"rules" binding:"required"`
}

type CreateRbacBindingReq struct {
	ClusterID   uint   `json:"clusterId" binding:"required"`
	Namespace   string `json:"namespace"`
	RoleID      uint   `json:"roleId" binding:"required"`
	SubjectType string `json:"subjectType" binding:"required,oneof=User Group"`
	SubjectID   uint   `json:"subjectId" binding:"required"`
}

type K8sRbacRoleVo struct {
	ID          uint      `json:"id"`
	ClusterID   uint      `json:"clusterId"`
	ClusterName string    `json:"clusterName"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	RulesStr    string    `gorm:"column:rules" json:"-"` // raw JSON from DB
	Rules       []K8sRule `json:"rules"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type K8sRbacBindingVo struct {
	ID          uint      `json:"id"`
	ClusterID   uint      `json:"clusterId"`
	ClusterName string    `json:"clusterName"`
	Namespace   string    `json:"namespace"`
	RoleID      uint      `json:"roleId"`
	RoleName    string    `json:"roleName"`
	SubjectType string    `json:"subjectType"`
	SubjectID   uint      `json:"subjectId"`
	SubjectName string    `json:"subjectName"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
