package model

import (
	"encoding/json"
	"time"
)

// Cluster is an operational boundary, independent of connection assets and installation runs.
type Cluster struct {
	VIP                      *VirtualIP        `gorm:"-" json:"vip,omitempty"`
	Replication              []ReplicationLink `gorm:"-" json:"replication"`
	LatestBackup             *BackupSummary    `gorm:"-" json:"latestBackup,omitempty"`
	CurrentOperation         *OperationSummary `gorm:"-" json:"currentOperation,omitempty"`
	PlacementWarnings        []string          `gorm:"-" json:"placementWarnings,omitempty"`
	CompatibilityReason      string            `gorm:"-" json:"compatibilityReason,omitempty"`
	ConfigurationStatus      string            `gorm:"size:32" json:"configurationStatus"`
	ConfigurationCollectedAt *time.Time        `json:"configurationCollectedAt"`
	ID                       uint              `gorm:"primaryKey" json:"id"`
	Name                     string            `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Engine                   string            `gorm:"size:20;index;not null" json:"engine"`
	Topology                 string            `gorm:"size:32;not null" json:"topology"`
	GroupID                  uint              `gorm:"index;not null" json:"groupId"`
	Environment              string            `gorm:"size:32;index" json:"environment"`
	Management               string            `gorm:"size:20;default:observe" json:"management"`
	Status                   string            `gorm:"size:24;default:unknown;index" json:"status"`
	DeploymentID             *uint             `gorm:"uniqueIndex" json:"deploymentId"`
	Revision                 uint64            `gorm:"not null;default:1" json:"revision"`
	TopologyHash             string            `gorm:"size:64" json:"topologyHash"`
	Policy                   json.RawMessage   `gorm:"type:text" json:"policy"`
	Health                   json.RawMessage   `gorm:"type:text" json:"health"`
	Credentials              string            `gorm:"type:text" json:"-"`
	CollectedAt              *time.Time        `json:"collectedAt"`
	HAValidatedAt            *time.Time        `json:"haValidatedAt"`
	RestoreVerifiedAt        *time.Time        `json:"restoreVerifiedAt"`
	CreatedBy                uint              `json:"createdBy"`
	CreatedAt                time.Time         `json:"createdAt"`
	UpdatedAt                time.Time         `json:"updatedAt"`
	Members                  []Member          `gorm:"foreignKey:ClusterID" json:"members,omitempty"`
	Endpoints                []Endpoint        `gorm:"foreignKey:ClusterID" json:"endpoints,omitempty"`
}

func (Cluster) TableName() string { return "database_cluster" }

type Member struct {
	BaselinePending bool            `json:"baselinePending"`
	DataMemberID    uint            `gorm:"index" json:"dataMemberId"`
	TLSMode         string          `gorm:"size:20" json:"tlsMode"`
	ID              uint            `gorm:"primaryKey" json:"id"`
	ClusterID       uint            `gorm:"index;not null" json:"clusterId"`
	HostID          uint            `gorm:"index" json:"hostId"`
	SourceID        uint            `gorm:"index" json:"sourceId"`
	AccountID       uint            `json:"accountId"`
	Name            string          `gorm:"size:100" json:"name"`
	Kind            string          `gorm:"size:24" json:"kind"`
	Address         string          `gorm:"size:253;uniqueIndex:database_member_address" json:"address"`
	Port            int             `gorm:"uniqueIndex:database_member_address" json:"port"`
	Role            string          `gorm:"size:32" json:"role"`
	NativeID        string          `gorm:"size:128" json:"nativeId"`
	ParentID        string          `gorm:"size:128" json:"parentId"`
	Shard           string          `gorm:"size:64" json:"shard"`
	Version         string          `gorm:"size:64" json:"version"`
	DataDir         string          `gorm:"size:255" json:"dataDir"`
	Service         string          `gorm:"size:100" json:"service"`
	Managed         bool            `json:"managed"`
	State           string          `gorm:"size:24" json:"state"`
	Metrics         json.RawMessage `gorm:"type:longtext" json:"metrics"`
	CollectedAt     *time.Time      `json:"collectedAt"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func (Member) TableName() string { return "database_member" }

type Endpoint struct {
	ID                uint            `gorm:"primaryKey" json:"id"`
	ClusterID         uint            `gorm:"index" json:"clusterId"`
	Name              string          `gorm:"size:100" json:"name"`
	Mode              string          `gorm:"size:24" json:"mode"`
	Addresses         json.RawMessage `gorm:"type:text" json:"addresses"`
	MasterName        string          `gorm:"size:100" json:"masterName"`
	AccountID         uint            `json:"accountId"`
	SentinelAccountID uint            `json:"sentinelAccountId"`
	TLSMode           string          `gorm:"size:20" json:"tlsMode"`
	Status            string          `gorm:"size:24" json:"status"`
	VerifiedAt        *time.Time      `json:"verifiedAt"`
}

func (Endpoint) TableName() string { return "database_endpoint" }

type OperationPlan struct {
	IdempotencyKey string          `gorm:"-" json:"idempotencyKey"`
	ID             uint            `gorm:"primaryKey" json:"id"`
	ClusterID      uint            `gorm:"index" json:"clusterId"`
	Revision       uint64          `json:"revision"`
	Action         string          `gorm:"size:40" json:"action"`
	Parameters     json.RawMessage `gorm:"type:text" json:"parameters"`
	Checks         json.RawMessage `gorm:"type:text" json:"checks"`
	Executable     bool            `json:"executable"`
	UserID         uint            `json:"userId"`
	CreatedAt      time.Time       `json:"createdAt"`
	ExpiresAt      time.Time       `json:"expiresAt"`
}

func (OperationPlan) TableName() string { return "database_operation_plan" }

type Operation struct {
	AuditDeletedAt *time.Time      `gorm:"index" json:"auditDeletedAt,omitempty"`
	APIKeyID       uint            `json:"apiKeyId"`
	ID             uint            `gorm:"primaryKey" json:"id"`
	ClusterID      uint            `gorm:"index;uniqueIndex:database_operation_idempotency" json:"clusterId"`
	PlanID         uint            `json:"planId"`
	Action         string          `gorm:"size:40" json:"action"`
	Parameters     json.RawMessage `gorm:"type:text" json:"parameters"`
	IdempotencyKey string          `gorm:"size:80;uniqueIndex:database_operation_idempotency" json:"idempotencyKey"`
	RequestHash    string          `gorm:"size:64" json:"-"`
	Status         string          `gorm:"size:24;index" json:"status"`
	UserID         uint            `json:"userId"`
	WorkerID       string          `gorm:"size:64" json:"workerId"`
	HeartbeatAt    *time.Time      `json:"heartbeatAt"`
	Message        string          `gorm:"size:1000" json:"message"`
	CreatedAt      time.Time       `json:"createdAt"`
	StartedAt      *time.Time      `json:"startedAt"`
	FinishedAt     *time.Time      `json:"finishedAt"`
	Steps          []OperationStep `gorm:"foreignKey:OperationID" json:"steps,omitempty"`
}

func (Operation) TableName() string { return "database_operation" }

type OperationStep struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	OperationID uint       `gorm:"index" json:"operationId"`
	Name        string     `gorm:"size:100" json:"name"`
	Status      string     `gorm:"size:24" json:"status"`
	Log         string     `gorm:"type:longtext" json:"log"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt"`
}

func (OperationStep) TableName() string { return "database_operation_step" }

type OperationLock struct {
	ClusterID   uint      `gorm:"primaryKey" json:"clusterId"`
	OperationID uint      `json:"operationId"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (OperationLock) TableName() string { return "database_operation_lock" }

type StorageSnapshot struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	ClusterID      uint            `gorm:"index" json:"clusterId"`
	MemberID       uint            `gorm:"index" json:"memberId"`
	HostID         uint            `json:"hostId"`
	Kind           string          `gorm:"size:24" json:"kind"`
	Name           string          `gorm:"size:255" json:"name"`
	FilesystemID   string          `gorm:"size:255" json:"filesystemId"`
	TotalBytes     uint64          `json:"totalBytes"`
	UsedBytes      uint64          `json:"usedBytes"`
	AvailableBytes uint64          `json:"availableBytes"`
	Details        json.RawMessage `gorm:"type:text" json:"details"`
	CollectedAt    time.Time       `gorm:"index" json:"collectedAt"`
}

func (StorageSnapshot) TableName() string { return "database_storage_snapshot" }

type Repository struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"size:100;uniqueIndex" json:"name"`
	GroupID   uint       `gorm:"index" json:"groupId"`
	Kind      string     `gorm:"size:16" json:"kind"`
	HostID    uint       `json:"hostId"`
	Path      string     `gorm:"size:255" json:"path"`
	Endpoint  string     `gorm:"size:255" json:"endpoint"`
	Bucket    string     `gorm:"size:100" json:"bucket"`
	Region    string     `gorm:"size:64" json:"region"`
	KeyID     uint       `json:"keyId"`
	Status    string     `gorm:"size:24" json:"status"`
	CheckedAt *time.Time `json:"checkedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (Repository) TableName() string { return "database_repository" }

type BackupPolicy struct {
	APIKeyID      uint       `json:"-"`
	ID            uint       `gorm:"primaryKey" json:"id"`
	ClusterID     uint       `gorm:"index" json:"clusterId"`
	RepositoryID  uint       `json:"repositoryId"`
	Name          string     `gorm:"size:100" json:"name"`
	Method        string     `gorm:"size:24" json:"method"`
	Cron          string     `gorm:"size:100" json:"cron"`
	RetentionDays int        `json:"retentionDays"`
	Enabled       bool       `json:"enabled"`
	CreatedBy     uint       `json:"createdBy"`
	NextRunAt     *time.Time `gorm:"index" json:"nextRunAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func (BackupPolicy) TableName() string { return "database_backup_policy" }

type Backup struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	ClusterID    uint            `gorm:"index" json:"clusterId"`
	OperationID  uint            `json:"operationId"`
	RepositoryID uint            `json:"repositoryId"`
	ParentID     uint            `json:"parentId"`
	Method       string          `gorm:"size:24" json:"method"`
	Status       string          `gorm:"size:24" json:"status"`
	Manifest     json.RawMessage `gorm:"type:longtext" json:"manifest"`
	Credentials  string          `gorm:"type:longtext" json:"-"`
	LegacyRunID  *uint           `gorm:"uniqueIndex" json:"legacyRunId"`
	VerifiedAt   *time.Time      `json:"verifiedAt"`
	CreatedAt    time.Time       `json:"createdAt"`
}

func (Backup) TableName() string { return "database_backup" }

type ConfigRevision struct {
	MemberID    uint            `json:"memberId"`
	RuntimeOnly bool            `json:"runtimeOnly"`
	ID          uint            `gorm:"primaryKey" json:"id"`
	ClusterID   uint            `gorm:"index" json:"clusterId"`
	OperationID uint            `json:"operationId"`
	Previous    json.RawMessage `gorm:"type:text" json:"previous"`
	Desired     json.RawMessage `gorm:"type:text" json:"desired"`
	ApplyMode   string          `gorm:"size:20" json:"applyMode"`
	Status      string          `gorm:"size:24" json:"status"`
	UserID      uint            `json:"userId"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func (ConfigRevision) TableName() string { return "database_config_revision" }

type Event struct {
	AuditDeletedAt *time.Time      `gorm:"index" json:"auditDeletedAt,omitempty"`
	ExternalKey    *string         `gorm:"size:160;uniqueIndex:database_native_event" json:"externalKey,omitempty"`
	Details        json.RawMessage `gorm:"type:text" json:"details,omitempty"`
	ID             uint            `gorm:"primaryKey" json:"id"`
	ClusterID      uint            `gorm:"index;uniqueIndex:database_native_event" json:"clusterId"`
	MemberID       uint            `json:"memberId"`
	OperationID    uint            `json:"operationId"`
	UserID         uint            `json:"userId"`
	Kind           string          `gorm:"size:40" json:"kind"`
	Severity       string          `gorm:"size:16" json:"severity"`
	Message        string          `gorm:"size:1000" json:"message"`
	CreatedAt      time.Time       `gorm:"index" json:"createdAt"`
}

func (Event) TableName() string { return "database_event" }

func Models() []any {
	return []any{&Cluster{}, &Member{}, &Endpoint{}, &OperationPlan{}, &Operation{}, &OperationStep{}, &OperationLock{}, &StorageSnapshot{}, &StorageThreshold{}, &Repository{}, &BackupPolicy{}, &Backup{}, &ConfigRevision{}, &Event{}, &CollectionLease{}, &AuditPolicy{}, &VirtualIP{}}
}

// StorageThreshold overrides capacity limits for a business group and children.
type StorageThreshold struct {
	GroupID         uint      `gorm:"primaryKey;autoIncrement:false" json:"groupId"`
	WarningPercent  int       `json:"warningPercent"`
	CriticalPercent int       `json:"criticalPercent"`
	UpdatedBy       uint      `json:"updatedBy"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (StorageThreshold) TableName() string { return "database_storage_threshold" }

// List summaries deliberately omit credentials, artifact manifests and step logs.
type BackupSummary struct {
	ID         uint       `json:"id"`
	ClusterID  uint       `json:"clusterId"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	VerifiedAt *time.Time `json:"verifiedAt"`
}
type OperationSummary struct {
	ID        uint   `json:"id"`
	ClusterID uint   `json:"clusterId"`
	Action    string `json:"action"`
	Status    string `json:"status"`
}

// CollectionLease prevents API/worker replicas from polling the same cluster together.
type CollectionLease struct {
	ClusterID uint      `gorm:"primaryKey;autoIncrement:false"`
	Owner     string    `gorm:"size:64"`
	Until     time.Time `gorm:"index"`
}

func (CollectionLease) TableName() string { return "database_collection_lease" }

type AuditPolicy struct {
	ID            uint      `gorm:"primaryKey;autoIncrement:false" json:"id"`
	RetentionDays int       `json:"retentionDays"`
	UpdatedBy     uint      `json:"updatedBy"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (AuditPolicy) TableName() string { return "database_audit_policy" }

type VirtualIP struct {
	Address       *string         `gorm:"size:45;uniqueIndex" json:"address,omitempty"`
	ClusterID     uint            `gorm:"primaryKey;autoIncrement:false" json:"clusterId"`
	CIDR          string          `gorm:"size:64;uniqueIndex" json:"cidr"`
	Interface     string          `gorm:"size:32" json:"interface"`
	RouterID      int             `gorm:"uniqueIndex" json:"routerId"`
	MemberIDs     json.RawMessage `gorm:"type:text" json:"memberIds"`
	Status        string          `gorm:"size:24" json:"status"`
	OwnerMemberID uint            `json:"ownerMemberId"`
	Observation   json.RawMessage `gorm:"type:text" json:"observation"`
	CollectedAt   *time.Time      `json:"collectedAt"`
	VerifiedAt    *time.Time      `json:"verifiedAt"`
	LastDrill     json.RawMessage `gorm:"type:text" json:"lastDrill"`
}

func (VirtualIP) TableName() string { return "database_virtual_ip" }

type ReplicationLink struct {
	SourceID    uint           `json:"sourceId"`
	TargetID    uint           `json:"targetId"`
	State       string         `json:"state"`
	Message     string         `json:"message"`
	Evidence    map[string]any `json:"evidence"`
	CollectedAt *time.Time     `json:"collectedAt"`
}
