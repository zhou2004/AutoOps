package model

import "time"

// APIKey stores only the hash of a random bearer credential.
type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"userId"`
	IssuedBy   uint       `json:"issuedBy"`
	Name       string     `gorm:"size:100" json:"name"`
	Prefix     string     `gorm:"size:20" json:"prefix"`
	Hash       string     `gorm:"size:64;uniqueIndex" json:"-"`
	GrantMode  string     `gorm:"size:16;not null;default:''" json:"grantMode"`
	Scopes     string     `gorm:"type:text" json:"scopes"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (APIKey) TableName() string { return "sys_api_key" }
