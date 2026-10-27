package model

import "time"

const (
	TaskTriggerManual    = 1
	TaskTriggerScheduled = 2
)

// TaskTemplateSnapshot 保存一次执行实际使用的模板版本，用于审计和回滚。
type TaskTemplateSnapshot struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Type    int    `json:"type"`
	Content string `json:"content"`
	Remark  string `json:"remark"`
}

// TaskExecutionHistory 是任务作业的一次完整执行记录。
type TaskExecutionHistory struct {
	ID                uint               `gorm:"primaryKey" json:"id"`
	TaskID            uint               `gorm:"index;not null" json:"task_id"`
	TaskName          string             `gorm:"size:255" json:"task_name"`
	Trigger           int                `gorm:"not null" json:"trigger"`
	Operator          string             `gorm:"size:100" json:"operator"`
	Status            int                `gorm:"index;not null" json:"status"`
	ExecutionMode     int                `gorm:"not null" json:"execution_mode"`
	Command           string             `gorm:"type:longtext" json:"command"`
	CommandType       int                `json:"command_type"`
	MaxHistoryKeep    int                `json:"max_history_keep"`
	Shell             string             `gorm:"type:text" json:"shell"`
	HostIDs           string             `gorm:"type:text" json:"host_ids"`
	CronExpr          string             `gorm:"size:255" json:"cron_expr"`
	Remark            string             `gorm:"type:text" json:"remark"`
	TemplateSnapshots string             `gorm:"type:longtext" json:"template_snapshots"`
	StartedAt         time.Time          `gorm:"index" json:"started_at"`
	FinishedAt        *time.Time         `json:"finished_at"`
	Duration          int                `json:"duration"`
	RolledBackAt      *time.Time         `json:"rolled_back_at"`
	RolledBackBy      string             `gorm:"size:100" json:"rolled_back_by"`
	CreatedAt         time.Time          `json:"created_at"`
	Logs              []TaskExecutionLog `gorm:"foreignKey:HistoryID" json:"logs,omitempty"`
}

func (TaskExecutionHistory) TableName() string { return "task_execution_history" }

// TaskExecutionLog 保存一次执行中单个“主机 × 模板”的日志和命令快照。
type TaskExecutionLog struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	HistoryID       uint       `gorm:"index;not null" json:"history_id"`
	TaskID          uint       `gorm:"index;not null" json:"task_id"`
	TaskWorkID      uint       `gorm:"index;not null" json:"task_work_id"`
	TemplateID      uint       `gorm:"index" json:"template_id"`
	HostID          uint       `gorm:"index" json:"host_id"`
	HostName        string     `gorm:"size:255" json:"host_name"`
	TemplateName    string     `gorm:"size:255" json:"template_name"`
	CommandType     int        `json:"command_type"`
	CommandSnapshot string     `gorm:"type:longtext" json:"command_snapshot"`
	Status          int        `gorm:"index;not null" json:"status"`
	Log             string     `gorm:"type:longtext" json:"log"`
	LogPath         string     `gorm:"type:text" json:"log_path"`
	StartedAt       *time.Time `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at"`
	Duration        int        `json:"duration"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (TaskExecutionLog) TableName() string { return "task_execution_log" }
