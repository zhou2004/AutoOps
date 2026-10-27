package model

import "time"

// One row per role, including an explicit empty grant. Absence uses legacy menu grants.
type RolePermission struct {
	RoleID      uint      `gorm:"primaryKey;autoIncrement:false" json:"roleId"`
	Permissions string    `gorm:"type:text;not null" json:"-"`
	UpdatedBy   uint      `json:"updatedBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (RolePermission) TableName() string { return "sys_role_permission" }
