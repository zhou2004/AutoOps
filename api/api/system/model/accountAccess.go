package model

import "time"

// AccountChallenge stores only a salted digest. Codes never enter API responses.
type AccountChallenge struct {
	ID        string `gorm:"primaryKey;size:64"`
	Email     string `gorm:"size:254;index"`
	Purpose   string `gorm:"size:16"`
	UserID    uint
	CodeHash  string `gorm:"size:64" json:"-"`
	Attempts  int
	Consumed  bool
	ExpiresAt time.Time `gorm:"index"`
}

func (AccountChallenge) TableName() string { return "sys_account_challenge" }

type AccountRateLimit struct {
	ID        string `gorm:"primaryKey;size:64"`
	Hits      int
	ExpiresAt time.Time `gorm:"index"`
}

func (AccountRateLimit) TableName() string { return "sys_account_rate_limit" }

// Reserves normalized identities across concurrent public registrations.
type AccountRegistration struct {
	Username string `gorm:"primaryKey;size:64"`
	Email    string `gorm:"uniqueIndex;size:254"`
	UserID   uint
}

func (AccountRegistration) TableName() string { return "sys_account_registration" }

// Password resets invalidate every prior browser session, including across replicas.
type AccountSessionVersion struct {
	UserID  uint `gorm:"primaryKey"`
	Version uint
}

func (AccountSessionVersion) TableName() string { return "sys_account_session_version" }
