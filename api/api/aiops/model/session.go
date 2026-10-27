package model

import "time"

type Session struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index" json:"-"`
	Title      string     `gorm:"size:160" json:"title"`
	ModelID    uint       `json:"modelId"`
	SkillID    uint       `json:"skillId"`
	Targets    string     `gorm:"type:text" json:"targets"`
	Actions    string     `gorm:"size:200" json:"actions"`
	GrantUntil *time.Time `json:"grantUntil"`
	BusyUntil  *time.Time `json:"busyUntil"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"index" json:"updatedAt"`
}

func (Session) TableName() string { return "aiops_session" }

type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID uint      `gorm:"index" json:"sessionId"`
	Role      string    `gorm:"size:16" json:"role"`
	Content   string    `gorm:"type:longtext" json:"content"`
	Status    string    `gorm:"size:20" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func (Message) TableName() string { return "aiops_message" }

type ToolCall struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	SessionID  uint       `gorm:"uniqueIndex:idx_ai_tool_request;index" json:"sessionId"`
	CallID     string     `gorm:"size:80;uniqueIndex:idx_ai_tool_request" json:"callId"`
	Tool       string     `gorm:"size:40" json:"tool"`
	Arguments  string     `gorm:"type:text" json:"arguments"`
	Result     string     `gorm:"type:longtext" json:"result"`
	Status     string     `gorm:"size:20" json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

func (ToolCall) TableName() string { return "aiops_tool_call" }

type SessionRequest struct {
	Title   string `json:"title"`
	ModelID uint   `json:"modelId"`
	SkillID uint   `json:"skillId"`
	Targets []uint `json:"targets"`
}
type GrantRequest struct {
	Actions []string `json:"actions"`
	Minutes int      `json:"minutes"`
}
type ToolRequest struct {
	CallID       string `json:"callId"`
	Tool         string `json:"tool"`
	ConnectionID uint   `json:"connectionId"`
	Container    string `json:"container"`
}

// Short-lived, per-turn bridge credentials; only a hash is persisted.
type HarnessTicket struct {
	Hash       string `gorm:"size:64;primaryKey" json:"-"`
	UserID     uint
	SessionID  uint
	APIKeyID   uint
	CanOperate bool
	ExpiresAt  time.Time `gorm:"index"`
}

func (HarnessTicket) TableName() string { return "aiops_harness_ticket" }
