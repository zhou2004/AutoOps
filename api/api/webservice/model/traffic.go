package model

import (
	"encoding/json"
	"time"
)

// TrafficState advances together with its sample; failed inserts never consume logs.
type TrafficState struct {
	InstanceID     uint `gorm:"primaryKey"`
	SourceRevision uint64
	Identity       string `gorm:"size:128"`
	CollectedAt    time.Time
	Metrics        json.RawMessage `gorm:"type:longtext"`
	Cursors        json.RawMessage `gorm:"type:longtext"`
	Revision       uint64
}

func (TrafficState) TableName() string { return "web_traffic_state" }

type TrafficSample struct {
	ID         uint      `gorm:"primaryKey"`
	InstanceID uint      `gorm:"index:idx_web_traffic_instance_at,priority:1"`
	At         time.Time `gorm:"index;index:idx_web_traffic_instance_at,priority:2"`
	Seconds    float64
	Metrics    json.RawMessage `gorm:"type:longtext"`
	Logs       json.RawMessage `gorm:"type:longtext"`
	Messages   json.RawMessage `gorm:"type:text"`
}

func (TrafficSample) TableName() string { return "web_traffic_sample" }
