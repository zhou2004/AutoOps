package model

import "time"

// DatabaseDeployment is a managed Ansible installation, independent of SQL sessions.
type DatabaseDeployment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Engine    string    `gorm:"size:20;not null" json:"engine"`
	Mode      string    `gorm:"size:20;not null" json:"mode"`
	GroupID   uint      `json:"groupId"`
	Nodes     string    `gorm:"type:text;not null" json:"nodes"`
	Managers  string    `gorm:"type:text" json:"managers"`
	Port      int       `json:"port"`
	Version   string    `gorm:"size:32" json:"version"`
	Status    string    `gorm:"size:24" json:"status"`
	Secrets   string    `gorm:"type:text" json:"-"`
	CreatedBy uint      `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (DatabaseDeployment) TableName() string { return "cmdb_database_deployment" }

type DatabaseAutomationRun struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	DeploymentID uint       `gorm:"index;not null" json:"deploymentId"`
	Action       string     `gorm:"size:32" json:"action"`
	Status       string     `gorm:"size:24" json:"status"`
	UserID       uint       `json:"userId"`
	Log          string     `gorm:"type:longtext" json:"log"`
	Artifact     string     `gorm:"size:255" json:"artifact"`
	CreatedAt    time.Time  `json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

func (DatabaseAutomationRun) TableName() string { return "cmdb_database_automation_run" }

// Unique deployment lock also serializes actions across multiple API replicas.
type DatabaseAutomationLock struct {
	Key       string `gorm:"primaryKey;size:100"`
	RunID     uint
	CreatedAt time.Time
}

func (DatabaseAutomationLock) TableName() string { return "cmdb_database_automation_lock" }
