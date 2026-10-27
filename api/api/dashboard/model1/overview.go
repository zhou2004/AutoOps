package model

import "time"

// Overview uses nil sections for unavailable data, never fabricated zeroes.
type Overview struct {
	GeneratedAt  time.Time          `json:"generatedAt"`
	Days         int                `json:"days"`
	Timezone     string             `json:"timezone"`
	StartDate    string             `json:"startDate"`
	EndDate      string             `json:"endDate"`
	AssetScope   string             `json:"assetScope"`
	Errors       map[string]string  `json:"errors"`
	Hosts        *Inventory         `json:"hosts"`
	Databases    *Inventory         `json:"databases"`
	Physical     *Inventory         `json:"physical"`
	Network      *Inventory         `json:"network"`
	Clusters     *Inventory         `json:"clusters"`
	Applications *Inventory         `json:"applications"`
	Deployments  *ExecutionOverview `json:"deployments"`
	Jobs         *ExecutionOverview `json:"jobs"`
	Ansible      *ExecutionOverview `json:"ansible"`
	Agents       *Inventory         `json:"agents"`
	Endpoints    *Inventory         `json:"endpoints"`
	Certificates *Inventory         `json:"certificates"`
	Alerts       *Inventory         `json:"alerts"`
	Incidents    *IncidentOverview  `json:"incidents"`
}

type CountItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type Inventory struct {
	Total        int64       `json:"total"`
	Statuses     []CountItem `json:"statuses"`
	Distribution []CountItem `json:"distribution"`
}

type ExecutionOverview struct {
	AllTimeTotal int64             `json:"allTimeTotal"`
	Total        int64             `json:"total"`
	Success      int64             `json:"success"`
	Failed       int64             `json:"failed"`
	Running      int64             `json:"running"`
	Pending      int64             `json:"pending"`
	Other        int64             `json:"other"`
	SuccessRate  *float64          `json:"successRate"`
	Trend        []ExecutionDay    `json:"trend"`
	Recent       []ExecutionRecord `json:"recent"`
}

type ExecutionDay struct {
	Date    string `json:"date"`
	Total   int64  `json:"total"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

type ExecutionRecord struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type IncidentOverview struct {
	Firing      int64            `json:"firing"`
	Critical    int64            `json:"critical"`
	Warning     int64            `json:"warning"`
	Info        int64            `json:"info"`
	Levels      []CountItem      `json:"levels"`
	PeriodTotal int64            `json:"periodTotal"`
	Trend       []CountItem      `json:"trend"`
	Recent      []IncidentRecord `json:"recent"`
}

type IncidentRecord struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
}
