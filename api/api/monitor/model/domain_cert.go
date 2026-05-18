package model

import "time"

// DomainCert 域名证书监控表
type DomainCert struct {
	ID            uint      `gorm:"primaryKey;column:id;autoIncrement;comment:主键ID" json:"id"`
	Domain        string    `gorm:"column:domain;type:varchar(255);not null;uniqueIndex:idx_domain;comment:域名" json:"domain"`
	Port          int       `gorm:"column:port;type:int;default:443;comment:端口" json:"port"`
	Issuer        string    `gorm:"column:issuer;type:varchar(512);comment:颁发者" json:"issuer"`
	Subject       string    `gorm:"column:subject;type:varchar(512);comment:主题" json:"subject"`
	NotBefore     string    `gorm:"column:not_before;type:varchar(64);comment:起始日期" json:"notBefore"`
	NotAfter      string    `gorm:"column:not_after;type:varchar(64);comment:到期日期" json:"notAfter"`
	RemainingDays int       `gorm:"column:remaining_days;type:int;default:-1;comment:剩余天数(-1=未知)" json:"remainingDays"`
	Status        int       `gorm:"column:status;type:tinyint(1);default:1;comment:状态:1-正常,2-即将过期(<=30天),3-已过期,4-检查失败" json:"status"`
	CheckTime     string    `gorm:"column:check_time;type:varchar(64);comment:最近检查时间" json:"checkTime"`
	ErrorMsg      string    `gorm:"column:error_msg;type:text;comment:错误信息" json:"errorMsg"`
	CreatedAt     time.Time `gorm:"column:create_time;type:datetime(3);not null;comment:创建时间" json:"createTime"`
	UpdatedAt     time.Time `gorm:"column:update_time;type:datetime(3);not null;comment:更新时间" json:"updateTime"`
}

func (DomainCert) TableName() string { return "monitor_domain_cert" }

type DomainCertListReq struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Domain   string `form:"domain" json:"domain"`
	Status   int    `form:"status" json:"status"`
}

type DomainCertAddReq struct {
	Domain string `json:"domain" binding:"required"`
	Port   int    `json:"port"`
}

type DomainCertUpdateReq struct {
	ID     uint   `json:"id" binding:"required"`
	Domain string `json:"domain" binding:"required"`
	Port   int    `json:"port"`
}

type DomainCertCheckResult struct {
	Domain        string `json:"domain"`
	Port          int    `json:"port"`
	Issuer        string `json:"issuer"`
	Subject       string `json:"subject"`
	NotBefore     string `json:"notBefore"`
	NotAfter      string `json:"notAfter"`
	RemainingDays int    `json:"remainingDays"`
	Status        int    `json:"status"`
	ErrorMsg      string `json:"errorMsg"`
}

// MonitorAPIEndpoint API端点监控表
type MonitorAPIEndpoint struct {
	ID              uint      `gorm:"primaryKey;column:id;autoIncrement;comment:主键ID" json:"id"`
	Name            string    `gorm:"column:name;type:varchar(255);not null;comment:名称" json:"name"`
	URL             string    `gorm:"column:url;type:varchar(1024);not null;comment:监控URL" json:"url"`
	Method          string    `gorm:"column:method;type:varchar(16);default:GET;comment:请求方法" json:"method"`
	Headers         string    `gorm:"column:headers;type:json;comment:请求头(JSON)" json:"headers"`
	Body            string    `gorm:"column:body;type:text;comment:请求体" json:"body"`
	CheckInterval   int       `gorm:"column:check_interval;type:int;default:300;comment:检查间隔(秒)" json:"checkInterval"`
	Timeout         int       `gorm:"column:timeout;type:int;default:10;comment:超时时间(秒)" json:"timeout"`
	ExpectedCode    int       `gorm:"column:expected_code;type:int;default:200;comment:期望HTTP状态码" json:"expectedCode"`
	ExpectedBody    string    `gorm:"column:expected_body;type:varchar(512);comment:期望响应体包含内容" json:"expectedBody"`
	LastStatusCode  int       `gorm:"column:last_status_code;type:int;default:0;comment:最后HTTP状态码" json:"lastStatusCode"`
	LastResponseTime int64    `gorm:"column:last_response_time;type:bigint;default:0;comment:最后响应时间(ms)" json:"lastResponseTime"`
	Status          int       `gorm:"column:status;type:tinyint(1);default:1;comment:状态:1-正常,2-异常,3-超时,4-检查失败" json:"status"`
	CheckTime       string    `gorm:"column:check_time;type:varchar(64);comment:最近检查时间" json:"checkTime"`
	ErrorMsg        string    `gorm:"column:error_msg;type:text;comment:错误信息" json:"errorMsg"`
	CreatedAt       time.Time `gorm:"column:create_time;type:datetime(3);not null;comment:创建时间" json:"createTime"`
	UpdatedAt       time.Time `gorm:"column:update_time;type:datetime(3);not null;comment:更新时间" json:"updateTime"`
}

func (MonitorAPIEndpoint) TableName() string { return "monitor_api_endpoint" }

type APIEndpointListReq struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Name     string `form:"name" json:"name"`
	Status   int    `form:"status" json:"status"`
}

type APIEndpointAddReq struct {
	Name          string `json:"name" binding:"required"`
	URL           string `json:"url" binding:"required"`
	Method        string `json:"method"`
	Headers       string `json:"headers"`
	Body          string `json:"body"`
	CheckInterval int    `json:"checkInterval"`
	Timeout       int    `json:"timeout"`
	ExpectedCode  int    `json:"expectedCode"`
	ExpectedBody  string `json:"expectedBody"`
}

type APIEndpointUpdateReq struct {
	ID            uint   `json:"id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	URL           string `json:"url" binding:"required"`
	Method        string `json:"method"`
	Headers       string `json:"headers"`
	Body          string `json:"body"`
	CheckInterval int    `json:"checkInterval"`
	Timeout       int    `json:"timeout"`
	ExpectedCode  int    `json:"expectedCode"`
	ExpectedBody  string `json:"expectedBody"`
}

// MonitorIncident 告警故障记录
type MonitorIncident struct {
	ID          uint      `gorm:"primaryKey;column:id;autoIncrement;comment:主键ID" json:"id"`
	Title       string    `gorm:"column:title;type:varchar(512);not null;comment:故障标题" json:"title"`
	Source      string    `gorm:"column:source;type:varchar(128);comment:来源(domain_cert/api_endpoint/prometheus)" json:"source"`
	SourceID    uint      `gorm:"column:source_id;type:int;default:0;comment:来源ID" json:"sourceId"`
	Level       string    `gorm:"column:level;type:varchar(32);default:warning;comment:告警等级:critical/warning/info" json:"level"`
	Status      string    `gorm:"column:status;type:varchar(32);default:firing;comment:状态:firing/resolved" json:"status"`
	Description string    `gorm:"column:description;type:text;comment:描述" json:"description"`
	AlertTime   string    `gorm:"column:alert_time;type:varchar(64);comment:告警时间" json:"alertTime"`
	ResolvedAt  string    `gorm:"column:resolved_at;type:varchar(64);comment:解决时间" json:"resolvedAt"`
	CreatedAt   time.Time `gorm:"column:create_time;type:datetime(3);not null;comment:创建时间" json:"createTime"`
	UpdatedAt   time.Time `gorm:"column:update_time;type:datetime(3);not null;comment:更新时间" json:"updateTime"`
}

func (MonitorIncident) TableName() string { return "monitor_incident" }

// IncidentStats 故障统计
type IncidentStats struct {
	TotalFiring  int64            `json:"totalFiring"`
	TotalResolved int64           `json:"totalResolved"`
	ByLevel      map[string]int64 `json:"byLevel"`
	BySource     map[string]int64 `json:"bySource"`
	Last24hCount int64            `json:"last24hCount"`
	TodayCount   int64            `json:"todayCount"`
}