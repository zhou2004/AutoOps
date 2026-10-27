package model

import "time"

// CmdbDatabaseAudit records access without storing query values or credentials.
type CmdbDatabaseAudit struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	DatabaseID uint       `gorm:"index;not null" json:"databaseId"`
	UserID     uint       `gorm:"index;not null" json:"userId"`
	Action     string     `gorm:"size:32;not null" json:"action"`
	Target     string     `gorm:"size:255" json:"target"`
	Status     string     `gorm:"size:16;not null" json:"status"`
	Message    string     `gorm:"size:500" json:"message"`
	ClientIP   string     `gorm:"size:64" json:"clientIp"`
	DurationMS int64      `json:"durationMs"`
	CreatedAt  time.Time  `gorm:"index" json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

func (CmdbDatabaseAudit) TableName() string { return "cmdb_database_audit" }
