package model

import "time"

// Config stores models, diagnostic skills and workflows. Secrets never appear in JSON.
type Config struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Kind           string    `gorm:"size:20;index" json:"kind"`
	Name           string    `gorm:"size:100" json:"name"`
	Description    string    `gorm:"size:1000" json:"description"`
	Enabled        bool      `json:"enabled"`
	Provider       string    `gorm:"size:20" json:"provider"`
	BaseURL        string    `gorm:"size:1000" json:"baseUrl"`
	Model          string    `gorm:"size:200" json:"model"`
	Secret         string    `gorm:"type:text" json:"-"`
	Instructions   string    `gorm:"type:text" json:"instructions"`
	Modules        string    `gorm:"type:text" json:"-"`
	Steps          string    `gorm:"type:text" json:"-"`
	TimeoutSeconds int       `json:"timeoutSeconds"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (Config) TableName() string { return "aiops_config" }

type Step struct {
	Name         string `json:"name"`
	SkillID      uint   `json:"skillId"`
	Instructions string `json:"instructions"`
}
type ConfigDTO struct {
	Config
	Modules  []string `json:"modules"`
	Steps    []Step   `json:"steps"`
	APIKey   string   `json:"apiKey,omitempty"`
	ClearKey bool     `json:"clearKey,omitempty"`
	HasKey   bool     `json:"hasKey"`
}
type Run struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index" json:"-"`
	ModelID    uint       `json:"modelId"`
	SkillID    uint       `json:"skillId"`
	WorkflowID uint       `json:"workflowId"`
	Prompt     string     `gorm:"type:text" json:"prompt"`
	Modules    string     `gorm:"type:text" json:"modules"`
	Locale     string     `gorm:"size:10" json:"locale"`
	Status     string     `gorm:"size:20;index" json:"status"`
	Output     string     `gorm:"type:longtext" json:"output"`
	Evidence   string     `gorm:"type:longtext" json:"evidence"`
	Error      string     `gorm:"size:500" json:"error"`
	CreatedAt  time.Time  `gorm:"index" json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

func (Run) TableName() string { return "aiops_run" }

type DiagnoseRequest struct {
	ModelID    uint     `json:"modelId"`
	SkillID    uint     `json:"skillId"`
	WorkflowID uint     `json:"workflowId"`
	Prompt     string   `json:"prompt"`
	Modules    []string `json:"modules"`
	Days       int      `json:"days"`
	Locale     string   `json:"locale"`
}
type Event struct {
	Type   string      `json:"type"`
	RunID  uint        `json:"runId,omitempty"`
	Text   string      `json:"text,omitempty"`
	Step   string      `json:"step,omitempty"`
	Status string      `json:"status,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}
