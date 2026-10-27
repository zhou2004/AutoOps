package model

import "time"

// KubeaszOperation is an auditable system Ansible execution. Credentials are
// supplied by the deployment environment, never by the request or this record.
type KubeaszOperation struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	Cluster       string     `json:"cluster" gorm:"size:63;index;not null"`
	Action        string     `json:"action" gorm:"size:40;not null"`
	Parameters    string     `json:"parameters" gorm:"type:text"`
	Command       string     `json:"command" gorm:"type:text"`
	Version       string     `json:"version" gorm:"size:20"`
	Status        string     `json:"status" gorm:"size:20;index;not null"`
	OperatorID    uint       `json:"operator_id"`
	OperatorName  string     `json:"operator_name" gorm:"size:100"`
	Error         string     `json:"error" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	ReconciledAt  *time.Time `json:"reconciled_at"`
	ReconciledBy  string     `json:"reconciled_by" gorm:"size:100"`
	ReconcileNote string     `json:"reconcile_note" gorm:"type:text"`
}

func (KubeaszOperation) TableName() string { return "task_kubeasz_operation" }
