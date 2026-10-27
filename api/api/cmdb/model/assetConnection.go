package model

import "time"

// CmdbAssetConnection keeps connection endpoints independent of asset inventory
// and of each other. Credentials are referenced, never returned in this model.
type CmdbAssetConnection struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AssetType    string    `gorm:"size:16;not null;index:idx_asset_connection" json:"assetType"`
	AssetID      uint      `gorm:"not null;index:idx_asset_connection" json:"assetId"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Protocol     string    `gorm:"size:8;not null" json:"protocol"`
	Address      string    `gorm:"size:253;not null" json:"address"`
	Port         int       `json:"port"`
	Username     string    `gorm:"size:128" json:"username"`
	CredentialID uint      `json:"credentialId"`
	Domain       string    `gorm:"size:128" json:"domain"`
	Security     string    `gorm:"size:8" json:"security"`
	IgnoreCert   bool      `json:"ignoreCert"`
	ReadOnly     bool      `json:"readOnly"`
	Enabled      bool      `json:"enabled"`
	HostKey      string    `gorm:"size:128" json:"hostKey"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (CmdbAssetConnection) TableName() string { return "cmdb_asset_connection" }

// One-use tickets are stored hashed to support atomic redemption across API replicas.
type CmdbConnectionTicket struct {
	Hash         string `gorm:"size:64;primaryKey" json:"-"`
	UserID       uint
	ConnectionID uint
	Origin       string    `gorm:"size:512;not null;default:''" json:"-"`
	ExpiresAt    time.Time `gorm:"index"`
	ConsumedAt   *time.Time
}

func (CmdbConnectionTicket) TableName() string { return "cmdb_connection_ticket" }

type CmdbConnectionAudit struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	AssetType    string     `gorm:"size:16;index:idx_connection_audit_asset" json:"assetType"`
	AssetID      uint       `gorm:"index:idx_connection_audit_asset" json:"assetId"`
	ConnectionID uint       `json:"connectionId"`
	UserID       uint       `json:"userId"`
	Protocol     string     `gorm:"size:8" json:"protocol"`
	Action       string     `gorm:"size:24" json:"action"`
	Status       string     `gorm:"size:16" json:"status"`
	Message      string     `gorm:"size:255" json:"message"`
	ClientIP     string     `gorm:"size:64" json:"clientIp"`
	StartedAt    time.Time  `json:"startedAt"`
	EndedAt      *time.Time `json:"endedAt"`
}

func (CmdbConnectionAudit) TableName() string { return "cmdb_connection_audit" }
