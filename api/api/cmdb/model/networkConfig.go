package model

import "time"

// CmdbNetworkConfigSnapshot stores immutable, encrypted configuration versions.
// ContentCipher is never serialized; viewing a snapshot returns redacted text.
type CmdbNetworkConfigSnapshot struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	NetworkID     uint      `gorm:"not null;index" json:"networkId"`
	ConnectionID  uint      `json:"connectionId"`
	Platform      string    `gorm:"size:32;not null" json:"platform"`
	Checksum      string    `gorm:"size:64;not null" json:"checksum"`
	ByteSize      int       `json:"byteSize"`
	ContentCipher string    `gorm:"type:longtext;not null" json:"-"`
	Source        string    `gorm:"size:32" json:"source"`
	Remark        string    `gorm:"size:500" json:"remark"`
	Restorable    bool      `json:"restorable"`
	CreatedBy     uint      `json:"createdBy"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index" json:"createdAt"`
}

func (CmdbNetworkConfigSnapshot) TableName() string { return "cmdb_network_config_snapshot" }

// CmdbNetworkConfigAudit records attempts before opening a remote connection.
type CmdbNetworkConfigAudit struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	NetworkID    uint      `gorm:"not null;index" json:"networkId"`
	ConnectionID uint      `json:"connectionId"`
	SnapshotID   uint      `json:"snapshotId"`
	Action       string    `gorm:"size:32;not null" json:"action"`
	Status       string    `gorm:"size:32;not null" json:"status"`
	Message      string    `gorm:"size:1000" json:"message"`
	CreatedBy    uint      `json:"createdBy"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CmdbNetworkConfigAudit) TableName() string { return "cmdb_network_config_audit" }
