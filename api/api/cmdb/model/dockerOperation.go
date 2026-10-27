package model

import "time"

type CmdbDockerAudit struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AssetType    string     `gorm:"size:16;index:idx_docker_audit_asset;not null" json:"assetType"`
	AssetID      uint       `gorm:"index:idx_docker_audit_asset;not null" json:"assetId"`
	ConnectionID uint       `json:"connectionId"`
	UserID       uint       `gorm:"index;not null" json:"userId"`
	Action       string     `gorm:"size:32;not null" json:"action"`
	Target       string     `gorm:"size:255" json:"target"`
	Status       string     `gorm:"size:16;not null" json:"status"`
	Message      string     `gorm:"size:500" json:"message"`
	ClientIP     string     `gorm:"size:64" json:"clientIp"`
	DurationMS   int64      `json:"durationMs"`
	CreatedAt    time.Time  `gorm:"index" json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

func (CmdbDockerAudit) TableName() string { return "cmdb_docker_audit" }
