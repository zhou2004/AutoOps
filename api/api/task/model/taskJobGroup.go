package model

import "time"

// TaskJobGroup organizes job definitions without changing their execution state.
type TaskJobGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TaskJobGroup) TableName() string { return "task_job_group" }

type TaskJobGroupSummary struct {
	TaskJobGroup
	TaskCount int64 `json:"task_count"`
}
