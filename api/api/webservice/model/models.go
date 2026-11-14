package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Resource keeps stable asset identity separate from engine-specific configuration.
// Secret is never serialized. All writes are validated through typed domain specs.
type Resource struct {
	Connection  *ConnectionView `gorm:"-" json:"connection,omitempty"`
	ID          uint            `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"size:100;not null" json:"name"`
	GroupID     uint            `gorm:"index" json:"groupId"`
	ClusterID   uint            `gorm:"index" json:"clusterId"`
	SourceID    uint            `gorm:"index" json:"sourceId"`
	Engine      string          `gorm:"size:20;default:nginx" json:"engine"`
	Environment string          `gorm:"size:32" json:"environment"`
	Status      string          `gorm:"size:32;default:unknown" json:"status"`
	Management  string          `gorm:"size:16;default:observe" json:"management"`
	Revision    uint64          `gorm:"default:1" json:"revision"`
	LocatorHash *string         `gorm:"size:64;uniqueIndex" json:"-"`
	Spec        json.RawMessage `gorm:"type:longtext" json:"spec"`
	Observed    json.RawMessage `gorm:"type:longtext" json:"observed"`
	Secret      string          `gorm:"type:longtext" json:"-"`
	CreatedBy   uint            `json:"createdBy"`
	CollectedAt *time.Time      `json:"collectedAt"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ConnectionView is display-only CMDB metadata; credentials never enter responses.
type ConnectionView struct {
	HostID   uint   `json:"hostId"`
	HostName string `json:"hostName"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Alias    string `json:"alias"`
}

var Tables = map[string]string{
	"sources": "web_source", "instances": "web_instance", "clusters": "web_cluster", "sites": "web_site", "upstreams": "web_upstream", "configurations": "web_configuration", "config-groups": "web_config_group", "certificates": "web_certificate", "certificate-bindings": "web_certificate_binding", "acme-accounts": "web_acme_account", "dns-accounts": "web_dns_account", "renewal-policies": "web_renewal_policy",
}

// File is one Nginx configuration file. Group is an optional ConfigGroup name
// used to filter the file list; it never changes how the file is published.
type File struct {
	Deleted  bool   `json:"deleted,omitempty"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Hash     string `json:"hash"`
	Mode     uint32 `json:"mode"`
	Group    string `json:"group,omitempty"`
	Editable *bool  `json:"editable,omitempty"`
	Included *bool  `json:"included,omitempty"`
}
type SourceSpec struct {
	StatusURL      string   `json:"statusUrl"`
	HostID         uint     `json:"hostId"`
	ConnectionID   uint     `json:"connectionId"`
	Runtime        string   `json:"runtime"`
	Container      string   `json:"container"`
	Binary         string   `json:"binary"`
	ConfigPath     string   `json:"configPath"`
	Prefix         string   `json:"prefix"`
	Service        string   `json:"service"`
	ManagedPaths   []string `json:"managedPaths"`
	HealthURL      string   `json:"healthUrl"`
	HealthHost     string   `json:"healthHost"`
	ExpectedStatus int      `json:"expectedStatus"`
	Sudo           bool     `json:"sudo"`
	Image          string   `json:"image"`
	PackageVersion string   `json:"packageVersion"`
	Port           int      `json:"port"`
}
type Entry struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	Host           string `json:"host"`
	ExpectedStatus int    `json:"expectedStatus"`
}
type VIPSpec struct {
	CIDR                string `json:"cidr"`
	Interface           string `json:"interface"`
	RouterID            int    `json:"routerId"`
	InstanceIDs         []uint `json:"instanceIds"`
	PreferredInstanceID uint   `json:"preferredInstanceId,omitempty"`
}
type ClusterSpec struct {
	Entries []Entry  `json:"entries"`
	VIP     *VIPSpec `json:"vip,omitempty"`
}
type Location struct {
	Path      string            `json:"path"`
	Kind      string            `json:"kind"`
	Target    string            `json:"target"`
	Headers   map[string]string `json:"headers"`
	Websocket bool              `json:"websocket"`
	Timeout   int               `json:"timeout"`
}
type SiteSpec struct {
	HealthURL            string     `json:"healthUrl"`
	HealthHost           string     `json:"healthHost"`
	ExpectedStatus       int        `json:"expectedStatus"`
	Domains              []string   `json:"domains"`
	Ports                []int      `json:"ports"`
	TLS                  bool       `json:"tls"`
	CertificateID        uint       `json:"certificateId"`
	CertificatePath      string     `json:"certificatePath"`
	KeyPath              string     `json:"keyPath"`
	CertificateVersionID uint       `json:"certificateVersionId"`
	Mode                 string     `json:"mode"`
	Path                 string     `json:"path"`
	Raw                  string     `json:"raw"`
	Locations            []Location `json:"locations"`
	MaxBodySize          string     `json:"maxBodySize"`
	AccessLog            string     `json:"accessLog"`
	ErrorLog             string     `json:"errorLog"`
}
type ConfigSpec struct {
	InstanceIDs   []uint                     `json:"instanceIds,omitempty"`
	Certificates  map[uint]map[uint]uint     `json:"certificates,omitempty"`
	Nodes         map[uint][]File            `json:"nodes,omitempty"`
	Files         []File                     `json:"files"`
	Baselines     map[uint]string            `json:"baselines"`
	Variables     map[uint]map[string]string `json:"variables"`
	ConfigGroupID uint                       `json:"configGroupId"`
}

// ConfigGroupSpec is presentation metadata for filtering configuration drafts.
type ConfigGroupSpec struct {
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Paths       []string `json:"paths"`
}
type CertificateSpec struct {
	Domains          []string `json:"domains"`
	Owner            string   `json:"owner"`
	ACMEAccountID    uint     `json:"acmeAccountId"`
	DNSAccountID     uint     `json:"dnsAccountId"`
	Challenge        string   `json:"challenge"`
	InstanceIDs      []uint   `json:"instanceIds"`
	ChallengeRoot    string   `json:"challengeRoot"`
	CurrentVersionID uint     `json:"currentVersionId"`
}
type AccountSpec struct {
	Email        string          `json:"email"`
	DirectoryURL string          `json:"directoryUrl"`
	Provider     string          `json:"provider"`
	CredentialID uint            `json:"credentialId"`
	Registration json.RawMessage `json:"registration,omitempty"`
}
type BindingSpec struct {
	VerifyHost      string `json:"verifyHost"`
	VerifyPort      int    `json:"verifyPort"`
	ServerName      string `json:"serverName"`
	CertificateID   uint   `json:"certificateId"`
	VersionID       uint   `json:"versionId"`
	CertificatePath string `json:"certificatePath"`
	KeyPath         string `json:"keyPath"`
}
type RenewalSpec struct {
	APIKeyID           uint          `json:"apiKeyId,omitempty"`
	BindingIDs         []uint        `json:"bindingIds"`
	DeployOperationIDs map[uint]uint `json:"deployOperationIds,omitempty"`
	PendingOperationID uint          `json:"pendingOperationId,omitempty"`
	DeployedVersionID  uint          `json:"deployedVersionId,omitempty"`
	CertificateID      uint          `json:"certificateId"`
	Enabled            bool          `json:"enabled"`
	AutoDeploy         bool          `json:"autoDeploy"`
	NextRunAt          time.Time     `json:"nextRunAt"`
}
type Revision struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	ClusterID  uint            `gorm:"index" json:"clusterId"`
	ResourceID uint            `json:"resourceId"`
	Kind       string          `gorm:"size:24" json:"kind"`
	Name       string          `gorm:"size:100" json:"name"`
	Digest     string          `gorm:"size:64" json:"digest"`
	Content    json.RawMessage `gorm:"type:longtext" json:"content"`
	Secret     string          `gorm:"type:longtext" json:"-"`
	CreatedBy  uint            `json:"createdBy"`
	CreatedAt  time.Time       `json:"createdAt"`
}

func (Revision) TableName() string { return "web_revision" }

type Plan struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	UserID         uint            `json:"userId"`
	APIKeyID       uint            `json:"-"`
	TargetType     string          `gorm:"size:32" json:"targetType"`
	TargetID       uint            `json:"targetId"`
	Action         string          `gorm:"size:32" json:"action"`
	Revision       uint64          `json:"revision"`
	Parameters     json.RawMessage `gorm:"type:longtext" json:"parameters"`
	Checks         json.RawMessage `gorm:"type:text" json:"checks"`
	Snapshot       json.RawMessage `gorm:"type:longtext" json:"-"`
	Executable     bool            `json:"executable"`
	ExpiresAt      time.Time       `json:"expiresAt"`
	CreatedAt      time.Time       `json:"createdAt"`
	IdempotencyKey string          `gorm:"-" json:"idempotencyKey"`
}

func (Plan) TableName() string { return "web_operation_plan" }

type Operation struct {
	TargetName      string     `gorm:"-" json:"name,omitempty"`
	ID              uint       `gorm:"primaryKey" json:"id"`
	PlanID          uint       `gorm:"uniqueIndex" json:"planId"`
	UserID          uint       `json:"userId"`
	APIKeyID        uint       `json:"-"`
	ClusterID       uint       `gorm:"index" json:"clusterId"`
	TargetType      string     `gorm:"size:32" json:"targetType"`
	TargetID        uint       `json:"targetId"`
	Action          string     `gorm:"size:32" json:"action"`
	Status          string     `gorm:"size:32;index" json:"status"`
	IdempotencyKey  string     `gorm:"size:80;uniqueIndex" json:"idempotencyKey"`
	WorkerID        string     `gorm:"size:100" json:"workerId"`
	Message         string     `gorm:"size:1000" json:"message"`
	CancelRequested bool       `json:"cancelRequested"`
	HeartbeatAt     *time.Time `json:"heartbeatAt"`
	StartedAt       *time.Time `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

func (Operation) TableName() string { return "web_operation" }

type Step struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	OperationID uint            `gorm:"index" json:"operationId"`
	InstanceID  uint            `json:"instanceId"`
	Name        string          `gorm:"size:100" json:"name"`
	Status      string          `gorm:"size:32" json:"status"`
	Message     string          `gorm:"type:text" json:"message"`
	Evidence    json.RawMessage `gorm:"type:longtext" json:"evidence"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func (Step) TableName() string { return "web_operation_step" }

type Lock struct {
	Key         string `gorm:"size:100;primaryKey"`
	OperationID uint
	CreatedAt   time.Time
}

func (Lock) TableName() string { return "web_operation_lock" }

type Lease struct {
	Key       string `gorm:"size:100;primaryKey"`
	Owner     string `gorm:"size:100"`
	ExpiresAt time.Time
}

func (Lease) TableName() string { return "web_lease" }

type Ticket struct {
	APIKeyID   uint
	Hash       string `gorm:"size:64;primaryKey"`
	UserID     uint
	InstanceID uint
	Origin     string `gorm:"size:512"`
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

func (Ticket) TableName() string { return "web_terminal_ticket" }

type Event struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"userId"`
	ClusterID uint      `gorm:"index" json:"clusterId"`
	SourceID  uint      `json:"sourceId"`
	Kind      string    `gorm:"size:40" json:"kind"`
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `gorm:"index" json:"createdAt"`
}

func (Event) TableName() string { return "web_event" }

type Order struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CertificateID  uint      `gorm:"index" json:"certificateId"`
	IdentityDigest string    `gorm:"size:64" json:"-"`
	URL            string    `gorm:"size:512" json:"url"`
	Status         string    `gorm:"size:32" json:"status"`
	Secret         string    `gorm:"type:longtext" json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (Order) TableName() string { return "web_acme_order" }
func Migrate(db *gorm.DB) error {
	for _, table := range Tables {
		if err := db.Table(table).AutoMigrate(&Resource{}); err != nil {
			return err
		}
	}
	return db.AutoMigrate(&Revision{}, &Plan{}, &Operation{}, &Step{}, &Lock{}, &Lease{}, &Ticket{}, &Event{}, &Order{}, &TrafficState{}, &TrafficSample{})
}
